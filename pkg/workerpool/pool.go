// Package workerpool provides a generic WorkerPoolExecutor
// that can run arbitrary functions concurrently over a slice of inputs.
package workerpool

import (
	"context"
	"fmt"
	"runtime"
	"sync"
)

type PoolOptions struct {
	NumWorkers int
}

type PoolOptionFunc func(*PoolOptions)

func defaultOpts() PoolOptions {
	return PoolOptions{
		NumWorkers: runtime.NumCPU(),
	}
}

// WithWorkers allows customization of the number of concurrent workers.
func WithWorkers(num int) PoolOptionFunc {
	return func(opts *PoolOptions) {
		opts.NumWorkers = num
	}
}

// WorkerPoolExecutor manages a pool of goroutines to execute tasks.
// T is the input type, R is the output type.
type WorkerPoolExecutor[T any, R any] struct {
	PoolOptions
}

// New creates a new WorkerPoolExecutor with optional configuration.
func New[T any, R any](opts ...PoolOptionFunc) *WorkerPoolExecutor[T, R] {
	o := defaultOpts()
	for _, fn := range opts {
		fn(&o)
	}
	return &WorkerPoolExecutor[T, R]{PoolOptions: o}
}

// Run executes the given function fn on each input using a pool of workers.
// It returns the results in the same order as inputs, and an error if canceled early.
func (w *WorkerPoolExecutor[T, R]) Run(ctx context.Context, inputs []T, fn func(context.Context, T) R) ([]R, error) {
	type task struct {
		idx   int
		input T
	}
	type result struct {
		idx    int
		output R
	}

	tasks := make(chan task)
	results := make(chan result, len(inputs)) // buffered to avoid blocking
	var wg sync.WaitGroup

	// Start worker goroutines
	wg.Add(w.NumWorkers)
	for i := 0; i < w.NumWorkers; i++ {
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case t, ok := <-tasks:
					if !ok {
						return
					}
					out := fn(ctx, t.input)
					select {
					case results <- result{t.idx, out}:
					case <-ctx.Done():
						return
					}
				}
			}
		}()
	}

	// Feed tasks to the channel
	go func() {
		for i, input := range inputs {
			select {
			case <-ctx.Done():
				return
			case tasks <- task{idx: i, input: input}:
			}
		}
		close(tasks)
	}()

	// Wait for all workers to finish, then close results
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	outputs := make([]R, len(inputs))
	received := 0
	for r := range results {
		outputs[r.idx] = r.output
		received++
	}

	// Check for early cancellation
	if err := ctx.Err(); err != nil {
		return outputs, fmt.Errorf("worker pool exited early: %w", err)
	}
	if received < len(inputs) {
		return outputs, fmt.Errorf("worker pool did not return all results: got %d of %d", received, len(inputs))
	}

	return outputs, nil
}
