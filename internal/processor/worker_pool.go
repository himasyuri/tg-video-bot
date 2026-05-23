package processor

import (
	"context"
	"sync"
)

// TaskQueue manages concurrent execution of tasks using a semaphore and a FIFO queue.
type TaskQueue struct {
	maxConcurrent int
	semaphore     chan struct{}
	mu            sync.Mutex
	waiting       []chan struct{}
}

// NewTaskQueue creates a new TaskQueue with the given limit.
func NewTaskQueue(maxConcurrent int) *TaskQueue {
	return &TaskQueue{
		maxConcurrent: maxConcurrent,
		semaphore:     make(chan struct{}, maxConcurrent),
	}
}

// Enter adds a task to the queue. 
func (q *TaskQueue) Enter(ctx context.Context) (int, <-chan struct{}, func()) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.semaphore) < q.maxConcurrent && len(q.waiting) == 0 {
		q.semaphore <- struct{}{}
		return 0, nil, func() { q.release() }
	}

	// Add to waiting list
	ready := make(chan struct{})
	q.waiting = append(q.waiting, ready)
	pos := len(q.waiting)

	return pos, ready, func() { q.release() }
}

func (q *TaskQueue) release() {
	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.waiting) > 0 {
		next := q.waiting[0]
		q.waiting = q.waiting[1:]
		close(next)
	} else {
		<-q.semaphore
	}
}
