package api

import (
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
	WorkflowName string            `json:"workflow_name"`
	Vars         map[string]string `json:"vars"`
	Debug        bool              `json:"debug"`
}

type runDetailResponse struct {
	models.Run
	Steps []models.Step `json:"steps"`
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
