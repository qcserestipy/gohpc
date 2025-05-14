package main

import (
	"fmt"
	"math/rand"
	"runtime"
	"time"

	"github.com/qcserestipy/gohpc/pkg/workerpool"
)

func main() {
	const nTests = 10_000_000_000

	// Split work into tasks
	numWorkers := runtime.NumCPU()
	type Task struct{ Count int }
	tasks := make([]Task, numWorkers)
	chunk := nTests / numWorkers
	for i := range tasks {
		tasks[i] = Task{Count: chunk}
	}

	// Create pool
	pool := workerpool.New[Task, float64](numWorkers)

	// Define work
	work := func(t Task) float64 {
		inCircle := 0
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		for i := 0; i < t.Count; i++ {
			x, y := r.Float64(), r.Float64()
			if x*x+y*y < 1 {
				inCircle++
			}
		}
		return float64(inCircle)
	}

	// Run
	start := time.Now()
	results := pool.Run(tasks, work)
	elapsed := time.Since(start)

	// Aggregate
	total := 0.0
	for _, v := range results {
		total += v
	}
	piApprox := 4 * (total / float64(nTests))

	fmt.Printf("π ≈ %0.8f (computed in %s)", piApprox, elapsed)
}
