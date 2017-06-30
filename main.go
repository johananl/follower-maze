package main

import (
	"bufio"
	"fmt"
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
		fmt.Println("Error listening for events:", err.Error())
		os.Exit(1)
	}
	defer func() {
		fmt.Println("Closing event listener...")
		es.Close()
	}()
	fmt.Println("Listening for events on " + host + ":" + eventSourcePort)

	// Initialize user clients listener
	uc, err := net.Listen("tcp", host+":"+userClientsPort)
	if err != nil {
		fmt.Println("Error listening for clients:", err.Error())
		os.Exit(1)
	}
	defer func () {
		fmt.Println("Closing client listener...")
		uc.Close()
	}()
	fmt.Println("Listening for user clients on " + host + ":" + userClientsPort)

	// Handle requests concurrently
	go acceptEvents(es)
	go acceptClients(uc)

	// Block main goroutine
	fmt.Scanln()
}

func acceptEvents(l net.Listener) {
	for {
		c, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting:", err.Error())
		}
		go handleEvents(c)
	}
}

func acceptClients(l net.Listener) {
	for {
		c, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting:", err.Error())
		}
		ch := make(chan User)
		go handleClient(c, ch)
		u := <-ch
		users[u.Id] = u.Connection
	}
}

func handleEvents(conn net.Conn) {
	defer func() {
		fmt.Println("Closing event connection...")
		conn.Close()
	}()

	// Continually read from connection
	for {
		message, err := bufio.NewReader(conn).ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			fmt.Println("Error reading event:", err.Error())
		}
		fmt.Println("Got an event:", strings.TrimSpace(message))
	}
}

func handleClient(conn net.Conn, ch chan User) {
	// TODO Keep connection open after receiving user ID (channels?)
	defer func() {
		fmt.Printf("Closing client connection for %v...\n", &conn)
		conn.Close()
	}()

	for {
		message, err := bufio.NewReader(conn).ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			fmt.Println("Error reading client request:", err.Error())
		}
		fmt.Println("Got a message from user:", strings.TrimSpace(message))

		// Parse user ID
		userId, err := strconv.Atoi(strings.TrimSpace(message))
		if err != nil {
			fmt.Printf("Invalid user ID %s: %s", userId, err.Error())
		}

		// Register user (map ID to connection)
		//users[userId] = &conn
		ch <- User{userId, &conn}
	}
}
