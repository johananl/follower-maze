package main

import (
	"log"
	"os"
	"os/signal"

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

	// Listen for SIGINT and shutdown gracefully
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt)
	go func() {
		<-shutdown
		log.Println("SIGINT received - shutting down")
		// TODO Cleanup
		os.Exit(0)
	}()

	// Handle events and users concurrently (acceptUsers runs in the main goroutine)
	go eh.Run()
	uh.Run()
}
