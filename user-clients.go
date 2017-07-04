package main

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
	users map[int]net.Conn
	followers map[int][]int
	lock sync.RWMutex
}

func (uh UserHandler) acceptUsers(l net.Listener) {
	// Continually accept client connections
	for {
		c, err := l.Accept()
		if err != nil {
			log.Println("Error accepting:", err.Error())
			continue
		}

		// TODO Do we need channels here? Maybe using a mutex is simpler.
		ch := make(chan User)
		go uh.handleUser(c, ch)
		user := <-ch // Blocks until handleUser() returns a User
		uh.users[user.id] = user.connection
	}
}

func (uh UserHandler) handleUser(conn net.Conn, ch chan User) {
	// TODO Handle client disconnections?
	// Close connection when done reading
	defer func() {
		log.Printf("Closing client connection at %v...\n", &conn)
		conn.Close()
	}()

	// Continually read from connection
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

		// Register user (map ID to connection)
		ch <- User{userId, conn}
	}
}

func (uh UserHandler) notifyUser(id int, message string) {
	//log.Printf("Notifying user %d with message %s", id, message)
	// Get connection for user
	if c, ok := uh.users[id]; ok {
		c.Write([]byte(message))
	}
}

func (uh UserHandler) follow(from, to int) {
	//log.Printf("User %d follows %d", from, to)
	uh.lock.Lock()
	defer uh.lock.Unlock()
	uh.followers[to] = append(uh.followers[to], from)
}

func (uh UserHandler) unfollow(from, to int) {
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

func NewUserHandler() *UserHandler {
	return &UserHandler{
		users: make(map[int]net.Conn),
		followers: make(map[int][]int),
	}
}
