package main

import (
	"log"

	"bitbucket.org/johananl/follower-maze/events"
	"bitbucket.org/johananl/follower-maze/userclients"
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

	// Handle events and users concurrently (acceptUsers runs in the main goroutine)
	go eh.Run()
	uh.Run()

	// TODO Handle graceful shutdown
}
