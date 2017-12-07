package main

import (
	"log"
	"net"
	"os"

	"bitbucket.org/johananl/follower-maze/events"
	"bitbucket.org/johananl/follower-maze/userclients"
)

const (
	host            = "localhost"
	userClientsPort = "9099"
)

func main() {
	// Set logging
	log.SetFlags(log.Lshortfile | log.Lmicroseconds)

	// Initialize queue manager
	qm := events.NewQueueManager()

	// Initialize user handler
	uh := userclients.NewUserHandler()

	// Initialize event handler
	eh := events.NewEventHandler(qm, uh)

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

	// Handle events and users concurrently (acceptUsers runs in the main goroutine)
	go eh.Run()
	uh.AcceptUsers(uc)
}
