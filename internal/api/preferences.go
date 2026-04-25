package api

import (
	"encoding/json"
	"net/http"
)

type volumeDefault struct {
	Name      string `json:"name"`
	MountPath string `json:"mount_path"`
}

type preferencesResponse struct {
	DefaultVolumes []volumeDefault `json:"default_volumes"`
	DefaultSecrets []volumeDefault `json:"default_secrets"`
}

type updatePreferencesRequest struct {
	DefaultVolumes []volumeDefault `json:"default_volumes"`
	DefaultSecrets []volumeDefault `json:"default_secrets"`
}

func (s *Server) handleGetPreferences(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)

	prefs, err := s.db.GetPreferences(r.Context(), claims.UserID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to get preferences"})
		return
	}

	resp := preferencesResponse{
		DefaultVolumes: []volumeDefault{},
		DefaultSecrets: []volumeDefault{},
	}
	if prefs != nil {
		json.Unmarshal([]byte(prefs.DefaultVolumes), &resp.DefaultVolumes)
		json.Unmarshal([]byte(prefs.DefaultSecrets), &resp.DefaultSecrets)
		if resp.DefaultVolumes == nil {
			resp.DefaultVolumes = []volumeDefault{}
		}
		if resp.DefaultSecrets == nil {
			resp.DefaultSecrets = []volumeDefault{}
		}
	}

	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleUpdatePreferences(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)

	var req updatePreferencesRequest
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if req.DefaultVolumes == nil {
		req.DefaultVolumes = []volumeDefault{}
	}
	if req.DefaultSecrets == nil {
		req.DefaultSecrets = []volumeDefault{}
	}

	volsJSON, _ := json.Marshal(req.DefaultVolumes)
	secretsJSON, _ := json.Marshal(req.DefaultSecrets)

	_, err := s.db.UpsertPreferences(r.Context(), claims.UserID, string(volsJSON), string(secretsJSON))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to save preferences"})
		return
	}

	resp := preferencesResponse{
		DefaultVolumes: req.DefaultVolumes,
		DefaultSecrets: req.DefaultSecrets,
	}
	writeJSON(w, http.StatusOK, resp)
}
