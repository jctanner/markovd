package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"
)

type callbackEvent struct {
	EventType string          `json:"event_type"`
	RunID     string          `json:"run_id"`
	Timestamp time.Time       `json:"timestamp"`
	Payload   json.RawMessage `json:"-"`
	Raw       json.RawMessage `json:"-"`
}

func (s *Server) handleEvent(w http.ResponseWriter, r *http.Request) {
	header := r.Header.Get("Authorization")
	token := strings.TrimPrefix(header, "Bearer ")
	if token == "" || token != s.callbackToken {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid callback token"})
		return
	}

	var raw json.RawMessage
	if err := readJSON(r, &raw); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}

	var evt callbackEvent
	if err := json.Unmarshal(raw, &evt); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid event structure"})
		return
	}

	if evt.RunID == "" || evt.EventType == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "run_id and event_type required"})
		return
	}

	_, err := s.db.InsertEvent(r.Context(), evt.RunID, evt.EventType, string(raw))
	if err != nil {
		log.Printf("failed to insert event: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to store event"})
		return
	}

	var payload map[string]any
	json.Unmarshal(raw, &payload)

	s.processEvent(r, evt.RunID, evt.EventType, payload)

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// rootRunID extracts the 8-char root run ID from a compound sub-workflow ID.
// e.g. "97639fd9-deploy_all-0-health_check" → "97639fd9"
func rootRunID(id string) string {
	if len(id) >= 8 {
		return id[:8]
	}
	return id
}

// forkID returns the sub-workflow context suffix, or "" for top-level steps.
// e.g. "97639fd9-deploy_all-0" → "deploy_all-0"
func forkID(id string) string {
	if len(id) > 9 && id[8] == '-' {
		return id[9:]
	}
	return ""
}

func (s *Server) processEvent(r *http.Request, runID, eventType string, payload map[string]any) {
	ctx := r.Context()
	root := rootRunID(runID)
	fork := forkID(runID)

	getString := func(key string) string {
		if v, ok := payload[key].(string); ok {
			return v
		}
		return ""
	}

	getTime := func(key string) *time.Time {
		if v, ok := payload[key].(string); ok && v != "" {
			if t, err := time.Parse(time.RFC3339, v); err == nil {
				return &t
			}
		}
		return nil
	}

	ts := getTime("timestamp")

	switch eventType {
	case "run_started":
		_ = s.db.UpsertRunFromEvent(ctx, root, getString("workflow_name"), "running", ts, nil)

	case "run_completed":
		_ = s.db.UpsertRunFromEvent(ctx, root, getString("workflow_name"), "completed", nil, ts)

	case "run_failed":
		_ = s.db.UpsertRunFromEvent(ctx, root, getString("workflow_name"), "failed", nil, ts)

	case "sub_run_started", "sub_run_completed", "sub_run_failed":
		// stored in events table only; no separate runs row needed

	case "step_started":
		_ = s.db.UpsertStep(ctx, root, fork, getString("workflow_name"), getString("step_name"),
			getString("step_type"), "running", "", "", ts, nil)

	case "step_completed":
		outputJSON := ""
		if o, ok := payload["output"]; ok {
			b, _ := json.Marshal(o)
			outputJSON = string(b)
		}
		_ = s.db.UpsertStep(ctx, root, fork, getString("workflow_name"), getString("step_name"),
			getString("step_type"), "completed", outputJSON, "", nil, ts)

	case "step_failed":
		_ = s.db.UpsertStep(ctx, root, fork, getString("workflow_name"), getString("step_name"),
			getString("step_type"), "failed", "", getString("error"), nil, ts)

	case "step_skipped":
		_ = s.db.UpsertStep(ctx, root, fork, getString("workflow_name"), getString("step_name"),
			"", "skipped", "", getString("reason"), ts, ts)

	case "gate_evaluated":
		outputJSON := ""
		if b, err := json.Marshal(map[string]any{
			"action":      payload["action"],
			"fired_rules": payload["fired_rules"],
			"facts":       payload["facts"],
		}); err == nil {
			outputJSON = string(b)
		}
		_ = s.db.UpsertStep(ctx, root, fork, getString("workflow_name"), getString("step_name"),
			"gate", "completed", outputJSON, "", nil, ts)
	}
}
