package userclients

import (
	"bufio"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
)

const (
	host = "localhost"
	port = "9099"
)

// User represents a user client that is connected to the server. id is the user's ID and conn is
// holds the connection on which that user is reachable.
type User struct {
	id         int
	connection net.Conn
}

// UserHandler handles users. It is responsible for registering users when they connect to the
// server, sending events to users and updating their followers status.
// Both the Users and the followers fields store data in a map for efficient lookups. The followers
// map has a mutex lock since multiple goroutines access it concurrently for both read and write
// operations.
type UserHandler struct {
	Users     map[int]net.Conn
	uLock     sync.RWMutex
	followers map[int][]int
	fLock     sync.RWMutex
}

// AcceptConnections accepts TCP connections from user clients and sends back net.Conn structs.
func (uh *UserHandler) AcceptConnections(l net.Listener, ch chan<- net.Conn) {
	// Continually accept client connections. This loop iterates every time a new connection from
	// a user client is received and blocks at Accept().
	for {
		conn, err := l.Accept()
		if err != nil {
			log.Println("Error accepting:", err.Error())
			continue
		}
		log.Printf("Accepted a client connection from %v", conn.RemoteAddr())

		ch <- conn
	}
}

// Reads a user ID from the TCP connection and registers the user.
func (uh *UserHandler) handleUser(conn net.Conn) {
	// Close connection when done reading.
	defer func() {
		log.Printf("Closing client connection at %v\n", conn.RemoteAddr())
		conn.Close()
	}()

	// This loop iterates every time a newline-delimited string is read from
	// the TCP connection.
	br := bufio.NewReader(conn)
	for {
		message, err := br.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Println("Error reading client request:", err.Error())
			continue
		}

		// Parse user ID
		userID, err := strconv.Atoi(strings.TrimSpace(message))
		if err != nil {
			log.Printf("Invalid user ID %d: %s", userID, err.Error())
			continue
		}

		uh.registerUser(User{userID, conn})
	}
}

// registerUser maps a user ID to a connection.
func (uh *UserHandler) registerUser(u User) {
	uh.uLock.Lock()
	defer uh.uLock.Unlock()
	uh.Users[u.id] = u.connection
}

// NotifyUser sends a string-encoded event to a user.
func (uh *UserHandler) NotifyUser(id int, message string) {
	// Get connection by user ID. No need to lock here as long as we have just one event source.
	if c, ok := uh.Users[id]; ok {
		c.Write([]byte(message))
	}
}

// Follow registers a user as a follower of another user.
func (uh *UserHandler) Follow(from, to int) {
	uh.fLock.Lock()
	defer uh.fLock.Unlock()
	uh.followers[to] = append(uh.followers[to], from)
}

// Unfollow removes a user from another user's followers list.
func (uh *UserHandler) Unfollow(from, to int) {
	uh.fLock.Lock()
	defer uh.fLock.Unlock()
	// TODO If performance for array lookups becomes a problem, use a sorted array + binary search.
	for i := 0; i < len(uh.followers[to]); i++ {
		if uh.followers[to][i] == from {
			uh.followers[to] = append(uh.followers[to][:i], uh.followers[to][i+1:]...)
		}
	}
}

// Followers returns a list of followers for the given user ID.
func (uh *UserHandler) Followers(id int) []int {
	uh.fLock.RLock()
	defer uh.fLock.RUnlock()
	return uh.followers[id]
}

// NewUserHandler constructs a new UserHandler and returns a pointer to it.
func NewUserHandler() *UserHandler {
	return &UserHandler{
		Users:     make(map[int]net.Conn),
		followers: make(map[int][]int),
	}
}

// Run starts the user handler.
func (uh *UserHandler) Run() {
	// Initialize user clients listener
	l, err := net.Listen("tcp", host+":"+port)
	if err != nil {
		log.Println("Error listening for clients:", err.Error())
		// TODO Replace os.Exit()
		os.Exit(1)
	}
	defer func() {
		log.Println("Closing client listener")
		l.Close()
	}()
	log.Println("Listening for user clients on " + host + ":" + port)

	ch := make(chan net.Conn)
	go uh.AcceptConnections(l, ch)
	for c := range ch {
		go uh.handleUser(c)
	}
}
