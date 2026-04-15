package services

import (
	"context"
)

// WorkerPool limits concurrent execution of resource-intensive tasks.
type WorkerPool struct {
	sem chan struct{}
}

// NewWorkerPool creates a new WorkerPool with the given size.
func NewWorkerPool(size int) *WorkerPool {
	return &WorkerPool{
		sem: make(chan struct{}, size),
	}
}

// Do executes a function within the pool's concurrency limit.
// It blocks until a slot is available or the context is cancelled.
func (p *WorkerPool) Do(ctx context.Context, task func()) error {
	select {
	case p.sem <- struct{}{}:
		defer func() { <-p.sem }()
		task()
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Acquire takes a slot in the pool. Must call Release afterwards.
func (p *WorkerPool) Acquire() {
	p.sem <- struct{}{}
}

// Release frees a slot in the pool.
func (p *WorkerPool) Release() {
	<-p.sem
}
