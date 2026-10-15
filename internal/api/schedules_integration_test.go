package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/jctanner/markovd/internal/auth"
	"github.com/jctanner/markovd/internal/db"
	"github.com/jctanner/markovd/internal/models"
	"github.com/jctanner/markovd/internal/runner"
)

type scheduleRunner struct {
	runner.Runner
	starts []runner.RunRequest
	fail   bool
}

func (r *scheduleRunner) Start(_ context.Context, req runner.RunRequest) (string, error) {
	r.starts = append(r.starts, req)
	if r.fail {
		return "", errors.New("simulated lost launch response")
	}
	return req.RunID, nil
}

func TestScheduleAPIAndDispatch(t *testing.T) {
	url := os.Getenv("MARKOVD_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("MARKOVD_TEST_DATABASE_URL not set")
	}
	database, err := db.New(url)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	ctx := context.Background()
	// Match the schedule fixture lock in internal/db: packages share the test DB.
	lock, err := database.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = lock.ExecContext(ctx, `SELECT pg_advisory_lock(6424960)`); err != nil {
		lock.Close()
		t.Fatal(err)
	}
	defer func() { _, _ = lock.ExecContext(ctx, `SELECT pg_advisory_unlock(6424960)`); _ = lock.Close() }()
	user, err := database.CreateUser(ctx, fmt.Sprintf("schedule-api-%d", time.Now().UnixNano()), "not-a-credential")
	if err != nil {
		t.Fatal(err)
	}
	wf, err := database.CreateWorkflow(ctx, user.Username, "entrypoint: main\nworkflows: []\n", user.ID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = database.Exec(`UPDATE workflows SET source_commit='frozen123' WHERE id=$1`, wf.ID)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_, _ = database.Exec(`DELETE FROM schedules WHERE created_by=$1`, user.ID)
		_, _ = database.Exec(`DELETE FROM runs WHERE triggered_by=$1`, user.ID)
		_ = database.DeleteWorkflow(ctx, wf.Name)
		_, _ = database.Exec(`DELETE FROM users WHERE id=$1`, user.ID)
	}()
	jwt := auth.NewJWTManager("test-only", time.Hour)
	token, _ := jwt.Issue(user.ID, user.Username)
	fake := &scheduleRunner{}
	server := NewServer(database, nil, jwt, fake, "", "current-token", "http://current-callback", t.TempDir())
	router := server.Router()
	call := func(method, path string, body any, authenticated bool) *httptest.ResponseRecorder {
		raw, _ := json.Marshal(body)
		req := httptest.NewRequest(method, "/api/v1"+path, bytes.NewReader(raw))
		if authenticated {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		res := httptest.NewRecorder()
		router.ServeHTTP(res, req)
		return res
	}
	future := time.Now().Add(time.Hour)
	input := scheduleRequest{Name: "Test Opus-style payload without any model call", Kind: "once", Timezone: "America/New_York", RunAt: &future,
		Request: createRunRequest{WorkflowName: wf.Name, WorkflowEntrypoint: "main", Vars: map[string]string{"tier": "s1-test", "only_tests": "S1", "variants": "opus", "repeats": "1", "workflow_bg_wait_ceiling_ms": "17700000"}, Debug: true,
			Volumes: []runner.PVCMount{{Name: "data", PVC: "test-pvc", MountPath: "/out", ReadOnly: true}}, SecretVolumes: []runner.SecretMount{{Name: "auth", Secret: "test-secret", MountPath: "/auth", ReadOnly: true}}}}
	if got := call("POST", "/schedules", input, false); got.Code != http.StatusUnauthorized {
		t.Fatalf("auth=%d", got.Code)
	}
	bad := input
	bad.Kind = "bad"
	if got := call("POST", "/schedules", bad, true); got.Code != http.StatusBadRequest {
		t.Fatalf("invalid=%d %s", got.Code, got.Body)
	}
	response := call("POST", "/schedules", input, true)
	if response.Code != http.StatusCreated {
		t.Fatalf("create=%d %s", response.Code, response.Body)
	}
	var saved models.Schedule
	if err = json.Unmarshal(response.Body.Bytes(), &saved); err != nil {
		t.Fatal(err)
	}
	if saved.SourceCommit != "frozen123" || len(fake.starts) != 0 {
		t.Fatal("schedule should save snapshot without launching")
	}
	path := fmt.Sprintf("/schedules/%d", saved.ID)
	for _, action := range []string{"pause", "resume"} {
		if got := call("POST", path+"/"+action, nil, true); got.Code != http.StatusOK {
			t.Fatalf("%s=%d %s", action, got.Code, got.Body)
		}
	}
	input.Name = "Edited timing, same frozen definition"
	if _, err = database.UpdateWorkflow(ctx, wf.Name, "entrypoint: changed\n"); err != nil {
		t.Fatal(err)
	}
	if got := call("PUT", path, input, true); got.Code != http.StatusOK {
		t.Fatalf("edit=%d %s", got.Code, got.Body)
	}
	// Simulate the due time without waiting an hour.
	_, err = database.Exec(`UPDATE schedules SET next_run_at=now()-interval '1 minute' WHERE id=$1`, saved.ID)
	if err != nil {
		t.Fatal(err)
	}
	server.scheduleTick(ctx, time.Now())
	if len(fake.starts) != 1 {
		t.Fatalf("starts=%d", len(fake.starts))
	}
	req := fake.starts[0]
	if req.WorkflowYAML != wf.YAML || req.WorkflowEntrypoint != "main" || req.Vars["workflow_source_commit"] != "frozen123" || req.CallbackToken != "current-token" || req.CallbackURL != "http://current-callback" || !req.Debug {
		t.Fatalf("dispatch lost snapshot/options: %+v", req)
	}
	if !reflect.DeepEqual(req.Volumes, input.Request.Volumes) || !reflect.DeepEqual(req.SecretVolumes, input.Request.SecretVolumes) {
		t.Fatal("mount attributes changed")
	}
	for key, value := range input.Request.Vars {
		if req.Vars[key] != value {
			t.Fatalf("var %s changed", key)
		}
	}
	server.scheduleTick(ctx, time.Now())
	if len(fake.starts) != 1 {
		t.Fatal("one-time schedule launched again")
	}
	history, err := database.ScheduleOccurrences(ctx, saved.ID)
	if err != nil || len(history) != 1 || history[0].RunID != req.RunID {
		t.Fatalf("history: %+v %v", history, err)
	}
	// Restart after a durable claim, with no evidence of an external launch.
	future = time.Now().Add(time.Hour)
	input.Name = "Recovery test"
	response = call("POST", "/schedules", input, true)
	if response.Code != http.StatusCreated {
		t.Fatalf("create recovery: %s", response.Body)
	}
	if err = json.Unmarshal(response.Body.Bytes(), &saved); err != nil {
		t.Fatal(err)
	}
	_, _ = database.Exec(`UPDATE schedules SET next_run_at=now()-interval '1 minute' WHERE id=$1`, saved.ID)
	claimed, err := database.ClaimDueSchedule(ctx, time.Now(), fmt.Sprintf("recovery-%d", saved.ID))
	if err != nil || claimed == nil {
		t.Fatalf("claim recovery: %v %v", claimed, err)
	}
	_, _ = database.Exec(`UPDATE schedules SET updated_at=now()-interval '3 minutes' WHERE id=$1`, saved.ID)
	restarted := NewServer(database, nil, jwt, fake, "", "", "", t.TempDir())
	restarted.scheduleTick(ctx, time.Now())
	got, _ := database.GetSchedule(ctx, saved.ID)
	if len(fake.starts) != 1 || got.State != "failed" || got.LastError == "" {
		t.Fatalf("unsafe recovery: starts=%d state=%s error=%s", len(fake.starts), got.State, got.LastError)
	}
	if got := call("DELETE", fmt.Sprintf("/schedules/%d", saved.ID), nil, true); got.Code != http.StatusOK {
		t.Fatalf("delete=%d %s", got.Code, got.Body)
	}
	// A runner error also stops scheduling instead of spending a second invocation.
	fake.fail = true
	response = call("POST", "/schedules", input, true)
	if response.Code != http.StatusCreated {
		t.Fatal(response.Body)
	}
	_ = json.Unmarshal(response.Body.Bytes(), &saved)
	_, _ = database.Exec(`UPDATE schedules SET next_run_at=now()-interval '1 minute' WHERE id=$1`, saved.ID)
	server.scheduleTick(ctx, time.Now())
	server.scheduleTick(ctx, time.Now())
	got, _ = database.GetSchedule(ctx, saved.ID)
	if got.State != "failed" || len(fake.starts) != 2 {
		t.Fatalf("error retried: %+v starts=%d", got, len(fake.starts))
	}
}
