package api

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jctanner/markovd/internal/models"
	"github.com/jctanner/markovd/internal/runner"
)

type createRunRequest struct {
	WorkflowName  string               `json:"workflow_name"`
	Vars          map[string]string    `json:"vars"`
	Debug         bool                 `json:"debug"`
	Volumes       []runner.PVCMount    `json:"volumes,omitempty"`
	SecretVolumes []runner.SecretMount `json:"secret_volumes,omitempty"`
}

type runDetailResponse struct {
	models.Run
	Steps []models.Step `json:"steps"`
}

func (s *Server) handleListPVCs(w http.ResponseWriter, r *http.Request) {
	pvcs, err := s.runner.ListPVCs(r.Context())
	if err != nil {
		log.Printf("failed to list PVCs: %v", err)
		writeJSON(w, http.StatusOK, []struct{}{})
		return
	}
	if pvcs == nil {
		pvcs = []runner.PVCInfo{}
	}
	writeJSON(w, http.StatusOK, pvcs)
}

func (s *Server) handleListSecrets(w http.ResponseWriter, r *http.Request) {
	secrets, err := s.runner.ListSecrets(r.Context())
	if err != nil {
		log.Printf("failed to list Secrets: %v", err)
		writeJSON(w, http.StatusOK, []struct{}{})
		return
	}
	if secrets == nil {
		secrets = []runner.SecretInfo{}
	}
	writeJSON(w, http.StatusOK, secrets)
}

func (s *Server) handleListRuns(w http.ResponseWriter, r *http.Request) {
	runs, err := s.db.ListRuns(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list runs"})
		return
	}
	if runs == nil {
		runs = []models.Run{}
	}
	writeJSON(w, http.StatusOK, runs)
}

func (s *Server) handleGetRun(w http.ResponseWriter, r *http.Request) {
	runID := chi.URLParam(r, "runID")

	run, err := s.db.GetRunByID(r.Context(), runID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to get run"})
		return
	}
	if run == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "run not found"})
		return
	}

	var steps []models.Step
	if sinceStr := r.URL.Query().Get("since"); sinceStr != "" {
		since, parseErr := time.Parse(time.RFC3339Nano, sinceStr)
		if parseErr != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid since parameter, expected RFC3339"})
			return
		}
		steps, err = s.db.GetStepsUpdatedSince(r.Context(), runID, since)
	} else {
		steps, err = s.db.GetStepsByRunID(r.Context(), runID)
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to get steps"})
		return
	}
	if steps == nil {
		steps = []models.Step{}
	}

	writeJSON(w, http.StatusOK, runDetailResponse{Run: *run, Steps: steps})
}

func (s *Server) handleCreateRun(w http.ResponseWriter, r *http.Request) {
	var req createRunRequest
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.WorkflowName == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "workflow_name required"})
		return
	}

	wf, err := s.db.GetWorkflowByName(r.Context(), req.WorkflowName)
	if err != nil || wf == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "workflow not found"})
		return
	}

	claims := getClaims(r)
	varsJSON, _ := json.Marshal(req.Vars)
	volumesJSON, _ := json.Marshal(req.Volumes)
	secretVolumesJSON, _ := json.Marshal(req.SecretVolumes)

	runReq := runner.RunRequest{
		WorkflowYAML:  wf.YAML,
		Vars:          req.Vars,
		CallbackURL:   s.callbackURL,
		CallbackToken: s.callbackToken,
		Debug:         req.Debug,
		Volumes:       req.Volumes,
		SecretVolumes: req.SecretVolumes,
	}

	runID, err := s.runner.Start(r.Context(), runReq)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("failed to start run: %v", err)})
		return
	}

	run, err := s.db.CreateRun(r.Context(), runID, req.WorkflowName, wf.ID, claims.UserID, string(varsJSON), string(volumesJSON), string(secretVolumesJSON))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to record run"})
		return
	}

	writeJSON(w, http.StatusCreated, run)
}

func (s *Server) handleCancelRun(w http.ResponseWriter, r *http.Request) {
	runID := chi.URLParam(r, "runID")

	run, err := s.db.GetRunByID(r.Context(), runID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to get run"})
		return
	}
	if run == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "run not found"})
		return
	}

	if run.Status != "running" && run.Status != "pending" {
		writeJSON(w, http.StatusConflict, map[string]string{"error": fmt.Sprintf("cannot cancel run with status %q", run.Status)})
		return
	}

	if err := s.runner.Cancel(runID); err != nil {
		log.Printf("Warning: cancel runner for %s: %v", runID, err)
	}

	now := time.Now()
	if err := s.db.UpdateRunStatus(r.Context(), runID, "cancelled", &now); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to update run status"})
		return
	}

	run.Status = "cancelled"
	run.CompletedAt = &now
	writeJSON(w, http.StatusOK, run)
}

func (s *Server) handleActiveJobs(w http.ResponseWriter, r *http.Request) {
	running, pending, err := s.db.CountActiveJobs(r.Context())
	if err != nil {
		log.Printf("failed to count active jobs: %v", err)
		writeJSON(w, http.StatusOK, map[string]int{"total": 0, "running": 0, "pending": 0})
		return
	}
	writeJSON(w, http.StatusOK, map[string]int{
		"total":   running + pending,
		"running": running,
		"pending": pending,
	})
}

func (s *Server) handleConcurrencyHistory(w http.ResponseWriter, r *http.Request) {
	buckets, err := s.db.GetConcurrencyHistory(r.Context())
	if err != nil {
		log.Printf("failed to get concurrency history: %v", err)
		writeJSON(w, http.StatusOK, []models.ConcurrencyBucket{})
		return
	}
	if buckets == nil {
		buckets = []models.ConcurrencyBucket{}
	}
	writeJSON(w, http.StatusOK, buckets)
}

func (s *Server) handleGetJobLogs(w http.ResponseWriter, r *http.Request) {
	jobName := chi.URLParam(r, "name")
	if jobName == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "job name required"})
		return
	}

	logs, err := s.runner.GetJobLogs(r.Context(), jobName)
	if err == nil {
		writeJSON(w, http.StatusOK, map[string]string{"logs": logs, "job_name": jobName})
		return
	}

	cachedLogs, found := s.db.FindCachedJobLogs(r.Context(), jobName)
	if found {
		writeJSON(w, http.StatusOK, map[string]string{"logs": cachedLogs, "job_name": jobName, "cached": "true"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"logs": "", "job_name": jobName, "error": err.Error()})
}

func (s *Server) handleStreamJobLogs(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "streaming not supported"})
		return
	}

	jobName := chi.URLParam(r, "name")
	if jobName == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "job name required"})
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	stream, err := s.runner.StreamJobLogs(r.Context(), jobName)
	if err != nil {
		fmt.Fprintf(w, "event: error\ndata: %s\n\n", err.Error())
		flusher.Flush()
		return
	}
	defer stream.Close()

	scanner := bufio.NewScanner(stream)
	for scanner.Scan() {
		if r.Context().Err() != nil {
			return
		}
		fmt.Fprintf(w, "data: %s\n\n", scanner.Text())
		flusher.Flush()
	}

	fmt.Fprintf(w, "event: done\ndata: stream ended\n\n")
	flusher.Flush()
}

func (s *Server) handleGetRunLogs(w http.ResponseWriter, r *http.Request) {
	runID := chi.URLParam(r, "runID")
	if runID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "run ID required"})
		return
	}

	logs, err := s.runner.GetJobLogs(r.Context(), runID)
	if err == nil {
		writeJSON(w, http.StatusOK, map[string]string{"logs": logs, "run_id": runID})
		return
	}

	cachedLogs, found := s.db.FindCachedJobLogs(r.Context(), runID)
	if found {
		writeJSON(w, http.StatusOK, map[string]string{"logs": cachedLogs, "run_id": runID, "cached": "true"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"logs": "", "run_id": runID, "error": err.Error()})
}

func (s *Server) handleStreamRunLogs(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "streaming not supported"})
		return
	}

	runID := chi.URLParam(r, "runID")
	if runID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "run ID required"})
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	stream, err := s.runner.StreamJobLogs(r.Context(), runID)
	if err != nil {
		fmt.Fprintf(w, "event: error\ndata: %s\n\n", err.Error())
		flusher.Flush()
		return
	}
	defer stream.Close()

	scanner := bufio.NewScanner(stream)
	for scanner.Scan() {
		if r.Context().Err() != nil {
			return
		}
		fmt.Fprintf(w, "data: %s\n\n", scanner.Text())
		flusher.Flush()
	}

	fmt.Fprintf(w, "event: done\ndata: stream ended\n\n")
	flusher.Flush()
}

func (s *Server) handleListActiveJobs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	jobs, err := s.db.ListActiveJobs(ctx)
	if err != nil {
		log.Printf("failed to list active jobs: %v", err)
		writeJSON(w, http.StatusOK, []models.ActiveJob{})
		return
	}

	if len(jobs) > 0 {
		k8sStatuses, auditErr := s.runner.AuditJobStatuses(ctx)
		if auditErr == nil && k8sStatuses != nil {
			reconciled := false
			now := time.Now()
			for _, job := range jobs {
				if job.JobName == "" {
					continue
				}
				k8sStatus, exists := k8sStatuses[job.JobName]
				if !exists {
					// job gone from K8s — mark completed
					k8sStatus = "completed"
				}
				if k8sStatus == "running" || k8sStatus == "pending" {
					continue
				}
				log.Printf("reconcile: %s %s (%s) DB=%s K8s=%s", job.Kind, job.JobName, job.StepName, job.Status, k8sStatus)
				reconciled = true
				if job.Kind == "run" {
					_ = s.db.UpdateRunStatus(ctx, job.RunID, k8sStatus, &now)
				} else {
					_ = s.db.UpdateStepStatus(ctx, job.RunID, job.ForkID, job.WorkflowName, job.StepName, k8sStatus, &now)
				}
			}
			if reconciled {
				jobs, _ = s.db.ListActiveJobs(ctx)
			}
		}
	}

	if jobs == nil {
		jobs = []models.ActiveJob{}
	}
	writeJSON(w, http.StatusOK, jobs)
}

type cancelJobRequest struct {
	Kind         string `json:"kind"`
	RunID        string `json:"run_id"`
	ForkID       string `json:"fork_id"`
	WorkflowName string `json:"workflow_name"`
	StepName     string `json:"step_name"`
}

func (s *Server) handleCancelJob(w http.ResponseWriter, r *http.Request) {
	var req cancelJobRequest
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	now := time.Now()

	if req.Kind == "run" {
		run, err := s.db.GetRunByID(r.Context(), req.RunID)
		if err != nil || run == nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "run not found"})
			return
		}
		if run.Status != "running" && run.Status != "pending" {
			writeJSON(w, http.StatusConflict, map[string]string{"error": fmt.Sprintf("cannot cancel run with status %q", run.Status)})
			return
		}
		if err := s.runner.Cancel(req.RunID); err != nil {
			log.Printf("Warning: cancel runner for %s: %v", req.RunID, err)
		}
		if err := s.db.UpdateRunStatus(r.Context(), req.RunID, "cancelled", &now); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to update run status"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "cancelled"})
		return
	}

	if req.Kind == "step" {
		step, err := s.db.GetStepByKey(r.Context(), req.RunID, req.ForkID, req.WorkflowName, req.StepName)
		if err != nil || step == nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "step not found"})
			return
		}
		if step.Status != "running" && step.Status != "pending" {
			writeJSON(w, http.StatusConflict, map[string]string{"error": fmt.Sprintf("cannot cancel step with status %q", step.Status)})
			return
		}
		var output map[string]any
		if step.OutputJSON != "" {
			json.Unmarshal([]byte(step.OutputJSON), &output)
		}
		if jobName, ok := output["job_name"].(string); ok && jobName != "" {
			if err := s.runner.Cancel(jobName); err != nil {
				log.Printf("Warning: cancel K8s job %s: %v", jobName, err)
			}
		}
		if err := s.db.UpdateStepStatus(r.Context(), req.RunID, req.ForkID, req.WorkflowName, req.StepName, "cancelled", &now); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to update step status"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "cancelled"})
		return
	}

	writeJSON(w, http.StatusBadRequest, map[string]string{"error": "kind must be 'run' or 'step'"})
}

func (s *Server) handleDeleteRun(w http.ResponseWriter, r *http.Request) {
	runID := chi.URLParam(r, "runID")

	run, err := s.db.GetRunByID(r.Context(), runID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to get run"})
		return
	}
	if run == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "run not found"})
		return
	}

	if err := s.runner.Cancel(runID); err != nil {
		log.Printf("Warning: cleanup K8s resources for %s: %v", runID, err)
	}

	if err := s.db.DeleteRun(r.Context(), runID); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to delete run"})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
