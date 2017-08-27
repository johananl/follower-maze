package userclients

import (
	"bufio"
	"io"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"
	"fmt"
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
// map has a mutex followersLock since multiple goroutines access it concurrently for both read and write
// operations. The users map doesn't need a followersLock since access to it is regulated by a channel.
type UserHandler struct {
	Users         map[int]net.Conn
	usersLock	  sync.RWMutex
	// TODO Document this
	following     map[int][]int
	followingLock sync.RWMutex
	followers     map[int][]int
	followersLock sync.RWMutex
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
		uh.unregisterUser(conn)
		uh.removeFollowers(conn)
		log.Println(uh.followers)
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
	// We don't need to followersLock here since this function is always called synchronously.
	// TODO ^^?
	uh.Users[u.id] = u.connection
}

// unregisterUser removes the user at connection c from the users database.
func (uh UserHandler) unregisterUser(c net.Conn) error {
	// TODO Find a way to avoid searching uh.Users.
	uid, err := uh.getUserIDByConn(c)
	if err != nil {
		return err
	}

	uh.usersLock.Lock()
	defer uh.usersLock.Unlock()
	delete(uh.Users, uid)
	log.Printf("User %d unregistered successfully", uid)
	return nil
}

// removeFollowers removes following status for the user at connection c. The function unfollows
// all users which were previously followed by the user and also removes the user's followers list.
func (uh UserHandler) removeFollowers(c net.Conn) error {
	uid, err := uh.getUserIDByConn(c)
	if err != nil {
		return err
	}

	for f := range uh.following[uid] {
		uh.Unfollow(uid, f)
	}
	delete(uh.followers, uid)
	return nil
}

// getUserIDByConn returns a user ID that is associated with connection c.
func (uh UserHandler) getUserIDByConn(c net.Conn) (int, error) {
	uh.usersLock.RLock()
	defer uh.usersLock.RUnlock()
	for k, v := range uh.Users {
		if v == c {
			return k, nil
		}
	}

	return 0, fmt.Errorf("No user found with connection %v", c)
}

// NotifyUser sends a string-encoded event to a user.
func (uh UserHandler) NotifyUser(id int, message string) {
	// Get connection by user ID. No need to followersLock here as long as we have just one event source.
	if c, ok := uh.Users[id]; ok {
		c.Write([]byte(message))
	}
}

// Follow registers a user as a follower of another user.
func (uh UserHandler) Follow(from, to int) {
	uh.followersLock.Lock()
	defer uh.followersLock.Unlock()
	uh.followers[to] = append(uh.followers[to], from)
}

// Unfollow removes a user from another user's followers list.
func (uh UserHandler) Unfollow(from, to int) {
	uh.followersLock.Lock()
	defer uh.followersLock.Unlock()
	// TODO If performance for array lookups becomes a problem, use a sorted array + binary search.
	for i := 0; i < len(uh.followers[to]); i++ {
		if uh.followers[to][i] == from {
			uh.followers[to] = append(uh.followers[to][:i], uh.followers[to][i+1:]...)
		}
	}
}

// Followers returns a list of followers for the given user ID.
func (uh UserHandler) Followers(id int) []int {
	uh.followersLock.RLock()
	defer uh.followersLock.RUnlock()
	return uh.followers[id]
}

// NewUserHandler constructs a new UserHandler and returns a pointer to it.
func NewUserHandler() *UserHandler {
	return &UserHandler{
		Users:     make(map[int]net.Conn),
		followers: make(map[int][]int),
	}
}
