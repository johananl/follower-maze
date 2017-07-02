package main

import "sync"

var queue = make(map[int]*Event)
var qLock = sync.RWMutex{}
var lastSeq int = 0

func queueEvent(e *Event) {
	//log.Printf("Putting sequence %s in queue", e.Sequence)
	qLock.Lock()
	defer qLock.Unlock()
	queue[e.Sequence] = e
}

func processQueue() {
	for {
		qLock.RLock()
		if e, ok := queue[lastSeq+1]; ok {
			//log.Printf("Processing sequence %d", e.Sequence)
			go processEvent(e) // Should probably be synchronous
		}
		qLock.RUnlock()
	}
}

func deleteEvent(e *Event) {
	//log.Printf("Deleting sequence %s from queue", e.Sequence)
	qLock.Lock()
	defer qLock.Unlock()
	delete(queue, e.Sequence)
	lastSeq = e.Sequence
}
