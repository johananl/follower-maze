package main

import (
	"bufio"
	"errors"
	"io"
	"log"
	"net"
	"regexp"
	"strconv"
)

const (
	follow       string = "F"
	unfollow     string = "U"
	broadcast    string = "B"
	privateMsg   string = "P"
	statusUpdate string = "S"
)

type Event struct {
	rawEvent   string
	sequence   int
	eventType  string
	fromUserId int
	toUserId   int
	index      int // Used for ordering in a priority queue
}

type EventHandler struct {
	queueManager *QueueManager
	userHandler  *UserHandler
}

func (eh EventHandler) acceptEvents(l net.Listener) {
	// Continually accept event connections
	// This loop iterates every time a new events connection is made.
	for {
		c, err := l.Accept()
		if err != nil {
			log.Println("Error accepting:", err.Error())
			continue
		}

		// We could actually block here since we're handling only one
		// event source, however executing handleEvents() as a
		// separate goroutine provides automatic support for multiple
		// event sources concurrently, should this become a requirement
		// in the future.
		go eh.handleEvents(c)
	}
}

func (eh EventHandler) handleEvents(conn net.Conn) {
	// Total valid events received from the connection
	totalReceived := 0

	// Close connection when done reading
	defer func() {
		log.Println("Closing event connection")
		log.Println("Total events received:", totalReceived)
		log.Println("Flushing queue")
		eh.flushQueue(eh.queueManager)
		conn.Close()
	}()

	// Continually read from connection
	br := bufio.NewReader(conn)
	// This loop iterates every time a newline-delimited string is read from
	// the TCP connection.
	for {
		message, err := br.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				log.Println("Got EOF on event connection")
				break
			}
			log.Println("Error reading event:", err.Error())
			continue
		}

		event, err := eh.parseEvent(message)
		if err != nil {
			log.Println("Event parsing failed:", err)
			continue
		}

		//log.Println("Received event:", strings.TrimSpace(message))
		totalReceived++

		eh.queueManager.queueEvent(event)
		// If we have enough events in the queue, process the first event.
		if eh.queueManager.queue.Len() > eventQueueSize {
			//log.Println("Got enough events in the queue - processing")
			eh.processEvent(eh.queueManager.popEvent())
		}
	}
}

// parseEvent gets a string and returns an Event or an error if it cannot parse.
func (eh EventHandler) parseEvent(e string) (*Event, error) {
	//log.Printf("Parsing event %s", e)

	// TODO Get rid of regex matching and handle string manually?
	fPattern := regexp.MustCompile(`^(\d+)\|F\|(\d+)\|(\d+)\n$`)
	uPattern := regexp.MustCompile(`^(\d+)\|U\|(\d+)\|(\d+)\n$`)
	bPattern := regexp.MustCompile(`^(\d+)\|B\n$`)
	pPattern := regexp.MustCompile(`^(\d+)\|P\|(\d+)\|(\d+)\n$`)
	sPattern := regexp.MustCompile(`^(\d+)\|S\|(\d+)\n$`)

	var result *Event

	if m := fPattern.FindStringSubmatch(e); len(m) != 0 {
		seq, _ := strconv.Atoi(m[1])
		fuid, _ := strconv.Atoi(m[2])
		tuid, _ := strconv.Atoi(m[3])

		result = &Event{
			rawEvent:   e,
			sequence:   seq,
			eventType:  follow,
			fromUserId: fuid,
			toUserId:   tuid,
		}
	} else if m := uPattern.FindStringSubmatch(e); len(m) != 0 {
		seq, _ := strconv.Atoi(m[1])
		fuid, _ := strconv.Atoi(m[2])
		tuid, _ := strconv.Atoi(m[3])
		result = &Event{
			rawEvent:   e,
			sequence:   seq,
			eventType:  unfollow,
			fromUserId: fuid,
			toUserId:   tuid,
		}
	} else if m := bPattern.FindStringSubmatch(e); len(m) != 0 {
		seq, _ := strconv.Atoi(m[1])
		result = &Event{
			rawEvent:  e,
			sequence:  seq,
			eventType: broadcast,
		}
	} else if m := pPattern.FindStringSubmatch(e); len(m) != 0 {
		seq, _ := strconv.Atoi(m[1])
		fuid, _ := strconv.Atoi(m[2])
		tuid, _ := strconv.Atoi(m[3])
		result = &Event{
			rawEvent:   e,
			sequence:   seq,
			eventType:  privateMsg,
			fromUserId: fuid,
			toUserId:   tuid,
		}
	} else if m := sPattern.FindStringSubmatch(e); len(m) != 0 {
		seq, _ := strconv.Atoi(m[1])
		fuid, _ := strconv.Atoi(m[2])
		result = &Event{
			rawEvent:   e,
			sequence:   seq,
			eventType:  statusUpdate,
			fromUserId: fuid,
		}
	} else {
		return nil, errors.New("Invalid message: " + e)
	}

	return result, nil
}

func (eh EventHandler) processEvent(e *Event) {
	switch e.eventType {
	case "F":
		//log.Println("Processing Follow event")
		eh.userHandler.follow(e.fromUserId, e.toUserId)
		eh.userHandler.notifyUser(e.toUserId, e.rawEvent)
	case "U":
		//log.Println("Processing Unfollow event")
		eh.userHandler.unfollow(e.fromUserId, e.toUserId)
	case "B":
		//log.Println("Processing broadcast event")
		// Notify all users
		// Block only "sender" object until end of broadcast processing (block getting next event from queue)
		// TODO Do we need the blank identifier here?
		for u, _ := range eh.userHandler.users {
			eh.userHandler.notifyUser(u, e.rawEvent)
		}
	case "P":
		//log.Println("Processing Private Msg event")
		eh.userHandler.notifyUser(e.toUserId, e.rawEvent)
	case "S":
		//log.Println("Processing Status Update event")
		eh.userHandler.lock.RLock()
		defer eh.userHandler.lock.RUnlock()
		for _, u := range eh.userHandler.followers[e.fromUserId] {
			eh.userHandler.notifyUser(u, e.rawEvent)
		}
	default:
		log.Println("Invalid event type - ignoring")
		return
	}
}

// flushQueue empties the queue by processing all remaining messages.
// This method is called once the event source connection is closed.
// TODO Flush after a timeout? Dead timeout at event source?
func (eh EventHandler) flushQueue(qm *QueueManager) {
	for qm.queue.Len() > 0 {
		eh.processEvent(qm.popEvent())
	}
}

func NewEventHandler(qm *QueueManager, uh *UserHandler) *EventHandler {
	return &EventHandler{qm, uh}
}
