package db

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jctanner/markovd/internal/models"
)

func scheduleFixture(t *testing.T, d *DB, kind string) *models.Schedule {
	t.Helper()
	ctx := context.Background()
	// API and DB packages can run simultaneously against the same test database.
	// Due claims are global, so serialize schedule fixtures across packages.
	lock, err := d.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = lock.ExecContext(ctx, `SELECT pg_advisory_lock(6424960)`); err != nil {
		lock.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = lock.ExecContext(ctx, `SELECT pg_advisory_unlock(6424960)`); _ = lock.Close() })
	u, err := d.CreateUser(ctx, fmt.Sprintf("schedule-%d", time.Now().UnixNano()), "not-a-credential")
	if err != nil {
		t.Fatal(err)
	}
	wf, err := d.CreateWorkflow(ctx, u.Username, "name: test\nworkflows: []\n", u.ID)
	if err != nil {
		t.Fatal(err)
	}
	payload := json.RawMessage(`{"workflow_name":"test","workflow_entrypoint":"nested","vars":{"model":"test-model","repeats":"1"},"debug":true,"volumes":[{"name":"artifacts","pvc":"test-pvc","mount_path":"/out","read_only":true}],"secret_volumes":[{"name":"auth","secret":"test-secret","mount_path":"/auth"}]}`)
	at := time.Now().Add(-time.Minute)
	s, err := d.SaveSchedule(ctx, &models.Schedule{Name: u.Username, Kind: kind, Cron: "* * * * *", Timezone: "UTC", RunAt: &at, NextRunAt: &at, Request: payload, Workflow: *wf, CreatedBy: u.ID, SourceCommit: "snapshot-commit"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = d.Exec(`DELETE FROM schedules WHERE id=$1`, s.ID)
		_, _ = d.Exec(`DELETE FROM runs WHERE triggered_by=$1`, u.ID)
		_ = d.DeleteWorkflow(ctx, wf.Name)
		_, _ = d.Exec(`DELETE FROM users WHERE id=$1`, u.ID)
	})
	return s
}

func TestSchedulesConcurrentClaimAndRestart(t *testing.T) {
	d := testDB(t)
	s := scheduleFixture(t, d, "once")
	ctx := context.Background()
	d.SetMaxOpenConns(8)
	var wg sync.WaitGroup
	var claimed atomic.Int32
	var winner *models.Schedule
	var mu sync.Mutex
	for i := 0; i < 64; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			got, err := d.ClaimDueSchedule(ctx, time.Now(), fmt.Sprintf("schedule-test-%d-%d", s.ID, i))
			if err != nil {
				t.Error(err)
				return
			}
			if got != nil {
				claimed.Add(1)
				mu.Lock()
				winner = got
				mu.Unlock()
			}
		}(i)
	}
	wg.Wait()
	if claimed.Load() != 1 {
		t.Fatalf("claims=%d, want 1", claimed.Load())
	}
	// A new connection/process sees the persisted claim and cannot launch it again.
	reopened, err := New(os.Getenv("MARKOVD_TEST_DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	got, err := reopened.ClaimDueSchedule(ctx, time.Now(), "must-not-launch")
	if err != nil || got != nil {
		t.Fatalf("reclaim: %+v %v", got, err)
	}
	saved, err := reopened.GetSchedule(ctx, s.ID)
	if err != nil || saved.LastRunID != winner.LastRunID || saved.SourceCommit != "snapshot-commit" {
		t.Fatalf("restart lost schedule: %+v %v", saved, err)
	}
	if saved.Workflow.YAML != s.Workflow.YAML || string(saved.Request) != string(winner.Request) {
		t.Fatal("snapshot changed")
	}
	run, err := d.GetRunByID(ctx, winner.LastRunID)
	if err != nil || run.Status != "pending" || run.WorkflowSourceCommit != "snapshot-commit" {
		t.Fatalf("run identity: %+v %v", run, err)
	}
	var vars map[string]string
	if err = json.Unmarshal([]byte(run.VarsJSON), &vars); err != nil || vars["model"] != "test-model" {
		t.Fatalf("vars: %v %v", vars, err)
	}
	if run.VolumesJSON == "[]" || run.SecretVolumesJSON == "[]" {
		t.Fatal("mounts missing")
	}
	if err = d.SetScheduleState(ctx, s.ID, "paused", nil); !errors.Is(err, ErrScheduleBusy) {
		t.Fatalf("pause claimed: %v", err)
	}
	if err = d.DeleteSchedule(ctx, s.ID); !errors.Is(err, ErrScheduleBusy) {
		t.Fatalf("delete claimed: %v", err)
	}
	// A fast callback's terminal status must survive dispatch bookkeeping.
	now := time.Now()
	if err = d.UpdateRunStatus(ctx, winner.LastRunID, "completed", &now); err != nil {
		t.Fatal(err)
	}
	if err = d.FinishScheduleDispatch(ctx, winner, nil, ""); err != nil {
		t.Fatal(err)
	}
	run, _ = d.GetRunByID(ctx, winner.LastRunID)
	if run.Status != "completed" {
		t.Fatal("completion overwritten")
	}
	history, err := d.ScheduleOccurrences(ctx, s.ID)
	if err != nil || len(history) != 1 || history[0].State != "dispatched" {
		t.Fatalf("history: %+v %v", history, err)
	}
	if err = d.FinishScheduleDispatch(ctx, winner, nil, ""); !errors.Is(err, ErrScheduleBusy) {
		t.Fatalf("finished twice: %v", err)
	}
}

func TestCronNoOverlapPauseAndUncertainty(t *testing.T) {
	d := testDB(t)
	s := scheduleFixture(t, d, "cron")
	ctx := context.Background()
	if err := d.SetScheduleState(ctx, s.ID, "paused", nil); err != nil {
		t.Fatal(err)
	}
	if got, err := d.ClaimDueSchedule(ctx, time.Now(), "not-paused"); got != nil || err != nil {
		t.Fatalf("paused claim: %v %v", got, err)
	}
	if err := d.SetScheduleState(ctx, s.ID, "active", s.NextRunAt); err != nil {
		t.Fatal(err)
	}
	first, err := d.ClaimDueSchedule(ctx, time.Now(), fmt.Sprintf("cron-%d-first", s.ID))
	if err != nil || first == nil {
		t.Fatalf("first: %v %v", first, err)
	}
	next := time.Now().Add(-time.Second)
	if err = d.FinishScheduleDispatch(ctx, first, &next, ""); err != nil {
		t.Fatal(err)
	}
	if got, err := d.ClaimDueSchedule(ctx, time.Now(), "must-not-overlap"); got != nil || err != nil {
		t.Fatalf("overlap: %v %v", got, err)
	}
	now := time.Now()
	if err = d.UpdateRunStatus(ctx, first.LastRunID, "completed", &now); err != nil {
		t.Fatal(err)
	}
	second, err := d.ClaimDueSchedule(ctx, time.Now(), fmt.Sprintf("cron-%d-second", s.ID))
	if err != nil || second == nil {
		t.Fatalf("second: %v %v", second, err)
	}
	if err = d.FinishScheduleDispatch(ctx, second, &next, "uncertain test launch"); err != nil {
		t.Fatal(err)
	}
	if got, err := d.ClaimDueSchedule(ctx, time.Now(), "must-not-retry"); got != nil || err != nil {
		t.Fatalf("uncertain retry: %v %v", got, err)
	}
	saved, _ := d.GetSchedule(ctx, s.ID)
	if saved.State != "failed" || saved.LastError != "uncertain test launch" {
		t.Fatalf("uncertainty lost: %+v", saved)
	}
}
