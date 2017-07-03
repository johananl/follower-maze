package main

import "sync"

var queue = make(map[int]*Event)
var qLock = sync.RWMutex{}
var lastSeq int = 0

func queueEvent(e *Event) {
	//log.Printf("Putting sequence %s in queue", e.sequence)
	qLock.Lock()
	defer qLock.Unlock()
	queue[e.sequence] = e
}

func processQueue() {
	for {
		qLock.RLock()
		if e, ok := queue[lastSeq+1]; ok {
			//log.Printf("Processing sequence %d", e.sequence)
			go processEvent(e) // Should probably be synchronous
		}
		qLock.RUnlock()
	}
}

func deleteEvent(e *Event) {
	//log.Printf("Deleting sequence %s from queue", e.sequence)
	qLock.Lock()
	defer qLock.Unlock()
	delete(queue, e.sequence)
	lastSeq = e.sequence
}
