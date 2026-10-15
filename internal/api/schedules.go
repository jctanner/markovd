package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
	_ "time/tzdata"

	"encoding/json"
	"github.com/go-chi/chi/v5"
	"github.com/jctanner/markovd/internal/db"
	"github.com/jctanner/markovd/internal/models"
	"github.com/jctanner/markovd/internal/runner"
	"github.com/robfig/cron/v3"
)

type scheduleRequest struct {
	Name     string           `json:"name"`
	Kind     string           `json:"kind"`
	Cron     string           `json:"cron"`
	Timezone string           `json:"timezone"`
	RunAt    *time.Time       `json:"run_at"`
	Request  createRunRequest `json:"request"`
}

func nextScheduleTime(kind, expression, zone string, at *time.Time, now time.Time) (*time.Time, error) {
	if _, err := time.LoadLocation(zone); err != nil {
		return nil, fmt.Errorf("invalid IANA timezone: %s", zone)
	}
	switch kind {
	case "once":
		if expression != "" {
			return nil, errors.New("one-time schedules cannot include cron")
		}
		if at == nil || !at.After(now) {
			return nil, errors.New("run_at must be a future RFC3339 timestamp with an offset")
		}
		t := at.UTC()
		return &t, nil
	case "cron":
		if at != nil {
			return nil, errors.New("cron schedules cannot include run_at")
		}
		if len(strings.Fields(expression)) != 5 {
			return nil, errors.New("cron requires five fields: minute hour day month weekday")
		}
		parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
		spec, err := parser.Parse("CRON_TZ=" + zone + " " + expression)
		if err != nil {
			return nil, fmt.Errorf("invalid cron: %w", err)
		}
		next := spec.Next(now).UTC()
		if next.IsZero() {
			return nil, errors.New("cron has no next occurrence")
		}
		return &next, nil
	default:
		return nil, errors.New("kind must be once or cron")
	}
}

func (s *Server) handleListSchedules(w http.ResponseWriter, r *http.Request) {
	schedules, err := s.db.ListSchedules(r.Context())
	if err != nil {
		scheduleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, schedules)
}

func (s *Server) scheduleFromPath(w http.ResponseWriter, r *http.Request) *models.Schedule {
	id, err := strconv.Atoi(chi.URLParam(r, "scheduleID"))
	if err != nil || id < 1 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid schedule ID"})
		return nil
	}
	item, err := s.db.GetSchedule(r.Context(), id)
	if err != nil {
		scheduleError(w, err)
		return nil
	}
	if item == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "schedule not found"})
	}
	return item
}

func (s *Server) handleGetSchedule(w http.ResponseWriter, r *http.Request) {
	item := s.scheduleFromPath(w, r)
	if item != nil {
		writeJSON(w, http.StatusOK, item)
	}
}

func (s *Server) handleSaveSchedule(w http.ResponseWriter, r *http.Request) {
	var req scheduleRequest
	// A frozen directory definition may be large; inputs remain bounded.
	r.Body = http.MaxBytesReader(w, r.Body, 2<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid schedule body: " + err.Error()})
		return
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "expected one JSON object"})
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	req.Request.WorkflowName = strings.TrimSpace(req.Request.WorkflowName)
	if req.Name == "" || len(req.Name) > 200 || req.Request.WorkflowName == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name (1-200 characters) and request.workflow_name required"})
		return
	}
	if req.Timezone == "" {
		req.Timezone = "UTC"
	}
	next, err := nextScheduleTime(req.Kind, req.Cron, req.Timezone, req.RunAt, time.Now())
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	var old *models.Schedule
	if r.Method == http.MethodPut {
		old = s.scheduleFromPath(w, r)
		if old == nil {
			return
		}
	}
	var wf *models.Workflow
	if old != nil && old.Workflow.Name == req.Request.WorkflowName {
		wf = &old.Workflow
	} else {
		wf, err = s.db.GetWorkflowByName(r.Context(), req.Request.WorkflowName)
		if err != nil {
			scheduleError(w, err)
			return
		}
		if wf == nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "workflow not found"})
			return
		}
	}
	req.Request.WorkflowEntrypoint = strings.TrimSpace(req.Request.WorkflowEntrypoint)
	req.Request.Vars = withSourceCommit(req.Request.Vars, wf.SourceCommit)
	payload, err := json.Marshal(req.Request)
	if err != nil {
		scheduleError(w, err)
		return
	}
	item := &models.Schedule{Name: req.Name, Kind: req.Kind, Cron: req.Cron, Timezone: req.Timezone, RunAt: req.RunAt, NextRunAt: next, Request: payload, Workflow: *wf, SourceCommit: wf.SourceCommit, CreatedBy: getClaims(r).UserID}
	if old != nil {
		item.ID = old.ID
	}
	saved, err := s.db.SaveSchedule(r.Context(), item)
	if err != nil {
		scheduleError(w, err)
		return
	}
	status := http.StatusCreated
	if old != nil {
		status = http.StatusOK
	}
	writeJSON(w, status, saved)
}

func (s *Server) handlePauseSchedule(w http.ResponseWriter, r *http.Request) {
	item := s.scheduleFromPath(w, r)
	if item == nil {
		return
	}
	if err := s.db.SetScheduleState(r.Context(), item.ID, "paused", nil); err != nil {
		scheduleError(w, err)
		return
	}
	item, _ = s.db.GetSchedule(r.Context(), item.ID)
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) handleResumeSchedule(w http.ResponseWriter, r *http.Request) {
	item := s.scheduleFromPath(w, r)
	if item == nil {
		return
	}
	next, err := nextScheduleTime(item.Kind, item.Cron, item.Timezone, item.RunAt, time.Now())
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error() + "; edit the schedule before resuming"})
		return
	}
	if err = s.db.SetScheduleState(r.Context(), item.ID, "active", next); err != nil {
		scheduleError(w, err)
		return
	}
	item, _ = s.db.GetSchedule(r.Context(), item.ID)
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) handleDeleteSchedule(w http.ResponseWriter, r *http.Request) {
	item := s.scheduleFromPath(w, r)
	if item == nil {
		return
	}
	if err := s.db.DeleteSchedule(r.Context(), item.ID); err != nil {
		scheduleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"deleted": true})
}

func (s *Server) handleScheduleOccurrences(w http.ResponseWriter, r *http.Request) {
	item := s.scheduleFromPath(w, r)
	if item == nil {
		return
	}
	rows, err := s.db.ScheduleOccurrences(r.Context(), item.ID)
	if err != nil {
		scheduleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, rows)
}

func scheduleError(w http.ResponseWriter, err error) {
	if errors.Is(err, db.ErrScheduleBusy) {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "schedule cannot be changed in its current state; a dispatched run is managed from Runs"})
		return
	}
	log.Printf("schedule error: %v", err)
	writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "schedule operation failed"})
}

// RunScheduler uses durable claims, so it can run in every server replica.
// Missed cron times coalesce; no occurrence is retried after an uncertain launch.
func (s *Server) RunScheduler(ctx context.Context) {
	timer := time.NewTicker(time.Second)
	defer timer.Stop()
	for {
		s.scheduleTick(ctx, time.Now())
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
		}
	}
}

func (s *Server) scheduleTick(ctx context.Context, now time.Time) {
	stale, err := s.db.StaleScheduleDispatches(ctx, now.Add(-2*time.Minute))
	if err != nil {
		log.Printf("scheduler recovery: %v", err)
		return
	}
	for i := range stale {
		item := &stale[i]
		run, err := s.db.GetRunByID(ctx, item.LastRunID)
		if err != nil {
			continue
		}
		message := "launch interrupted; inspect the linked run before creating a replacement (no automatic retry)"
		if run != nil && run.Status != "pending" {
			message = ""
		}
		next, _ := nextScheduleTime(item.Kind, item.Cron, item.Timezone, item.RunAt, now)
		if err = s.db.FinishScheduleDispatch(ctx, item, next, message); err != nil && !errors.Is(err, db.ErrScheduleBusy) {
			log.Printf("scheduler recovery finish: %v", err)
		}
	}
	for i := 0; i < 20 && ctx.Err() == nil; i++ {
		random := make([]byte, 12)
		if _, err = rand.Read(random); err != nil {
			return
		}
		item, err := s.db.ClaimDueSchedule(ctx, now, "markov-run-"+hex.EncodeToString(random))
		if err != nil {
			log.Printf("scheduler claim: %v", err)
			return
		}
		if item == nil {
			return
		}
		s.dispatchSchedule(ctx, item)
	}
}

func (s *Server) dispatchSchedule(ctx context.Context, item *models.Schedule) {
	var req createRunRequest
	err := json.Unmarshal(item.Request, &req)
	if err == nil {
		launchCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
		var id string
		id, err = s.runner.Start(launchCtx, runner.RunRequest{RunID: item.LastRunID, WorkflowYAML: item.Workflow.YAML,
			Workflow: models.WorkflowDefinition{Kind: item.Workflow.DefinitionKind, Files: item.Workflow.Files}, WorkflowEntrypoint: req.WorkflowEntrypoint,
			Vars: req.Vars, Debug: req.Debug, Volumes: req.Volumes, SecretVolumes: req.SecretVolumes, CallbackURL: s.callbackURL, CallbackToken: s.callbackToken})
		cancel()
		if err == nil && id != item.LastRunID {
			err = errors.New("runner returned an unexpected run ID")
		}
	}
	message := ""
	if err != nil {
		message = "launch uncertain: " + err.Error() + "; inspect the linked run before creating a replacement"
	}
	next, nextErr := nextScheduleTime(item.Kind, item.Cron, item.Timezone, item.RunAt, time.Now())
	if item.Kind == "cron" && nextErr != nil {
		message = nextErr.Error()
	}
	if err = s.db.FinishScheduleDispatch(ctx, item, next, message); err != nil {
		log.Printf("scheduler finish %d (%s): %v", item.ID, item.LastRunID, err)
		return
	}
	log.Printf("scheduler dispatched schedule=%d run=%s error=%q", item.ID, item.LastRunID, message)
}
