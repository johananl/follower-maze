package main

import (
	"log"
	"os"
	"os/signal"

	"github.com/johananl/follower-maze/events"
	"github.com/johananl/follower-maze/userclients"
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

	// Handle events and users concurrently
	stopEventHandler := eh.Run()
	stopUserHandler := uh.Run()

	// Listen for SIGINT and shutdown gracefully
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt)

	// Block until SIGINT is received
	// TODO Handle SIGTERM too
	<-shutdown
	log.Println("SIGINT received - shutting down")

	// TODO Cleanup & wait for shutdown (at the moment there is a timing problem caused by
	// deferred functions)
	stopEventHandler <- true
	stopUserHandler <- true

	log.Println("Graceful shutdown complete")
}
