// multinode.go
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

type ReturnType struct {
	Result []float64 `json:"result"`
	Total  int       `json:"total"`
}

type Task struct{ Count int }

func init() {
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: time.RFC3339,
	})
	logrus.SetLevel(logrus.DebugLevel)
}

func main() {
	logrus.Info("Starting Monte Carlo π approximation")

	totalPoints := 1_000_000_000_0
	nServers := 2

	servers := make([]*serve.ComputeServer[Task, float64], nServers)
	for i := 0; i < nServers; i++ {
		srv := serve.New[Task, float64]()
		srv.Router.Get("/", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		serve.CreateRoutes[ComputeRequest, ReturnType](
			srv.Router,
			"/compute",
			func(req ComputeRequest) (ReturnType, error) {
				// split req.Total across workers
				nTests := req.Total
				nWorkers := srv.WorkerPool.NumWorkers
				base := nTests / nWorkers
				rem := nTests % nWorkers
				tasks := make([]Task, nWorkers)
				for j := 0; j < nWorkers; j++ {
					extra := 0
					if j < rem {
						extra = 1
					}
					tasks[j] = Task{Count: base + extra}
				}

				work := func(t Task) float64 {
					rng := rand.New(rand.NewSource(time.Now().UnixNano()))
					inC := 0
					for k := 0; k < t.Count; k++ {
						x, y := rng.Float64(), rng.Float64()
						if x*x+y*y < 1 {
							inC++
						}
					}
					return float64(inC)
				}

				start := time.Now()
				partials := srv.WorkerPool.Run(tasks, work)
				logrus.WithFields(logrus.Fields{
					"duration": time.Since(start),
					"partials": partials,
				}).Info("Computed partials")

				return ReturnType{Result: partials, Total: nTests}, nil
			},
		)

		servers[i] = srv
	}

	for idx, srv := range servers {
		go serve.Launch(srv, 3000+idx)
	}

	var wg sync.WaitGroup
	results := make(chan ReturnType, nServers)

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

			data, _ := io.ReadAll(resp.Body)
			var rt ReturnType
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
