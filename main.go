package main

import (
	"log"
	"os"
	"os/signal"

	"bitbucket.org/johananl/follower-maze/events"
	"bitbucket.org/johananl/follower-maze/userclients"
)

// TODO Fix public / private symbols

func main() {
	// Set logging
	log.SetFlags(log.Lshortfile | log.Lmicroseconds)

	// Initialize queue manager
	qm := events.NewQueueManager()

	// Initialize user handler
	uh := userclients.NewUserHandler()

	// Initialize event handler
	eh := events.NewEventHandler(qm, uh)

	// Handle events and users concurrently
	stopEventHandler := eh.Run()
	stopUserHandler := uh.Run()

	// Listen for SIGINT and shutdown gracefully
	// TODO Verify this works on every OS
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt)

	<-shutdown
	log.Println("SIGINT received - shutting down")

	// TODO Cleanup & wait for shutdown
	stopEventHandler <- true
	stopUserHandler <- true

	log.Println("Graceful shutdown complete")
}
