package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"io"
)

const (
	host            = "localhost"
	eventSourcePort = "9090"
	userClientsPort = "9099"
)

var users = make(map[int]*net.Conn)

type User struct {
	Id int
	Connection *net.Conn
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
	defer func () {
		log.Println("Closing client listener...")
		uc.Close()
	}()
	log.Println("Listening for user clients on " + host + ":" + userClientsPort)

	// Handle requests concurrently
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
		u := <-ch
		users[u.Id] = u.Connection
	}
}

func handleEvents(conn net.Conn) {
	// Close connection when done reading
	defer func() {
		log.Println("Closing event connection...")
		conn.Close()
	}()

	// Continually read from connection
	for {
		message, err := bufio.NewReader(conn).ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Println("Error reading event:", err.Error())
		}
		log.Println("Got an event:", strings.TrimSpace(message))
	}
}

func handleClient(conn net.Conn, ch chan User) {
	// TODO Keep connection open after receiving user ID (channels?)
	// Close connection when done reading
	defer func() {
		log.Printf("Closing client connection for %v...\n", &conn)
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
		log.Println("Got a message from user:", strings.TrimSpace(message))

		// Parse user ID
		userId, err := strconv.Atoi(strings.TrimSpace(message))
		if err != nil {
			log.Printf("Invalid user ID %s: %s", userId, err.Error())
		}

		// Register user (map ID to connection)
		ch <- User{userId, &conn}
	}
}
