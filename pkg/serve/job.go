package serve

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

type Resources struct {
	Cpu    string `json:"cpu"`
	Memory string `json:"memory"`
}

type JobStatus string

const (
	StatusPending   JobStatus = "pending"
	StatusRunning   JobStatus = "running"
	StatusCompleted JobStatus = "completed"
	StatusFailed    JobStatus = "failed"
)

type Job struct {
	ID        int               `json:"id"`
	Status    JobStatus         `json:"status"`
	Resources Resources         `json:"resources"`
	Input     map[string]string `json:"input"`
	Output    map[string]string `json:"output"`
}

func NewJob(id int, res Resources, inp map[string]string) (Job, error) {
	if id <= 0 {
		return Job{}, errors.New("id must be > 0")
	}
	if res.Cpu == "" || res.Memory == "" {
		return Job{}, errors.New("resources.cpu and resources.memory are required")
	}
	return Job{
		ID:        id,
		Resources: res,
		Status:    StatusPending,
		Input:     inp,
		Output:    make(map[string]string),
	}, nil
}

func jobExists(w http.ResponseWriter, newJob Job, jobs []Job) bool {
	exists := false
	for _, job := range jobs {
		if newJob.ID == job.ID {
			exists = true
		}
	}
	if exists {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		resp := struct {
			Error string `json:"error"`
			JobID int    `json:"job_id"`
		}{
			Error: fmt.Sprintf("job already exists with ID %d", newJob.ID),
			JobID: newJob.ID,
		}
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			http.Error(w, resp.Error, http.StatusConflict)
		}
	}
	return exists
}

// ----------------------------------------------------------------
// HTTP routes
// ----------------------------------------------------------------

// createJobRoute wires up GET and POST handlers on /jobs.
//   - jobs *[]Job is a pointer so that append() mutates the caller’s slice.
//   - schema validation is strict (DisallowUnknownFields) and ensures
//     id, resources.cpu, and resources.memory are present.
func createJobRoute(r chi.Router, jobs *[]Job) {
	r.Get("/jobs", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(*jobs); err != nil {
			http.Error(w, "encode error: "+err.Error(), http.StatusInternalServerError)
		}
	})

	r.Get("/jobs/{id}", func(w http.ResponseWriter, req *http.Request) {
		idParam := chi.URLParam(req, "id")
		id, err := strconv.Atoi(idParam)
		if err != nil {
			http.Error(w,
				fmt.Sprintf("invalid job ID '%s': %v", idParam, err),
				http.StatusBadRequest,
			)
			return
		}

		for _, job := range *jobs {
			if job.ID == id {
				w.Header().Set("Content-Type", "application/json")
				if err := json.NewEncoder(w).Encode(job); err != nil {
					http.Error(w, "encode error: "+err.Error(), http.StatusInternalServerError)
				}
				return
			}
		}

		http.Error(w,
			fmt.Sprintf("job not found with ID %d", id),
			http.StatusNotFound,
		)
	})

	r.Post("/jobs", func(w http.ResponseWriter, req *http.Request) {
		type jobRequest struct {
			ID        int               `json:"id"`
			Resources Resources         `json:"resources"`
			Input     map[string]string `json:"input"`
		}
		var jr jobRequest

		decoder := json.NewDecoder(req.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&jr); err != nil {
			http.Error(w, "invalid JSON or schema mismatch: "+err.Error(), http.StatusBadRequest)
			return
		}

		newJob, err := NewJob(jr.ID, jr.Resources, jr.Input)
		if err != nil {
			http.Error(w, "validation error: "+err.Error(), http.StatusBadRequest)
			return
		}
		if jobExists(w, newJob, *jobs) {
			return
		}

		*jobs = append(*jobs, newJob)

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(newJob); err != nil {
			http.Error(w, "encode error: "+err.Error(), http.StatusInternalServerError)
		}
	})
}
