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

type ComputeRequest struct {
	Total int `json:"total"`
}

type Task struct {
	Count int
}

func main() {
	logrus.Info("Starting Monte Carlo π approximation")

	totalPoints := 1_000_000_0000
	nServers := 2

	servers := make([]*serve.ComputeServer[ComputeRequest, Task, float64], nServers)
	for i := 0; i < nServers; i++ {
		srv := serve.New[ComputeRequest, Task, float64]()

		srv.HandleCompute(
			"/compute",
			func(req ComputeRequest, nWorkers int) []Task {
				base := req.Total / nWorkers
				rem := req.Total % nWorkers
				tasks := make([]Task, nWorkers)
				for j := 0; j < nWorkers; j++ {
					extra := 0
					if j < rem {
						extra = 1
					}
					tasks[j] = Task{Count: base + extra}
				}
				return tasks
			},
			func(t Task) float64 {
				rng := rand.New(rand.NewSource(time.Now().UnixNano()))
				inC := 0
				for k := 0; k < t.Count; k++ {
					x, y := rng.Float64(), rng.Float64()
					if x*x+y*y < 1 {
						inC++
					}
				}
				return float64(inC)
			},
			func(req ComputeRequest) int {
				return req.Total
			},
		)

		servers[i] = srv
	}

	for idx, srv := range servers {
		go serve.Launch(srv, 3000+idx)
	}

	var wg sync.WaitGroup
	results := make(chan serve.ComputeResponse[float64], nServers)

	for idx := 0; idx < nServers; idx++ {
		wg.Add(1)
		go func(node, port int) {
			defer wg.Done()

			urlRoot := fmt.Sprintf("http://localhost:%d/", port)
			for {
				if resp, err := http.Get(urlRoot); err == nil {
					resp.Body.Close()
					break
				}
				time.Sleep(100 * time.Millisecond)
			}

			base := totalPoints / nServers
			rem := totalPoints % nServers
			nodeTotal := base
			if node < rem {
				nodeTotal++
			}
			reqBody := ComputeRequest{Total: nodeTotal}
			buf, _ := json.Marshal(reqBody)

			urlCompute := fmt.Sprintf("http://localhost:%d/compute", port)
			resp, err := http.Post(urlCompute, "application/json", bytes.NewBuffer(buf))
			if err != nil {
				logrus.Fatalf("client POST to %s failed: %v", urlCompute, err)
			}
			defer resp.Body.Close()

			var rt serve.ComputeResponse[float64]
			data, _ := io.ReadAll(resp.Body)
			if err := json.Unmarshal(data, &rt); err != nil {
				logrus.Fatalf("invalid JSON from %s: %s", urlCompute, string(data))
			}

			results <- rt
		}(idx, 3000+idx)
	}

	wg.Wait()
	close(results)

	totalInCircle := 0.0
	for rt := range results {
		for _, v := range rt.Result {
			totalInCircle += v
		}
	}
	pi := 4 * totalInCircle / float64(totalPoints)
	fmt.Printf("Final π ≈ %.8f\n", pi)
}
