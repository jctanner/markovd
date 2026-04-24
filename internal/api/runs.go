package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jctanner/markovd/internal/models"
	"github.com/jctanner/markovd/internal/runner"
)

type createRunRequest struct {
	WorkflowName string            `json:"workflow_name"`
	Vars         map[string]string `json:"vars"`
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
