package main

import (
	"container/heap"
	"sync"
)

// eventQueueSize has to be equal to or larger than the max batch
// size used by the event source. With a max batch size of 100, a
// queue size of 100 should suffice to avoid ordering problems.
// However, a larger queue size was used here as a safety measure
// since the performance impact (event delivery delay) is trivial.
const eventQueueSize = 200

// PriorityQueue implements heap.Interface and holds Events.
type PriorityQueue []*Event

func (pq PriorityQueue) Len() int { return len(pq) }

func (pq PriorityQueue) Less(i, j int) bool {
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

// QueueManager manages an event queue. The queue is a priority queue
// implemented using a heap for event ordering.
type QueueManager struct {
	queue *PriorityQueue
	lock  sync.RWMutex
}

func (qm QueueManager) queueEvent(e *Event) {
	// TODO Do we need the mutex here?
	qm.lock.Lock()
	defer qm.lock.Unlock()
	heap.Push(qm.queue, e)
}

func (qm QueueManager) popEvent() *Event {
	// TODO Do we need the mutex here?
	qm.lock.RLock()
	defer qm.lock.RUnlock()
	return heap.Pop(qm.queue).(*Event)
}

func NewQueueManager() *QueueManager {
	pq := make(PriorityQueue, 0)
	qm := QueueManager{
		queue: &pq,
		lock:  sync.RWMutex{},
	}
	heap.Init(qm.queue)

	return &qm
}
