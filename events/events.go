package events

import (
	"bufio"
	"errors"
	"io"
	"log"
	"net"
	"regexp"
	"strconv"

	"bitbucket.org/johananl/follower-maze/userclients"
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
	userHandler  *userclients.UserHandler
}

func (eh EventHandler) AcceptEvents(l net.Listener) {
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

		event, err := eh.ParseEvent(message)
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

// ParseEvent gets a string and returns an Event or an error if it cannot parse.
func (eh EventHandler) ParseEvent(e string) (*Event, error) {
	//log.Printf("Parsing event %s", e)
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
		return nil, errors.New("Invalid event: " + e)
	}

	return result, nil
}

func (eh EventHandler) processEvent(e *Event) {
	switch e.eventType {
	case follow:
		//log.Println("Processing Follow event")
		eh.userHandler.Follow(e.fromUserId, e.toUserId)
		eh.userHandler.NotifyUser(e.toUserId, e.rawEvent)
	case unfollow:
		//log.Println("Processing Unfollow event")
		eh.userHandler.Unfollow(e.fromUserId, e.toUserId)
	case broadcast:
		//log.Println("Processing broadcast event")
		// Notify all users
		// Block only "sender" object until end of broadcast processing (block getting next event from queue)
		for u := range eh.userHandler.Users {
			eh.userHandler.NotifyUser(u, e.rawEvent)
		}
	case privateMsg:
		//log.Println("Processing Private Msg event")
		eh.userHandler.NotifyUser(e.toUserId, e.rawEvent)
	case statusUpdate:
		//log.Println("Processing Status Update event")
		for _, u := range eh.userHandler.Followers(e.fromUserId) {
			eh.userHandler.NotifyUser(u, e.rawEvent)
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

func NewEventHandler(qm *QueueManager, uh *userclients.UserHandler) *EventHandler {
	return &EventHandler{qm, uh}
}
