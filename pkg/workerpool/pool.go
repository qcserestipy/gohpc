// Package workerpool provides a generic WorkerPoolExecutor
// that can run arbitrary functions concurrently over a slice of inputs.
package workerpool

import (
	"sync"
)

// WorkerPoolExecutor manages a pool of goroutines to execute tasks.
// NumWorkers sets how many workers will process tasks in parallel.
// The type parameters T (input) and R (output) specify the task signature.
type WorkerPoolExecutor[T any, R any] struct {
	NumWorkers int
}

// New creates a WorkerPoolExecutor with the given number of workers.
func New[T any, R any](numWorkers int) *WorkerPoolExecutor[T, R] {
	return &WorkerPoolExecutor[T, R]{NumWorkers: numWorkers}
}

// Run dispatches each input through fn concurrently, using up to NumWorkers.
// It returns a slice of results in the same order as inputs.
func (w *WorkerPoolExecutor[T, R]) Run(inputs []T, fn func(T) R) []R {
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
	results := make(chan result)

	var wg sync.WaitGroup
	wg.Add(w.NumWorkers)
	// Start worker goroutines
	for range w.NumWorkers {
		go func() {
			defer wg.Done()
			for t := range tasks {
				// Execute the provided function
				out := fn(t.input)
				results <- result{idx: t.idx, output: out}
			}
		}()
	}

	// Feed tasks and close channels appropriately
	go func() {
		for i, input := range inputs {
			tasks <- task{idx: i, input: input}
		}
		close(tasks)
	}()

	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results into the output slice
	var outputs []R
	for r := range results {
		outputs = append(outputs, r.output)
	}
	return outputs
}
