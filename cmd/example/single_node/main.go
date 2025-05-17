package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/qcserestipy/gohpc/pkg/serve"
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
	// numbPtr := flag.Int("n", 10000000000, "Number of Trials")
	// flag.Parse()
	// nTests := *numbPtr

	type ComputeRequest struct {
		Total int `json:"total"`
	}

	type ReturnType struct {
		Result []float64 `json:"result"`
		Total  int       `json:"total"`
	}

	type Task struct{ Count int }

	server := serve.New[Task, float64](4)
	logrus.Infof("System: %d CPU cores available", server.NumWorkers)

	serve.CreateRoutes[ComputeRequest, ReturnType](
		server.Router,
		"/compute",
		func(req ComputeRequest) (ReturnType, error) {

			nTests := req.Total
			logrus.Infof("Number of Trials: %d", nTests)
			tasks := make([]Task, server.NumWorkers)
			chunk := nTests / server.NumWorkers
			remainder := nTests % server.NumWorkers

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
				"workers":         server.NumWorkers,
				"points_per_task": chunk,
				"remainder":       remainder,
				"total_allocated": totalAllocated,
			}).Info("Work distribution prepared")

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

			start := time.Now()
			partials := server.WorkerPool.Run(tasks, work)
			logrus.WithFields(logrus.Fields{
				"duration": time.Since(start),
				"partials": partials,
			}).Info("Computed partials")

			rt := ReturnType{
				Result: partials,
				Total:  nTests,
			}
			return rt, nil
		},
	)

	// Simulate a client to the server
	var wg sync.WaitGroup
	wg.Add(1)

	// Start server in a goroutine
	go func() {
		serve.Launch(server, 3000)
	}()

	// Start client after server is up
	go func() {
		defer wg.Done()
		// Wait for server to listen
		for {
			resp, err := http.Get("http://localhost:3000/")
			if err == nil {
				resp.Body.Close()
				break
			}
			time.Sleep(100 * time.Millisecond)
		}

		// POST to /compute
		resp, err := http.Post(
			"http://localhost:3000/compute",
			"application/json",
			bytes.NewBuffer([]byte(`{"total":10000000000}`)),
		)
		if err != nil {
			logrus.Fatalf("Client POST failed: %v", err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			logrus.Fatalf("Reading response failed: %v", err)
		}

		// Decode partials and compute pi
		var partials ReturnType
		if err := json.Unmarshal(body, &partials); err != nil {
			logrus.Fatalf("Invalid JSON response: %v\n%s", err, string(body))
		}
		sum := 0.0
		for _, v := range partials.Result {
			sum += v
		}
		pi := 4 * sum / float64(partials.Total)

		fmt.Printf("Client: partials=%v\nπ≈%0.8f\n", partials, pi)
	}()

	wg.Wait()

}
