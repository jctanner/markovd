package api

import (
	"database/sql"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jctanner/markovd/internal/models"
)

type createWorkflowRequest struct {
	Name string `json:"name"`
	YAML string `json:"yaml"`
}

type updateWorkflowRequest struct {
	YAML string `json:"yaml"`
}

func (s *Server) handleListWorkflows(w http.ResponseWriter, r *http.Request) {
	workflows, err := s.db.ListWorkflows(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list workflows"})
		return
	}
	if workflows == nil {
		workflows = []models.Workflow{}
	}
	writeJSON(w, http.StatusOK, workflows)
}

func (s *Server) handleGetWorkflow(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	wf, err := s.db.GetWorkflowByName(r.Context(), name)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to get workflow"})
		return
	}
	if wf == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "workflow not found"})
		return
	}
	writeJSON(w, http.StatusOK, wf)
}

func (s *Server) handleCreateWorkflow(w http.ResponseWriter, r *http.Request) {
	var req createWorkflowRequest
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.Name == "" || req.YAML == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name and yaml required"})
		return
	}

	claims := getClaims(r)
	wf, err := s.db.CreateWorkflow(r.Context(), req.Name, req.YAML, claims.UserID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create workflow"})
		return
	}
	writeJSON(w, http.StatusCreated, wf)
}

func (s *Server) handleUpdateWorkflow(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	existing, err := s.db.GetWorkflowByName(r.Context(), name)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to get workflow"})
		return
	}
	if existing == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "workflow not found"})
		return
	}
	if existing.ProjectID != nil {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "cannot edit a project-sourced workflow; edit the source repository and re-sync"})
		return
	}

	var req updateWorkflowRequest
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.YAML == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "yaml required"})
		return
	}

	wf, err := s.db.UpdateWorkflow(r.Context(), name, req.YAML)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to update workflow"})
		return
	}
	writeJSON(w, http.StatusOK, wf)
}

func (s *Server) handleWorkflowDiagram(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	wf, err := s.db.GetWorkflowByName(r.Context(), name)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to get workflow"})
		return
	}
	if wf == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "workflow not found"})
		return
	}

	diagram, err := generateDiagramFromYAML(wf.YAML)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to generate diagram"})
		return
	}

	writeJSON(w, http.StatusOK, diagram)
}

func (s *Server) handleDeleteWorkflow(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	if err := s.db.DeleteWorkflow(r.Context(), name); err != nil {
		if err == sql.ErrNoRows {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "workflow not found"})
		} else {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
