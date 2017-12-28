package events

import (
	"container/heap"
	"sync"
)

// eventQueueSize has to be equal to or larger than the max batch size used by the event source.
// With a max batch size of 100, a queue size of 100 should suffice to avoid ordering problems.
// However, a larger queue size was used here as a safety measure since the performance impact
// (event delivery delay) is trivial.
const eventQueueSize = 200

// QueueManager manages an event queue. The queue is a priority queue implemented using a min heap
// data structure for event ordering. A heap provides a good solution here since it employs
// efficient sorting upon insertion as well as quick retrieval at a constant time.
type QueueManager struct {
	queue *PriorityQueue
	lock  sync.RWMutex
}

// Stores an event in the queue.
func (qm *QueueManager) queueEvent(e *Event) {
	// This lock isn't necessary as long as there is just one event source since there is no chance
	// for concurrent access to the queue.
	qm.lock.Lock()
	defer qm.lock.Unlock()
	heap.Push(qm.queue, e)
}

// Returns the top (first) event in the queue and deletes it from the queue.
func (qm *QueueManager) popEvent() *Event {
	qm.lock.RLock()
	defer qm.lock.RUnlock()
	return heap.Pop(qm.queue).(*Event)
}

// NewQueueManager constructs a new QueueManager and returns a pointer to it. It initializes the
// queue's data structure (a min heap) and performs a heapify operation on it before returning.
func NewQueueManager() *QueueManager {
	pq := make(PriorityQueue, 0)
	qm := QueueManager{
		queue: &pq,
		lock:  sync.RWMutex{},
	}
	heap.Init(qm.queue)

	return &qm
}

// PriorityQueue implements heap.Interface and holds Events.
type PriorityQueue []*Event

// Len returns the size of the queue.
func (pq PriorityQueue) Len() int { return len(pq) }

// Less returns true if the priority of i is lower than the priority of j.
func (pq PriorityQueue) Less(i, j int) bool {
	return pq[i].sequence < pq[j].sequence
}

// Swap switches the location of i and j in the queue.
func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

// Push inserts a new element to the queue.
func (pq *PriorityQueue) Push(x interface{}) {
	n := len(*pq)
	item := x.(*Event)
	item.index = n
	*pq = append(*pq, item)
}

// Pop returns the first element in the queue and deletes it.
func (pq *PriorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	item.index = -1
	*pq = old[0 : n-1]
	return item
}
