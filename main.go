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
	es, err := net.Listen("tcp", host+":"+eventSourcePort)
	if err != nil {
		fmt.Println("Error listening:", err.Error())
		os.Exit(1)
	}
	defer es.Close()
	fmt.Println("Listening for events on " + host + ":" + eventSourcePort)

	uc, err := net.Listen("tcp", host+":"+userClientsPort)
	if err != nil {
		fmt.Println("Error listening:", err.Error())
		os.Exit(1)
	}
	defer uc.Close()
	fmt.Println("Listening for user clients on " + host + ":" + userClientsPort)

	go acceptLoop(es)
	acceptLoop(uc)
}

func acceptLoop(l net.Listener) {
	for {
		c, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting:", err.Error())
		}
		go handleRequest(c)
	}
}

func handleRequest(conn net.Conn) {
	buf := make([]byte, 1024)
	_, err := conn.Read(buf)
	if err != nil {
		fmt.Println("Error reading:", err.Error())
	}
	fmt.Println("Got a message:", string(buf))
	conn.Close()
}
