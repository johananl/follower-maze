package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"sync"
)

const (
	host            = "localhost"
	eventSourcePort = "9090"
	userClientsPort = "9099"
)

var followers = make(map[int][]int)
var fLock = sync.RWMutex{}

func main() {
	// Initialize logging
	f, err := os.OpenFile("follower-maze.log", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		log.Println("Cannot open log file:", err)
	}
	defer f.Close()

	//mw := io.MultiWriter(os.Stdout, f)
	//log.SetOutput(mw)
	log.SetOutput(f)
	log.SetFlags(log.Lshortfile | log.Lmicroseconds)

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
	defer func() {
		log.Println("Closing client listener...")
		uc.Close()
	}()
	log.Println("Listening for user clients on " + host + ":" + userClientsPort)

	// Handle requests concurrently
	//go processQueue()
	go acceptEvents(es)
	go acceptClients(uc)

	// Block main goroutine
	fmt.Scanln()
}
