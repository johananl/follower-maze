package main

import (
	"bufio"
	"io"
	"log"
	"net"
	"strconv"
	"strings"
)

type User struct {
	id         int
	connection net.Conn
}

// TODO Rename users var to avoid confusion
var users = make(map[int]net.Conn)

func acceptClients(l net.Listener) {
	// Continually accept client connections
	for {
		c, err := l.Accept()
		if err != nil {
			log.Println("Error accepting:", err.Error())
			continue
		}

		// TODO Do we need channels here? Maybe better to run synchronously and return
		ch := make(chan User)
		go handleClient(c, ch)
		user := <-ch // Blocks until handleClient() returns a User
		users[user.id] = user.connection
	}
}

func handleClient(conn net.Conn, ch chan User) {
	// TODO Handle client disconnections?
	// Close connection when done reading
	defer func() {
		log.Printf("Closing client connection for %v...\n", &conn)
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

func notifyUser(id int, message string) {
	log.Printf("Notifying user %d with message %s", id, message)
	// Get connection for user
	if c, ok := users[id]; ok {
		c.Write([]byte(message))
	}
}

func follow(from, to int) {
	log.Printf("User %d follows %d", from, to)
	fLock.Lock()
	defer fLock.Unlock()
	followers[to] = append(followers[to], from)
}

func unfollow(from, to int) {
	log.Printf("User %d unfollows %d", from, to)
	fLock.Lock()
	defer fLock.Unlock()
	// TODO If performance for array lookup is too expensive, use a sorted array + binary search
	for i := 0; i < len(followers[to]); i++ {
		if followers[to][i] == from {
			//log.Printf("Found follower %s for user %s - removing", from, to)
			followers[to] = append(followers[to][:i], followers[to][i+1:]...)
		}
	}
}
