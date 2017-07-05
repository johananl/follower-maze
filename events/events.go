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

// Valid event types.
const (
	follow       string = "F"
	unfollow     string = "U"
	broadcast    string = "B"
	privateMsg   string = "P"
	statusUpdate string = "S"
)

// Event represents an event received from the event source. Events are handled by an EventHandler.
// The rawEvent field is used to store the original event (after parsing) as received from the TCP
// connection. This is done to avoid having to reconstruct the raw event before sending it to user
// clients.
type Event struct {
	rawEvent   string
	sequence   int
	eventType  string
	fromUserId int
	toUserId   int
	index      int // Used for ordering in a priority queue
}

// EventHandler handles events. It saves them in a priority queue for ordering and communicates
// with a UserHandler for user-related operations.
type EventHandler struct {
	queueManager *QueueManager
	userHandler  *userclients.UserHandler
}

// AcceptEvents accepts TCP connections from event sources and triggers handling of messages over
// these connections.
func (eh EventHandler) AcceptEvents(l net.Listener) {
	// Continually accept event connections. This loop iterates every time a new connection from an
	// event source is received and blocks at Accept().
	for {
		c, err := l.Accept()
		if err != nil {
			log.Println("Error accepting:", err.Error())
			continue
		}

		// We could actually block here since we're handling only one event source, however
		// executing handleEvents() as a separate goroutine provides automatic support for multiple
		// event sources concurrently, should this become a requirement in the future.
		go eh.handleEvents(c)
	}
}

// Reads a stream of events from a TCP connection and stores them in a priority queue.
func (eh EventHandler) handleEvents(conn net.Conn) {
	// A counter for the total number of valid events received from the connection.
	totalReceived := 0

	// Close connection when done reading.
	defer func() {
		log.Println("Closing event connection")
		log.Println("Total events received:", totalReceived)

		// Send any events left in the queue after we stopped receiving events.
		log.Println("Flushing queue")
		eh.flushQueue(eh.queueManager)
		conn.Close()
	}()

	br := bufio.NewReader(conn)
	// Continually read from connection. This loop iterates every time a newline-delimited string
	// is read from the TCP connection. The loop blocks at ReadString().
	for {
		message, err := br.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				log.Println("Got EOF on event connection")
				break // No more events - stop reading.
			}
			log.Println("Error reading event:", err.Error())
			continue // Skip this event and move to the next one.
		}

		event, err := eh.ParseEvent(message)
		if err != nil {
			log.Println("Event parsing failed:", err)
			continue // Skip this event and move to the next one.
		}

		// Event looks good. Count it and put it in the queue.
		totalReceived++
		eh.queueManager.queueEvent(event)

		// If we have enough events in the queue, process the top event.
		if eh.queueManager.queue.Len() > eventQueueSize {
			eh.processEvent(eh.queueManager.popEvent())
		}
	}
}

// ParseEvent gets a string and returns an Event or an error.
func (eh EventHandler) ParseEvent(e string) (*Event, error) {
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

// Processes the received event. Depending on the event's type, processing may include registering
// a Follow or Unfollow event and sending the event to one or more user clients.
func (eh EventHandler) processEvent(e *Event) {
	switch e.eventType {
	case follow:
		// Register fromUserId as a follower of toUserId and notify toUserId.
		eh.userHandler.Follow(e.fromUserId, e.toUserId)
		eh.userHandler.NotifyUser(e.toUserId, e.rawEvent)
	case unfollow:
		// Remove fromUserId from toUserId's followers.
		eh.userHandler.Unfollow(e.fromUserId, e.toUserId)
	case broadcast:
		// Notify all connected users.
		for u := range eh.userHandler.Users {
			eh.userHandler.NotifyUser(u, e.rawEvent)
		}
	case privateMsg:
		// Notify toUserId.
		eh.userHandler.NotifyUser(e.toUserId, e.rawEvent)
	case statusUpdate:
		// Notify all followers of fromUserId.
		for _, u := range eh.userHandler.Followers(e.fromUserId) {
			eh.userHandler.NotifyUser(u, e.rawEvent)
		}
	default:
		// This is just for safety since all received events should have been parsed successfully.
		log.Println("Invalid event type - ignoring")
		return
	}
}

// Empties the queue by processing all remaining messages. This method is called once the event
// source connection has been closed.
func (eh EventHandler) flushQueue(qm *QueueManager) {
	for qm.queue.Len() > 0 {
		eh.processEvent(qm.popEvent())
	}
}

// NewEventHandler constructs a new EventHandler. It receives a pointer to a QueueManager as well
// as a pointer to a UserHandler.
func NewEventHandler(qm *QueueManager, uh *userclients.UserHandler) *EventHandler {
	return &EventHandler{qm, uh}
}
