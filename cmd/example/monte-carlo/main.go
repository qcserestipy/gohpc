package main

import (
	"flag"
	"math"
	"math/rand"
	"runtime"
	"time"

	"github.com/qcserestipy/gohpc/pkg/workerpool"
	"github.com/sirupsen/logrus"
)

func init() {
	formatter := &logrus.TextFormatter{}
	formatter.FullTimestamp = true
	formatter.TimestampFormat = time.RFC3339
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(formatter)
}

func main() {
	logrus.Info("Starting Monte Carlo π approximation")
	numbPtr := flag.Int("n", 10000000000, "Number of Trials")
	flag.Parse()
	nTests := *numbPtr
	logrus.Infof("Number of Trials: %d", nTests)

	// Split work into tasks
	numWorkers := runtime.NumCPU()
	logrus.Infof("System: %d CPU cores available", numWorkers)

	type Task struct{ Count int }
	tasks := make([]Task, numWorkers)
	chunk := nTests / numWorkers
	remainder := nTests % numWorkers

	var totalAllocated int
	for i := range tasks {
		extraPoints := 0
		if i < remainder {
			extraPoints = 1
		}
		tasks[i] = Task{Count: chunk + extraPoints}
		totalAllocated += tasks[i].Count
	}

	logrus.WithFields(logrus.Fields{
		"workers":         numWorkers,
		"points_per_task": chunk,
		"remainder":       remainder,
		"total_allocated": totalAllocated,
	}).Info("Work distribution prepared")

	setupStart := time.Now()
	// Create pool
	pool := workerpool.New[Task, float64](numWorkers)
	logrus.Infof("Worker pool initialized in %v", time.Since(setupStart))

	// Define work
	work := func(t Task) float64 {
		taskStart := time.Now()
		inCircle := 0
		seed := time.Now().UnixNano()
		r := rand.New(rand.NewSource(seed))
		logrus.Debugf("Worker started with seed %d for %d points", seed, t.Count)

		for i := 0; i < t.Count; i++ {
			x, y := r.Float64(), r.Float64()
			if x*x+y*y < 1 {
				inCircle++
			}
		}

		ratio := float64(inCircle) / float64(t.Count)
		logrus.WithFields(logrus.Fields{
			"points_processed": t.Count,
			"points_in_circle": inCircle,
			"local_ratio":      ratio,
			"local_pi_approx":  4 * ratio,
			"duration":         time.Since(taskStart),
		}).Debug("Worker completed")

		return float64(inCircle)
	}

	// Run
	start := time.Now()
	logrus.Info("Starting computation...")
	results := pool.Run(tasks, work)
	elapsed := time.Since(start)

	// Aggregate
	total := 0.0
	for _, v := range results {
		total += v
	}
	piApprox := 4 * (total / float64(nTests))

	logrus.WithFields(logrus.Fields{
		"pi_approximation": piApprox,
		"error":            math.Abs(piApprox - math.Pi),
		"duration":         elapsed,
		"points_per_sec":   float64(nTests) / elapsed.Seconds(),
	}).Info("Computation completed")

	logrus.Infof("π ≈ %0.8f (error: %0.8f, computed in %s)",
		piApprox, math.Abs(piApprox-math.Pi), elapsed)
}
