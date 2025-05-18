package serve

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"

	logrusmiddleware "github.com/chi-middleware/logrus-logger"
	"github.com/qcserestipy/gohpc/pkg/workerpool"
	log "github.com/sirupsen/logrus"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type ComputeServer[T any, R any] struct {
	Router     *chi.Mux
	NumWorkers int
	WorkerPool *workerpool.WorkerPoolExecutor[T, R]
	Jobs       []Job
}

func New[T any, R any](workers ...int) *ComputeServer[T, R] {
	logger := log.New()
	logger.SetFormatter(&log.TextFormatter{FullTimestamp: true})
	numWorkers := runtime.NumCPU()
	if len(workers) > 0 && workers[0] > 0 {
		numWorkers = workers[0]
	}
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(logrusmiddleware.Logger("api", logger))
	r.Use(middleware.Recoverer)
	srv := &ComputeServer[T, R]{
		Router:     r,
		NumWorkers: numWorkers,
		WorkerPool: workerpool.New[T, R](numWorkers),
		Jobs:       make([]Job, 0),
	}
	createJobRoute(r, &srv.Jobs)
	log.Infof("🛠  ComputeServer initialized with %d workers", numWorkers)
	return srv
}

func Launch[T any, R any](s *ComputeServer[T, R], targetPort int) {
	addr := fmt.Sprintf(":%d", targetPort)
	log.Infof("🚀  Starting server on %s", addr)
	log.Infof("🔌  Ready to accept requests")
	if err := http.ListenAndServe(addr, s.Router); err != nil {
		log.Fatalf("❌  Server failed: %v", err)
	}
}

func CreateRoutes[T any, R any](
	r *chi.Mux,
	path string,
	fn func(T) (R, error),
) {
	r.Post(path, func(w http.ResponseWriter, r *http.Request) {
		var req T
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
			return
		}

		res, err := fn(req)
		if err != nil {
			http.Error(w, "processing error: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(res); err != nil {
			http.Error(w, "encode error: "+err.Error(), http.StatusInternalServerError)
			return
		}
	})
	log.Infof("📡  Route registered: POST %s", path)
}
