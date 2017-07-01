package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

const (
	host            = "localhost"
	eventSourcePort = "9090"
	userClientsPort = "9099"
)

var users = make(map[int]net.Conn)

var followers = make(map[int][]int)
var fLock = sync.RWMutex{}

var queue = make(map[int]*Event)
var qLock = sync.RWMutex{}
var lastSeq int = 0

type User struct {
	Id         int
	Connection net.Conn
}

type Event struct {
	// TODO Restrict types
	Sequence   int
	Type       string
	FromUserId int
	ToUserId   int
}

func main() {
	// Initialize event source listener
	es, err := net.Listen("tcp", host+":"+eventSourcePort)
	if err != nil {
		log.Println("Error listening for events:", err.Error())
		os.Exit(1)
	}
	defer func() {
		log.Println("Closing event listener...")
		es.Close()
	}()
	log.Println("Listening for events on " + host + ":" + eventSourcePort)

	// Initialize user clients listener
	uc, err := net.Listen("tcp", host+":"+userClientsPort)
	if err != nil {
		log.Println("Error listening for clients:", err.Error())
		os.Exit(1)
	}
	defer func() {
		log.Println("Closing client listener...")
		uc.Close()
	}()
	log.Println("Listening for user clients on " + host + ":" + userClientsPort)

	// Handle requests concurrently
	go processQueue()
	go acceptEvents(es)
	go acceptClients(uc)

	// Block main goroutine
	fmt.Scanln()
}

func acceptEvents(l net.Listener) {
	// Continually accept event connections
	for {
		c, err := l.Accept()
		if err != nil {
			log.Println("Error accepting:", err.Error())
		}

		go handleEvents(c)
	}
}

func acceptClients(l net.Listener) {
	// Continually accept client connections
	for {
		c, err := l.Accept()
		if err != nil {
			log.Println("Error accepting:", err.Error())
		}

		ch := make(chan User)
		go handleClient(c, ch)
		user := <-ch
		users[user.Id] = user.Connection
	}
}

func handleEvents(conn net.Conn) {
	// Close connection when done reading
	defer func() {
		log.Println("Closing event connection...")
		conn.Close()
	}()

	// Continually read from connection
	br := bufio.NewReader(conn)
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

		//go processEvent(event)
		go queueEvent(event)
	}
}

func handleClient(conn net.Conn, ch chan User) {
	// TODO Handle client disconnections?
	// Close connection when done reading
	defer func() {
		//log.Printf("Closing client connection for %v...\n", &conn)
		conn.Close()
	}()

	// Continually read from connection
	for {
		message, err := bufio.NewReader(conn).ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Println("Error reading client request:", err.Error())
		}
		//log.Println("Got a message from user:", strings.TrimSpace(message))

		// Parse user ID
		userId, err := strconv.Atoi(strings.TrimSpace(message))
		if err != nil {
			log.Printf("Invalid user ID %s: %s", userId, err.Error())
		}
		//userId := strings.TrimSpace(message)

		// Register user (map ID to connection)
		ch <- User{userId, conn}
	}
}

// parseEvent gets a string and returns an Event or an error if it cannot parse.
func parseEvent(e string) (*Event, error) {
	//log.Printf("Parsing event %s", e)

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
			Sequence:   seq,
			Type:       "F",
			FromUserId: fuid,
			ToUserId:   tuid,
		}
	} else if m := uPattern.FindStringSubmatch(e); len(m) != 0 {
		seq, _ := strconv.Atoi(m[1])
		fuid, _ := strconv.Atoi(m[2])
		tuid, _ := strconv.Atoi(m[3])
		result = &Event{
			Sequence:   seq,
			Type:       "U",
			FromUserId: fuid,
			ToUserId:   tuid,
		}
	} else if m := bPattern.FindStringSubmatch(e); len(m) != 0 {
		seq, _ := strconv.Atoi(m[1])
		result = &Event{
			Sequence: seq,
			Type:     "B",
		}
	} else if m := pPattern.FindStringSubmatch(e); len(m) != 0 {
		seq, _ := strconv.Atoi(m[1])
		fuid, _ := strconv.Atoi(m[2])
		tuid, _ := strconv.Atoi(m[3])
		result = &Event{
			Sequence:   seq,
			Type:       "P",
			FromUserId: fuid,
			ToUserId:   tuid,
		}
	} else if m := sPattern.FindStringSubmatch(e); len(m) != 0 {
		seq, _ := strconv.Atoi(m[1])
		fuid, _ := strconv.Atoi(m[2])
		result = &Event{
			Sequence:   seq,
			Type:       "S",
			FromUserId: fuid,
		}
	} else {
		return nil, errors.New("Invalid message: " + e)
	}

	return result, nil
}

func constructEvent(e *Event) string {
	var result string

	switch e.Type {
	case "F", "U", "P":
		result = fmt.Sprintf("%s|%s|%s|%s\n", e.Sequence, e.Type, e.FromUserId, e.ToUserId)
	case "B":
		result = fmt.Sprintf("%s|%s\n", e.Sequence, e.Type)
	case "S":
		result = fmt.Sprintf("%s|%s|%s\n", e.Sequence, e.Type, e.FromUserId)
	}

	return result
}

func processEvent(e *Event) {
	switch e.Type {
	case "F":
		//log.Println("Processing Follow event")
		follow(e.FromUserId, e.ToUserId)
		notifyUser(e.ToUserId, constructEvent(e))
	case "U":
		//log.Println("Processing Unfollow event")
		unfollow(e.FromUserId, e.ToUserId)
	case "B":
		//log.Println("Processing broadcast event")
		// Notify all users
		for u, _ := range users {
			notifyUser(u, constructEvent(e))
		}
	case "P":
		//log.Println("Processing Private Msg event")
		notifyUser(e.ToUserId, constructEvent(e))
	case "S":
		//log.Println("Processing Status Update event")
		fLock.RLock()
		defer fLock.RUnlock()
		for _, u := range followers[e.FromUserId] {
			notifyUser(u, constructEvent(e))
		}
	default:
		log.Println("Invalid event type - ignoring")
		return
	}

	// TODO Verify success before removing event
	deleteEvent(e)
}

func notifyUser(id int, message string) {
	log.Printf("Notifying user %s with message %s", id, message)
	// Get connection for user
	if c, ok := users[id]; ok {
		c.Write([]byte(message))
	}
}

func follow(from, to int) {
	log.Printf("User %s follows %s", from, to)
	fLock.Lock()
	defer fLock.Unlock()
	followers[to] = append(followers[to], from)
}

func unfollow(from, to int) {
	log.Printf("User %s unfollows %s", from, to)
	fLock.Lock()
	defer fLock.Unlock()
	for i := 0; i < len(followers[to]); i++ {
		if followers[to][i] == from {
			//log.Printf("Found follower %s for user %s - removing", from, to)
			followers[to] = append(followers[to][:i], followers[to][i+1:]...)
		}
	}
}

func queueEvent(e *Event) {
	log.Printf("Putting sequence %s in queue", e.Sequence)
	qLock.Lock()
	defer qLock.Unlock()
	queue[e.Sequence] = e
}

func processQueue() {
	for {
		qLock.RLock()
		if e, ok := queue[lastSeq + 1]; ok {
			log.Printf("Processing sequence %d", e.Sequence)
			go processEvent(e)
		}
		qLock.RUnlock()
	}
}

func deleteEvent(e *Event) {
	log.Printf("Deleting sequence %s from queue", e.Sequence)
	qLock.Lock()
	defer qLock.Unlock()
	delete(queue, e.Sequence)
	lastSeq = e.Sequence
}
