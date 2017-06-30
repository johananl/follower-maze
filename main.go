package main

import (
	"fmt"
	"net"
	"os"
)

const (
	HOST              = "localhost"
	EVENT_SOURCE_PORT = "9090"
	USER_CLIENTS_PORT = "9099"
)

func main() {
	es, err := net.Listen("tcp", HOST+":"+EVENT_SOURCE_PORT)
	if err != nil {
		fmt.Println("Error listening:", err.Error())
		os.Exit(1)
	}
	defer es.Close()
	fmt.Println("Listening for events on " + HOST + ":" + EVENT_SOURCE_PORT)

	uc, err := net.Listen("tcp", HOST+":"+USER_CLIENTS_PORT)
	if err != nil {
		fmt.Println("Error listening:", err.Error())
		os.Exit(1)
	}
	defer uc.Close()
	fmt.Println("Listening for user clients on " + HOST + ":" + USER_CLIENTS_PORT)

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
