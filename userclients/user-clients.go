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
	// TODO Remove locks
	Users     map[int]net.Conn
	uLock     sync.RWMutex
	followers map[int][]int
	fLock     sync.RWMutex
}

// AcceptConnections accepts TCP connections from user clients and sends back net.Conn structs.
func (uh *UserHandler) acceptConnections(l net.Listener) (<-chan net.Conn, chan<- bool) {
	ch := make(chan net.Conn)
	quit := make(chan bool)

	go func() {
		defer close(ch)
		// Continually accept user connections. This loop iterates every time a new connection from
		// a user client is received and blocks at Accept().
		for {
			conn, err := l.Accept()
			if err != nil {
				log.Println("Error accepting user connection:", err.Error())

				select {
				case <-quit:
					log.Println("Received quit signal - stopping to listen for user connections")
					return
				default:
				}
			}
			log.Printf("Accepted a user connection from %v", conn.RemoteAddr())

			ch <- conn
		}
	}()

	return ch, quit
}

// Reads a user ID from the TCP connection and returns a User.
func (uh *UserHandler) handleUser(conn net.Conn) <-chan User {
	ch := make(chan User)
	go func() {
		// Close connection when done reading.
		defer func() {
			log.Printf("Closing user connection at %v\n", conn.RemoteAddr())
			conn.Close()
		}()

		br := bufio.NewReader(conn)
		// This loop iterates every time a newline-delimited string is read from
		// the TCP connection.
		for {
			message, err := br.ReadString('\n')
			if err != nil {
				switch err {
				case io.EOF:
					log.Println("Got EOF on user connection")
					return
				case io.ErrClosedPipe: // Used mainly in tests
					log.Println("Got ErrClosedPipe on user connection")
					return
				default:
					log.Println("Error reading user request:", err.Error())
					continue // Skip this message and move to the next one.
				}
			}

			// Parse user ID
			userID, err := strconv.Atoi(strings.TrimSpace(message))
			if err != nil {
				log.Printf("Invalid user ID %d: %s", userID, err.Error())
				continue
			}

			ch <- User{userID, conn}
		}
	}()

	return ch
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
	uh.uLock.RLock()
	defer uh.uLock.RUnlock()
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
func (uh *UserHandler) Run(wg *sync.WaitGroup) chan<- bool {
	wg.Add(1)
	quit := make(chan bool)

	go func() {
		defer wg.Done()
		// Initialize user clients listener
		l, err := net.Listen("tcp", host+":"+port)
		if err != nil {
			log.Println("Error listening for users:", err.Error())
			// TODO Replace os.Exit()
			os.Exit(1)
		}
		defer func() {
			log.Println("Closing user listener")
			l.Close()
		}()

		log.Println("Listening for user clients on " + host + ":" + port)

		connections, stopAccept := uh.acceptConnections(l)
		defer close(stopAccept)

		for {
			select {
			case c := <-connections:
				uch := uh.handleUser(c)
				u := <-uch
				uh.registerUser(u)
			case <-quit:
				log.Println("Stopping user handler")
				return
			}
		}
	}()

	return quit
}
