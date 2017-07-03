package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"regexp"
	"strconv"
	"strings"
)

type Event struct {
	sequence   int
	eventType  string
	fromUserId int
	toUserId   int
}

func acceptEvents(l net.Listener) {
	// Continually accept event connections
	// This loop iterates every time a new events connection is made.
	for {
		c, err := l.Accept()
		if err != nil {
			log.Println("Error accepting:", err.Error())
			continue
		}

		go handleEvents(c)
	}
}

func handleEvents(conn net.Conn) {
	totalReceived := 0

	// Close connection when done reading
	defer func() {
		log.Println("Closing event connection...")
		log.Println("Total events received:", totalReceived)
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

		event, err := parseEvent(strings.TrimSpace(message))
		if err != nil {
			log.Println("Event parsing failed:", err)
			continue
		}

		log.Println("Received event:", strings.TrimSpace(message))
		totalReceived++
		processEvent(event)
		//go queueEvent(event)
	}
}

// parseEvent gets a string and returns an Event or an error if it cannot parse.
func parseEvent(e string) (*Event, error) {
	//log.Printf("Parsing event %s", e)

	// TODO Get rid of regex matching and handle string manually?
	fPattern := regexp.MustCompile(`^(\d+)\|F\|(\d+)\|(\d+)$`)
	uPattern := regexp.MustCompile(`^(\d+)\|U\|(\d+)\|(\d+)$`)
	bPattern := regexp.MustCompile(`^(\d+)\|B$`)
	pPattern := regexp.MustCompile(`^(\d+)\|P\|(\d+)\|(\d+)$`)
	sPattern := regexp.MustCompile(`^(\d+)\|S\|(\d+)$`)

	var result *Event

	if m := fPattern.FindStringSubmatch(e); len(m) != 0 {
		seq, _ := strconv.Atoi(m[1])
		fuid, _ := strconv.Atoi(m[2])
		tuid, _ := strconv.Atoi(m[3])

		result = &Event{
			sequence:   seq,
			eventType:  "F",
			fromUserId: fuid,
			toUserId:   tuid,
		}
	} else if m := uPattern.FindStringSubmatch(e); len(m) != 0 {
		seq, _ := strconv.Atoi(m[1])
		fuid, _ := strconv.Atoi(m[2])
		tuid, _ := strconv.Atoi(m[3])
		result = &Event{
			sequence:   seq,
			eventType:  "U",
			fromUserId: fuid,
			toUserId:   tuid,
		}
	} else if m := bPattern.FindStringSubmatch(e); len(m) != 0 {
		seq, _ := strconv.Atoi(m[1])
		result = &Event{
			sequence:  seq,
			eventType: "B",
		}
	} else if m := pPattern.FindStringSubmatch(e); len(m) != 0 {
		seq, _ := strconv.Atoi(m[1])
		fuid, _ := strconv.Atoi(m[2])
		tuid, _ := strconv.Atoi(m[3])
		result = &Event{
			sequence:   seq,
			eventType:  "P",
			fromUserId: fuid,
			toUserId:   tuid,
		}
	} else if m := sPattern.FindStringSubmatch(e); len(m) != 0 {
		seq, _ := strconv.Atoi(m[1])
		fuid, _ := strconv.Atoi(m[2])
		result = &Event{
			sequence:   seq,
			eventType:  "S",
			fromUserId: fuid,
		}
	} else {
		return nil, errors.New("Invalid message: " + e)
	}

	return result, nil
}

func processEvent(e *Event) {
	switch e.eventType {
	case "F":
		//log.Println("Processing Follow event")
		follow(e.fromUserId, e.toUserId)
		notifyUser(e.toUserId, constructEvent(e))
	case "U":
		//log.Println("Processing Unfollow event")
		unfollow(e.fromUserId, e.toUserId)
	case "B":
		//log.Println("Processing broadcast event")
		// Notify all users
		// Block only "sender" object until end of broadcast processing (block getting next event from queue)
		for u, _ := range users {
			notifyUser(u, constructEvent(e))
		}
	case "P":
		//log.Println("Processing Private Msg event")
		notifyUser(e.toUserId, constructEvent(e))
	case "S":
		//log.Println("Processing Status Update event")
		fLock.RLock()
		defer fLock.RUnlock()
		for _, u := range followers[e.fromUserId] {
			notifyUser(u, constructEvent(e))
		}
	default:
		log.Println("Invalid event type - ignoring")
		return
	}

	// TODO Verify success before removing event
	//deleteEvent(e)
}

// TODO Cancel this function and instead keep original "message" as a field under Event
func constructEvent(e *Event) string {
	var result string

	switch e.eventType {
	case "F", "U", "P":
		result = fmt.Sprintf("%d|%s|%d|%d\n", e.sequence, e.eventType, e.fromUserId, e.toUserId)
	case "B":
		result = fmt.Sprintf("%d|%s\n", e.sequence, e.eventType)
	case "S":
		result = fmt.Sprintf("%d|%s|%d\n", e.sequence, e.eventType, e.fromUserId)
	}

	return result
}
