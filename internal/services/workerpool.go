package services

import (
	"context"
	"fmt"
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

// AcquireContext takes a slot in the pool, blocking until available or context is cancelled.
// Must call Release afterwards if nil is returned.
func (p *WorkerPool) AcquireContext(ctx context.Context) error {
	select {
	case p.sem <- struct{}{}:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("worker pool full, coba lagi nanti: %w", ctx.Err())
	}
}

// Release frees a slot in the pool.
func (p *WorkerPool) Release() {
	<-p.sem
}
