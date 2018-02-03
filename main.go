package main

import (
	"log"
	"os"
	"os/signal"
	"sync"

	"bitbucket.org/johananl/follower-maze/events"
	"bitbucket.org/johananl/follower-maze/userclients"
)

// TODO Fix public / private symbols

var wg sync.WaitGroup

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
	stopEventHandler := eh.Run(&wg)
	stopUserHandler := uh.Run(&wg)

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

	wg.Wait()
	log.Println("Graceful shutdown complete")
}
