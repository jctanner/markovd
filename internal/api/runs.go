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

	steps, err := s.db.GetStepsByRunID(r.Context(), runID)
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

	run, err := s.db.CreateRun(r.Context(), runID, req.WorkflowName, wf.ID, claims.UserID, string(varsJSON))
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

	if run.Status == "running" || run.Status == "pending" {
		if err := s.runner.Cancel(runID); err != nil {
			log.Printf("Warning: cancel runner for %s before delete: %v", runID, err)
		}
	}

	if err := s.db.DeleteRun(r.Context(), runID); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to delete run"})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
