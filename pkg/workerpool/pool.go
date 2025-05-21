// Package workerpool provides a generic WorkerPoolExecutor
// that can run arbitrary functions concurrently over a slice of inputs.
package workerpool

import (
	"context"
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
// Run dispatches each input through fn concurrently, using up to NumWorkers.
// It returns a slice of results in the same order as inputs.
func (w *WorkerPoolExecutor[T, R]) Run(ctx context.Context, inputs []T, fn func(ctx context.Context, t T) R) ([]R, error) {
	// Internal types to carry index for ordering
	type task struct {
		idx   int
		input T
	}
	type result struct {
		idx    int
		output R
	}

	tasks := make(chan task)
	results := make(chan result, len(inputs))

	var wg sync.WaitGroup
	wg.Add(w.NumWorkers)
	// Start worker goroutines
	for i := 0; i < w.NumWorkers; i++ {
		go func() {
			defer wg.Done()

			for {
				select {
				// 1) Check for cancellation
				case <-ctx.Done():
					return

				// 2) Try to pull a task
				case t, ok := <-tasks:
					if !ok {
						// tasks channel closed → all done
						return
					}

					// 3) (Optional) check again before heavy work
					select {
					case <-ctx.Done():
						return
					default:
					}

					// 4) Do the real work
					out := fn(ctx, t.input)

					// 5) Try to send result, but return on cancel
					select {
					case <-ctx.Done():
						return
					case results <- result{idx: t.idx, output: out}:
					}
				}
			}
		}()
	}

	// Feed tasks and close channels appropriately
	go func() {
		for i, input := range inputs {
			select {
			case <-ctx.Done():
				return // exits the goroutine immediately
			case tasks <- task{idx: i, input: input}:
			}
		}
		close(tasks)
	}()

	go func() {
		wg.Wait()
		close(results)
	}()

	outputs := make([]R, len(inputs))
	collected := 0

	// Keep going until we've seen every expected result
	for collected < len(inputs) {
		select {
		case <-ctx.Done():
			// Context was cancelled: abort immediately, return the error
			return nil, ctx.Err()

		case r, ok := <-results:
			if !ok {
				// results closed unexpectedly (shouldn't really happen unless you close it early), so just return what we have
				return outputs, nil
			}
			// Store by index so order is preserved
			outputs[r.idx] = r.output
			collected++
		}
	}
	return outputs, nil
}
