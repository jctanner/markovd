package api

import "testing"

func TestWithSourceCommit(t *testing.T) {
	if got := withSourceCommit(nil, ""); got != nil {
		t.Fatalf("no commit: got %v, want nil", got)
	}
	in := map[string]string{"tier": "smoke"}
	got := withSourceCommit(in, "abc123")
	if got["workflow_source_commit"] != "abc123" || got["tier"] != "smoke" {
		t.Fatalf("got %v", got)
	}
	if _, changed := in["workflow_source_commit"]; changed {
		t.Fatal("caller's map was modified")
	}
	kept := withSourceCommit(map[string]string{"workflow_source_commit": "mine"}, "abc123")
	if kept["workflow_source_commit"] != "mine" {
		t.Fatalf("caller's value replaced: %v", kept)
	}
}
