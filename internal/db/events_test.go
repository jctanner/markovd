package db

import (
	"context"
	"fmt"
	"os"
	"testing"
)

// Requires a Postgres instance: MARKOVD_TEST_DATABASE_URL=postgres://...
func testDB(t *testing.T) *DB {
	t.Helper()
	url := os.Getenv("MARKOVD_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("MARKOVD_TEST_DATABASE_URL not set")
	}
	d, err := New(url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = d.Close() })
	return d
}

func progressPayload(runID, wf, step, kind string) string {
	return fmt.Sprintf(`{"event_type":"step_progress","run_id":%q,"workflow_name":%q,"step_name":%q,"step_type":"claude","kind":%q,"data":{"n":1}}`, runID, wf, step, kind)
}

func TestListStepProgressFiltersAndPages(t *testing.T) {
	d := testDB(t)
	ctx := context.Background()
	root := "tprogress1"
	t.Cleanup(func() { _ = d.DeleteRun(ctx, root); _ = d.DeleteRun(ctx, "tprogress10") })

	insert := func(runID, wf, step, kind string) int {
		e, err := d.InsertEvent(ctx, runID, "step_progress", progressPayload(runID, wf, step, kind))
		if err != nil {
			t.Fatal(err)
		}
		return e.ID
	}
	first := insert(root, "main", "ask", "init")
	insert(root, "main", "ask", "text")
	insert(root, "main", "other", "text")         // different step
	insert(root+"-fork-a", "main", "ask", "text") // fork of the run
	insert("tprogress10", "main", "ask", "text")  // different run sharing a prefix
	if _, err := d.InsertEvent(ctx, root, "step_started", `{"step_name":"ask"}`); err != nil {
		t.Fatal(err)
	}

	got, err := d.ListStepProgress(ctx, root, "", "main", "ask", 0, 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("root+forks = %d events, want 3 (prefix-sharing run must be excluded)", len(got))
	}

	got, _ = d.ListStepProgress(ctx, root, root+"-fork-a", "main", "ask", 0, 100)
	if len(got) != 1 || got[0].RunID != root+"-fork-a" {
		t.Fatalf("fork filter = %+v", got)
	}

	got, _ = d.ListStepProgress(ctx, root, "", "main", "ask", first, 100)
	if len(got) != 2 {
		t.Fatalf("after=first = %d events, want 2", len(got))
	}
	got, _ = d.ListStepProgress(ctx, root, "", "main", "ask", 0, 1)
	if len(got) != 1 {
		t.Fatalf("limit = %d events, want 1", len(got))
	}
	got, _ = d.ListStepProgress(ctx, root, "", "", "ask", 0, 100)
	if len(got) != 3 {
		t.Fatalf("empty workflow filter = %d events, want 3", len(got))
	}
}

func TestDeleteRunRemovesForkEvents(t *testing.T) {
	d := testDB(t)
	ctx := context.Background()
	root := "tdelete1"
	for _, id := range []string{root, root + "-fork-a", "tdelete10"} {
		if _, err := d.InsertEvent(ctx, id, "step_progress", progressPayload(id, "main", "ask", "text")); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() { _ = d.DeleteRun(ctx, "tdelete10") })
	if err := d.DeleteRun(ctx, root); err != nil {
		t.Fatal(err)
	}
	for id, want := range map[string]int{root: 0, root + "-fork-a": 0, "tdelete10": 1} {
		evs, err := d.GetEventsByRunID(ctx, id)
		if err != nil {
			t.Fatal(err)
		}
		if len(evs) != want {
			t.Errorf("events for %q = %d, want %d", id, len(evs), want)
		}
	}
}
