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

type User struct {
	id         int
	connection net.Conn
}

type UserHandler struct {
	Users     map[int]net.Conn
	followers map[int][]int
	lock      sync.RWMutex
}

func (uh UserHandler) AcceptUsers(l net.Listener) {
	// Continually accept client connections
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
		user := <-ch // Blocks until handleUser() returns a User
		uh.registerUser(user)
	}
}

func (uh UserHandler) handleUser(conn net.Conn, ch chan User) {
	// TODO Handle client disconnections?
	// Close connection when done reading
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
	uh.Users[u.id] = u.connection
}

func (uh UserHandler) NotifyUser(id int, message string) {
	//log.Printf("Notifying user %d with message %s", id, message)
	// Get connection for user
	if c, ok := uh.Users[id]; ok {
		c.Write([]byte(message))
	}
}

func (uh UserHandler) Follow(from, to int) {
	//log.Printf("User %d follows %d", from, to)
	uh.lock.Lock()
	defer uh.lock.Unlock()
	uh.followers[to] = append(uh.followers[to], from)
}

func (uh UserHandler) Unfollow(from, to int) {
	//log.Printf("User %d unfollows %d", from, to)
	uh.lock.Lock()
	defer uh.lock.Unlock()
	// TODO If performance for array lookup is too expensive, use a sorted array + binary search.
	for i := 0; i < len(uh.followers[to]); i++ {
		if uh.followers[to][i] == from {
			//log.Printf("Found follower %s for user %s - removing", from, to)
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

func NewUserHandler() *UserHandler {
	return &UserHandler{
		Users:     make(map[int]net.Conn),
		followers: make(map[int][]int),
	}
}
