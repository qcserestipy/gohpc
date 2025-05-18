// package serve

// import (
// 	"encoding/json"
// 	"fmt"
// 	"net/http"
// 	"runtime"

// 	"github.com/qcserestipy/gohpc/pkg/workerpool"
// 	log "github.com/sirupsen/logrus"

// 	"github.com/go-chi/chi/v5"
// 	"github.com/go-chi/chi/v5/middleware"
// )

// type ComputeServer[T any, R any] struct {
// 	Router     *chi.Mux
// 	NumWorkers int
// 	WorkerPool *workerpool.WorkerPoolExecutor[T, R]
// }

// func New[T any, R any](workers ...int) *ComputeServer[T, R] {
// 	numWorkers := runtime.NumCPU()
// 	if len(workers) > 0 && workers[0] > 0 {
// 		numWorkers = workers[0]
// 	}
// 	pool := workerpool.New[T, R](numWorkers)
// 	r := chi.NewRouter()
// 	r.Use(middleware.RequestID)
// 	r.Use(middleware.RealIP)
// 	r.Use(middleware.Logger)
// 	r.Use(middleware.Recoverer)
// 	log.Infof("🛠  ComputeServer initialized with %d workers", numWorkers)
// 	return &ComputeServer[T, R]{
// 		Router:     r,
// 		NumWorkers: numWorkers,
// 		WorkerPool: pool,
// 	}
// }

// func Launch[T any, R any](s *ComputeServer[T, R], targetPort int) {
// 	addr := fmt.Sprintf(":%d", targetPort)
// 	log.Infof("🚀  Starting server on %s", addr)
// 	log.Infof("🔌  Ready to accept requests")
// 	if err := http.ListenAndServe(addr, s.Router); err != nil {
// 		log.Fatalf("❌  Server failed: %v", err)
// 	}
// }

// func CreateRoutes[T any, R any](
// 	r *chi.Mux,
// 	path string,
// 	fn func(T) (R, error),
// ) {
// 	r.Post(path, func(w http.ResponseWriter, r *http.Request) {
// 		var req T
// 		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
// 			http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
// 			return
// 		}

// 		res, err := fn(req)
// 		if err != nil {
// 			http.Error(w, "processing error: "+err.Error(), http.StatusInternalServerError)
// 			return
// 		}

//			w.Header().Set("Content-Type", "application/json")
//			if err := json.NewEncoder(w).Encode(res); err != nil {
//				http.Error(w, "encode error: "+err.Error(), http.StatusInternalServerError)
//				return
//			}
//		})
//		log.Infof("📡  Route registered: POST %s", path)
//	}
package serve

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/qcserestipy/gohpc/pkg/workerpool"
	"github.com/sirupsen/logrus"
)

type ComputeServer[Req any, Task any, R any] struct {
	Router     *chi.Mux
	NumWorkers int
	WorkerPool *workerpool.WorkerPoolExecutor[Task, R]
}

type ComputeResponse[R any] struct {
	Result []R `json:"result"`
	Total  int `json:"total"`
}

func New[Req any, Task any, R any](workers ...int) *ComputeServer[Req, Task, R] {
	numWorkers := runtime.NumCPU()
	if len(workers) > 0 && workers[0] > 0 {
		numWorkers = workers[0]
	}
	pool := workerpool.New[Task, R](numWorkers)
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	logrus.Infof("🛠  ComputeServer initialized with %d workers", numWorkers)
	return &ComputeServer[Req, Task, R]{
		Router:     r,
		NumWorkers: numWorkers,
		WorkerPool: pool,
	}
}

func (s *ComputeServer[Req, Task, R]) HandleCompute(
	path string,
	splitter func(req Req, nWorkers int) []Task,
	worker func(task Task) R,
	getTotal func(req Req) int,
) {
	s.Router.Post(path, func(w http.ResponseWriter, r *http.Request) {
		var req Req
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
			return
		}

		tasks := splitter(req, s.NumWorkers)

		start := time.Now()
		partials := s.WorkerPool.Run(tasks, worker)
		logrus.WithFields(logrus.Fields{
			"duration": time.Since(start),
			"partials": partials,
		}).Info("Computed partials")

		resp := ComputeResponse[R]{
			Result: partials,
			Total:  getTotal(req),
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			http.Error(w, "encode error: "+err.Error(), http.StatusInternalServerError)
			return
		}
	})
	logrus.Infof("📡  Compute endpoint registered: POST %s", path)
}

func Launch[Req any, Task any, R any](s *ComputeServer[Req, Task, R], targetPort int) {
	addr := fmt.Sprintf(":%d", targetPort)
	logrus.Infof("🚀  Starting server on %s", addr)
	logrus.Infof("🔌  Ready to accept requests")
	if err := http.ListenAndServe(addr, s.Router); err != nil {
		logrus.Fatalf("❌  Server failed: %v", err)
	}
}
