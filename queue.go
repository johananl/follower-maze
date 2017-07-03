package main

import (
	"container/heap"
	"sync"
)

const (
	eventQueueSize = 200
)

// A PriorityQueue implements heap.Interface and holds Items.
type PriorityQueue []*Event

func (pq PriorityQueue) Len() int { return len(pq) }

func (pq PriorityQueue) Less(i, j int) bool {
	// We want Pop to give us the highest, not lowest, Sequence so we use greater than here.
	return pq[i].sequence < pq[j].sequence
}

func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

func (pq *PriorityQueue) Push(x interface{}) {
	n := len(*pq)
	item := x.(*Event)
	item.index = n
	*pq = append(*pq, item)
}

func (pq *PriorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	item.index = -1 // for safety
	*pq = old[0 : n-1]
	return item
}

// update modifies the Sequence and value of an Event in the queue.
func (pq *PriorityQueue) update(e *Event, et string, seq int) {
	e.eventType = et
	e.sequence = seq
	heap.Fix(pq, e.index)
}

var qLock = sync.RWMutex{}

func (pq *PriorityQueue) queueEvent(e *Event) {
	qLock.Lock()
	defer qLock.Unlock()
	heap.Push(pq, e)
}

func (pq *PriorityQueue) popEvent() *Event {
	qLock.RLock()
	defer qLock.RUnlock()
	return heap.Pop(pq).(*Event)
}

//var queue = make(map[int]*Event)
//var qLock = sync.RWMutex{}
//var lastSeq int = 0

//func queueEvent(e *Event) {
//	//log.Printf("Putting sequence %s in queue", e.sequence)
//	qLock.Lock()
//	defer qLock.Unlock()
//	queue[e.sequence] = e
//}
//
//func processQueue() {
//	for {
//		qLock.RLock()
//		if e, ok := queue[lastSeq+1]; ok {
//			//log.Printf("Processing sequence %d", e.sequence)
//			go processEvent(e) // Should probably be synchronous
//		}
//		qLock.RUnlock()
//	}
//}
//
//func deleteEvent(e *Event) {
//	//log.Printf("Deleting sequence %s from queue", e.sequence)
//	qLock.Lock()
//	defer qLock.Unlock()
//	delete(queue, e.sequence)
//	lastSeq = e.sequence
//}
