package events

import (
	"container/heap"
	"log"
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

var (
	pushChan = make(chan *Event)
	popChan  = make(chan chan *Event)
	lenChan  = make(chan chan int)
	stopChan = make(chan bool)
)

// Stores an event in the queue.
func (qm *QueueManager) pushEvent(e *Event) {
	// TODO why pass Event by reference?
	pushChan <- e
}

// Returns the top (first) event in the queue and deletes it from the queue.
func (qm *QueueManager) popEvent() *Event {
	result := make(chan *Event)
	popChan <- result

	return <-result
}

// queueLength returns the length of the queue. This function is used mainly for validating queue
// length during tests.
func (qm *QueueManager) queueLength() int {
	result := make(chan int)
	lenChan <- result

	return <-result
}

// Run starts watching for incoming queue operations (push / pop) and performs them in
// a thread-safe way. Selecting between push and pop operations serializes access to the
// queue, thus guaranteeing safety.
func (qm *QueueManager) Run() chan bool {
	log.Println("Starting queue")
	go func() {
		defer func() {
			log.Println("Stopping queue")
		}()
		for {
			select {
			case push := <-pushChan:
				heap.Push(qm.queue, push)
			case pop := <-popChan:
				pop <- heap.Pop(qm.queue).(*Event)
			case len := <-lenChan:
				len <- qm.queue.Len()
			case <-stopChan:
				return
			}
		}
	}()

	return stopChan
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
