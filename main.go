package main

import (
	"log"
	"net"
	"os"

	"bitbucket.org/johananl/follower-maze/events"
	"bitbucket.org/johananl/follower-maze/userclients"
)

// TODO Print stats

const (
	host            = "localhost"
	eventSourcePort = "9090"
	userClientsPort = "9099"
)

func main() {
	// Initialize logging
	log.SetFlags(log.Lshortfile | log.Lmicroseconds)

	// Initialize event source listener
	es, err := net.Listen("tcp", host+":"+eventSourcePort)
	if err != nil {
		log.Println("Error listening for events:", err.Error())
		os.Exit(1)
	}
	defer func() {
		log.Println("Closing event listener")
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
		log.Println("Closing client listener")
		uc.Close()
	}()
	log.Println("Listening for user clients on " + host + ":" + userClientsPort)

	// Initialize the queue manager
	qm := events.NewQueueManager()

	// Initialize the user handler
	uh := userclients.NewUserHandler()

	// Initialize the event handler
	eh := events.NewEventHandler(qm, uh)

	// Handle events and users concurrently (acceptUsers runs in the main goroutine).
	go eh.AcceptEvents(es)
	uh.AcceptUsers(uc)
}
