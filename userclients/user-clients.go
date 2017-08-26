package userclients

import (
	"bufio"
	"io"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"
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
// operations. The users map doesn't need a lock since access to it is regulated by a channel.
type UserHandler struct {
	Users     map[int]net.Conn
	followers map[int][]int
	lock      sync.RWMutex
}

// AcceptUsers accepts TCP connections from user clients and triggers registration for them.
func (uh UserHandler) AcceptUsers(l net.Listener) {
	// Continually accept client connections. This loop iterates every time a new connection from
	// a user client is received and blocks at Accept().
	for {
		c, err := l.Accept()
		if err != nil {
			log.Println("Error accepting:", err.Error())
			continue
		}

		log.Printf("Accepted a client connection from %v", c.RemoteAddr())

		// The channel is used to ensure safe writes to the users map (in registerUser).
		ch := make(chan User)
		go uh.handleUser(c, ch)
		user := <-ch // Blocks until handleUser() returns a User.
		uh.registerUser(user)
	}
}

// Reads a user ID from the TCP connection and sends a User struct over the channel.
func (uh UserHandler) handleUser(conn net.Conn, ch chan User) {
	// Close connection when done reading.
	defer func() {
		log.Printf("Closing client connection at %v\n", conn.RemoteAddr())
		// TODO Unregister user
		// TODO Remove followers status
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
		userId, err := strconv.Atoi(strings.TrimSpace(message))
		if err != nil {
			log.Printf("Invalid user ID %s: %s", userId, err.Error())
			continue
		}

		ch <- User{userId, conn}
	}
}

// registerUser maps a user ID to a connection.
func (uh UserHandler) registerUser(u User) {
	// We don't need to lock here since this function is always called synchronously.
	uh.Users[u.id] = u.connection
}

// NotifyUser sends a string-encoded event to a user.
func (uh UserHandler) NotifyUser(id int, message string) {
	// Get connection by user ID. No need to lock here as long as we have just one event source.
	if c, ok := uh.Users[id]; ok {
		c.Write([]byte(message))
	}
}

// Follow registers a user as a follower of another user.
func (uh UserHandler) Follow(from, to int) {
	uh.lock.Lock()
	defer uh.lock.Unlock()
	uh.followers[to] = append(uh.followers[to], from)
}

// Unfollow removes a user from another user's followers list.
func (uh UserHandler) Unfollow(from, to int) {
	uh.lock.Lock()
	defer uh.lock.Unlock()
	// TODO If performance for array lookups becomes a problem, use a sorted array + binary search.
	for i := 0; i < len(uh.followers[to]); i++ {
		if uh.followers[to][i] == from {
			uh.followers[to] = append(uh.followers[to][:i], uh.followers[to][i+1:]...)
		}
	}
}

// Followers returns a list of followers for the given user ID.
func (uh UserHandler) Followers(id int) []int {
	uh.lock.RLock()
	defer uh.lock.RUnlock()
	return uh.followers[id]
}

// NewUserHandler constructs a new UserHandler and returns a pointer to it.
func NewUserHandler() *UserHandler {
	return &UserHandler{
		Users:     make(map[int]net.Conn),
		followers: make(map[int][]int),
	}
}
