package api

import (
	"log"
	"net/http"
	"path/filepath"
	"regexp"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jctanner/markovd/internal/projects"
	"github.com/jctanner/markovd/internal/workflowdef"
)

type createProjectRequest struct {
	Name   string `json:"name"`
	URL    string `json:"url"`
	Branch string `json:"branch"`
}

type importFilesRequest struct {
	Files       []string                   `json:"files"`
	Definitions []projectDefinitionRequest `json:"definitions"`
}

type projectDefinitionRequest struct {
	Path string `json:"path"`
	Kind string `json:"kind"`
}

var safeNameRe = regexp.MustCompile(`[^a-zA-Z0-9._-]`)

func sanitizeProjectName(name string) string {
	return safeNameRe.ReplaceAllString(name, "-")
}

func (s *Server) handleListProjects(w http.ResponseWriter, r *http.Request) {
	list, err := s.db.ListProjects(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if list == nil {
		writeJSON(w, http.StatusOK, []struct{}{})
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) handleCreateProject(w http.ResponseWriter, r *http.Request) {
	var req createProjectRequest
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	if req.Name == "" || req.URL == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name and url are required"})
		return
	}
	if req.Branch == "" {
		req.Branch = "main"
	}

	clonePath := filepath.Join(s.projectsDir, sanitizeProjectName(req.Name))
	claims := getClaims(r)

	project, err := s.db.CreateProject(r.Context(), req.Name, req.URL, req.Branch, clonePath, claims.UserID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, project)
}

func (s *Server) handleGetProject(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid project id"})
		return
	}

	project, err := s.db.GetProject(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if project == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "project not found"})
		return
	}
	writeJSON(w, http.StatusOK, project)
}

func (s *Server) handleDeleteProject(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid project id"})
		return
	}

	clonePath, err := s.db.GetProjectClonePath(r.Context(), id)
	if err == nil && clonePath != "" {
		if rmErr := projects.RemoveClone(clonePath); rmErr != nil {
			log.Printf("failed to remove clone directory %s: %v", clonePath, rmErr)
		}
	}

	if err := s.db.DeleteProject(r.Context(), id); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleSyncProject(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid project id"})
		return
	}

	project, err := s.db.GetProject(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if project == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "project not found"})
		return
	}
	if project.SyncStatus == "syncing" {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "sync already in progress"})
		return
	}

	clonePath, _ := s.db.GetProjectClonePath(r.Context(), id)

	_ = s.db.UpdateProjectSyncStatus(r.Context(), id, "syncing", "", nil)

	if err := projects.CloneOrPull(project.URL, project.Branch, clonePath); err != nil {
		_ = s.db.UpdateProjectSyncStatus(r.Context(), id, "error", err.Error(), nil)
		project.SyncStatus = "error"
		project.SyncError = err.Error()
		writeJSON(w, http.StatusOK, project)
		return
	}

	now := time.Now()
	_ = s.db.UpdateProjectSyncStatus(r.Context(), id, "synced", "", &now)

	linked, _ := s.db.GetWorkflowsByProjectID(r.Context(), id)
	for _, wf := range linked {
		if wf.SourcePath == "" {
			continue
		}
		def, err := projects.ReadWorkflowDefinition(clonePath, wf.SourceRoot, wf.DefinitionKind)
		if err != nil {
			log.Printf("failed to re-sync workflow %s from project %d: %v", wf.Name, id, err)
			continue
		}
		if _, err := s.db.ImportProjectWorkflowDefinition(r.Context(), id, wf.Name, def, wf.SourcePath, wf.SourceRoot, wf.UploadedBy); err != nil {
			log.Printf("failed to update workflow %s: %v", wf.Name, err)
		}
	}

	updated, _ := s.db.GetProject(r.Context(), id)
	if updated == nil {
		updated = project
		updated.SyncStatus = "synced"
		updated.LastSyncedAt = &now
	}
	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) handleListProjectFiles(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid project id"})
		return
	}

	clonePath, err := s.db.GetProjectClonePath(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "project not found"})
		return
	}

	definitions, err := projects.ListWorkflowDefinitions(clonePath)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	linked, _ := s.db.GetWorkflowsByProjectID(r.Context(), id)
	importedPaths := map[string]bool{}
	for _, wf := range linked {
		importedPaths[wf.SourcePath] = true
	}

	type fileEntry struct {
		Path     string `json:"path"`
		Kind     string `json:"kind"`
		Name     string `json:"name"`
		Imported bool   `json:"imported"`
	}
	result := make([]fileEntry, 0, len(definitions))
	for _, d := range definitions {
		result = append(result, fileEntry{Path: d.Path, Kind: d.Kind, Name: d.Name, Imported: importedPaths[d.Path]})
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleImportProjectFiles(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid project id"})
		return
	}

	var req importFilesRequest
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	definitions := req.Definitions
	for _, file := range req.Files {
		definitions = append(definitions, projectDefinitionRequest{Path: file, Kind: "file"})
	}
	if len(definitions) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "no workflow definitions specified"})
		return
	}

	clonePath, err := s.db.GetProjectClonePath(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "project not found"})
		return
	}

	claims := getClaims(r)
	type importResult struct {
		Name  string `json:"name"`
		Path  string `json:"path"`
		Error string `json:"error,omitempty"`
	}
	var results []importResult

	for _, requested := range definitions {
		if requested.Kind == "" {
			requested.Kind = "file"
		}
		def, err := projects.ReadWorkflowDefinition(clonePath, requested.Path, requested.Kind)
		if err != nil {
			results = append(results, importResult{Path: requested.Path, Error: err.Error()})
			continue
		}
		if err := workflowdef.ValidateWithMarkov(r.Context(), s.markovBin, def); err != nil {
			results = append(results, importResult{Path: requested.Path, Error: err.Error()})
			continue
		}

		wfName := deriveWorkflowName(requested.Path)
		_, err = s.db.ImportProjectWorkflowDefinition(r.Context(), id, wfName, def, requested.Path, requested.Path, claims.UserID)
		if err != nil {
			results = append(results, importResult{Name: wfName, Path: requested.Path, Error: err.Error()})
			continue
		}
		results = append(results, importResult{Name: wfName, Path: requested.Path})
	}
	writeJSON(w, http.StatusOK, results)
}

func deriveWorkflowName(filePath string) string {
	return projects.DeriveWorkflowName(filePath)
}
