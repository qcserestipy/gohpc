package serve

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"

	log "github.com/sirupsen/logrus"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type ComputeResponse struct {
	Message string  `json:"message"`
	Result  float64 `json:"result,omitempty"`
}

type ComputeServer struct {
	NumWorkers int
	Router     *chi.Mux
}

func New(workers ...int) *ComputeServer {
	numWorkers := runtime.NumCPU()
	if len(workers) > 0 && workers[0] > 0 {
		numWorkers = workers[0]
	}
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	return &ComputeServer{numWorkers, r}
}

func Launch(s *ComputeServer, targetPort int) {
	addr := fmt.Sprintf(":%d", targetPort)
	log.Infof("▶️  Starting server on %s", addr)
	// ListenAndServe blocks until an error occurs (e.g. port already in use).
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
}
