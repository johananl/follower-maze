package main

import (
	"fmt"
	"net"
	"os"
)

const (
	host            = "localhost"
	eventSourcePort = "9090"
	userClientsPort = "9099"
)

func main() {
	// Initialize event source listener
	es, err := net.Listen("tcp", host+":"+eventSourcePort)
	if err != nil {
		fmt.Println("Error listening for events:", err.Error())
		os.Exit(1)
	}
	defer es.Close()
	fmt.Println("Listening for events on " + host + ":" + eventSourcePort)

	// Initialize user clients listener
	uc, err := net.Listen("tcp", host+":"+userClientsPort)
	if err != nil {
		fmt.Println("Error listening for clients:", err.Error())
		os.Exit(1)
	}
	defer uc.Close()
	fmt.Println("Listening for user clients on " + host + ":" + userClientsPort)

	// Handle requests concurrently
	go acceptEvents(es)
	acceptClients(uc)
}

func acceptEvents(l net.Listener) {
	for {
		c, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting:", err.Error())
		}
		go handleEvent(c)
	}
}

func acceptClients(l net.Listener) {
	for {
		c, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting:", err.Error())
		}
		go handleClient(c)
	}
}

func handleEvent(conn net.Conn) {
	buf := make([]byte, 1024)
	_, err := conn.Read(buf)
	if err != nil {
		fmt.Println("Error reading event:", err.Error())
	}
	fmt.Println("Got an event:", string(buf))
	conn.Close()
}

func handleClient(conn net.Conn) {
	buf := make([]byte, 1024)
	_, err := conn.Read(buf)
	if err != nil {
		fmt.Println("Error reading client request:", err.Error())
	}
	fmt.Println("Got a client request:", string(buf))
	conn.Close()
}
