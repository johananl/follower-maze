package events

import (
	"bufio"
	"errors"
	"io"
	"log"
	"net"
	"os"
	"regexp"
	"strconv"

	"bitbucket.org/johananl/follower-maze/userclients"
)

// Server config
const (
	host = "localhost"
	port = "9090"
)

// Valid event types
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
// clients, which is relatively expensive.
type Event struct {
	rawEvent   string
	sequence   int
	eventType  string
	fromUserID int
	toUserID   int
	index      int // Used for ordering in a priority queue
}

// EventHandler handles events. It saves them in a priority queue for ordering and communicates
// with a UserHandler for user-related operations.
type EventHandler struct {
	queueManager *QueueManager
	userHandler  *userclients.UserHandler
}

// AcceptConnections accepts TCP connections from event sources and sends back net.Conn structs.
func (eh *EventHandler) AcceptConnections(l net.Listener) <-chan net.Conn {
	ch := make(chan net.Conn)
	go func() {
		defer close(ch)
		// Continually accept event connections. This loop iterates every time a new connection from an
		// event source is received and blocks at Accept().
		for {
			conn, err := l.Accept()
			if err != nil {
				log.Println("Error accepting:", err.Error())
				continue
			}

			ch <- conn
		}
	}()
	return ch
}

// Reads a stream of events from a TCP connection and stores them in a priority queue.
func (eh *EventHandler) handleEvents(conn net.Conn) <-chan Event {
	ch := make(chan Event)

	go func() {
		// A counter for the total number of valid events received from the connection.
		// TODO What happens when this overflows?
		totalReceived := 0

		// Close connection when done reading.
		defer func() {
			log.Println("Closing event connection")
			log.Println("Total events received:", totalReceived)

			// Send any events left in the queue after the event connection is closed.
			log.Println("Flushing queue")
			eh.flushQueue(eh.queueManager)
			conn.Close()
		}()

		br := bufio.NewReader(conn)
		// Continually read from connection. This loop iterates every time a newline-delimited string
		// is read from the TCP connection. The loop blocks at ReadString().
		for {
			// TODO Could get valid data AND an error
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

			totalReceived++
			ch <- *event
		}
	}()

	return ch
}

// These patterns are used by ParseEvent to match incoming events. They are initialized outside
// the function because compiling regex patterns is very expensive and ParseEvent is called
// intensively.
var fPattern = regexp.MustCompile(`^(\d+)\|F\|(\d+)\|(\d+)\n$`)
var uPattern = regexp.MustCompile(`^(\d+)\|U\|(\d+)\|(\d+)\n$`)
var bPattern = regexp.MustCompile(`^(\d+)\|B\n$`)
var pPattern = regexp.MustCompile(`^(\d+)\|P\|(\d+)\|(\d+)\n$`)
var sPattern = regexp.MustCompile(`^(\d+)\|S\|(\d+)\n$`)

// ParseEvent gets a string and returns an Event or an error.
// TODO Is it OK to return a pointer here?
func (eh *EventHandler) ParseEvent(e string) (*Event, error) {

	var result *Event

	if m := fPattern.FindStringSubmatch(e); len(m) != 0 {
		seq, _ := strconv.Atoi(m[1])
		fuid, _ := strconv.Atoi(m[2])
		tuid, _ := strconv.Atoi(m[3])

		result = &Event{
			rawEvent:   e,
			sequence:   seq,
			eventType:  follow,
			fromUserID: fuid,
			toUserID:   tuid,
		}
	} else if m := uPattern.FindStringSubmatch(e); len(m) != 0 {
		seq, _ := strconv.Atoi(m[1])
		fuid, _ := strconv.Atoi(m[2])
		tuid, _ := strconv.Atoi(m[3])
		result = &Event{
			rawEvent:   e,
			sequence:   seq,
			eventType:  unfollow,
			fromUserID: fuid,
			toUserID:   tuid,
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
			fromUserID: fuid,
			toUserID:   tuid,
		}
	} else if m := sPattern.FindStringSubmatch(e); len(m) != 0 {
		seq, _ := strconv.Atoi(m[1])
		fuid, _ := strconv.Atoi(m[2])
		result = &Event{
			rawEvent:   e,
			sequence:   seq,
			eventType:  statusUpdate,
			fromUserID: fuid,
		}
	} else {
		return nil, errors.New("Invalid event: " + e)
	}

	return result, nil
}

// Processes the received event. Depending on the event's type, processing may include registering
// a Follow or Unfollow event and sending the event to one or more user clients.
func (eh *EventHandler) processEvent(e *Event) {
	switch e.eventType {
	case follow:
		// Register fromUserID as a follower of toUserID and notify toUserID.
		eh.userHandler.Follow(e.fromUserID, e.toUserID)
		eh.userHandler.NotifyUser(e.toUserID, e.rawEvent)
	case unfollow:
		// Remove fromUserID from toUserID's followers.
		eh.userHandler.Unfollow(e.fromUserID, e.toUserID)
	case broadcast:
		// Notify all connected users.
		for u := range eh.userHandler.Users {
			eh.userHandler.NotifyUser(u, e.rawEvent)
		}
	case privateMsg:
		// Notify toUserID.
		eh.userHandler.NotifyUser(e.toUserID, e.rawEvent)
	case statusUpdate:
		// Notify all followers of fromUserID.
		for _, u := range eh.userHandler.Followers(e.fromUserID) {
			eh.userHandler.NotifyUser(u, e.rawEvent)
		}
	default:
		// This is just for safety and good practice since all received events should have been
		// parsed successfully and therefore should not have an invalid event type.
		log.Println("Invalid event type - ignoring")
	}
}

// Empties the queue by processing all remaining messages. This method is called once the event
// source connection has been closed.
func (eh *EventHandler) flushQueue(qm *QueueManager) {
	for qm.queue.Len() > 0 {
		eh.processEvent(qm.popEvent())
	}
}

// NewEventHandler constructs a new EventHandler and returns a pointer to it. It receives a pointer
// to a QueueManager as well as a pointer to a UserHandler.
func NewEventHandler(qm *QueueManager, uh *userclients.UserHandler) *EventHandler {
	return &EventHandler{qm, uh}
}

// Run starts the event handler.
func (eh *EventHandler) Run() {
	// Initialize event source listener
	l, err := net.Listen("tcp", host+":"+port)
	if err != nil {
		log.Println("Error listening for events:", err.Error())
		// TODO Replace os.Exit()
		os.Exit(1)
	}
	defer func() {
		log.Println("Closing event listener")
		l.Close()
	}()
	log.Println("Listening for events on " + host + ":" + port)

	conns := eh.AcceptConnections(l)
	for c := range conns {
		events := eh.handleEvents(c)
		for e := range events {
			// Event looks good. Count it and put it in the queue.
			eh.queueManager.queueEvent(&e)

			// If we have enough events in the queue, process the top event.
			if eh.queueManager.queue.Len() > eventQueueSize {
				eh.processEvent(eh.queueManager.popEvent())
			}
		}
	}
}
