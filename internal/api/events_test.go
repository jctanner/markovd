package api

import "testing"

func TestRootRunID(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"97639fd9", "97639fd9"},
		{"97639fd9-deploy_all-0-health_check", "97639fd9"},
		{"abcd1234-step1", "abcd1234"},
		{"short", "short"},
		{"markov-run-a3ab59e4", "markov-run-a3ab59e4"},
		{"markov-run-a3ab59e4-deploy_all-0-health_check", "markov-run-a3ab59e4"},
		{"markov-run-a3ab59e4-step1", "markov-run-a3ab59e4"},
		{"markov-run-", "markov-run-"},
	}

	for _, tt := range tests {
		got := rootRunID(tt.input)
		if got != tt.want {
			t.Errorf("rootRunID(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestForkID(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"97639fd9", ""},
		{"97639fd9-deploy_all-0", "deploy_all-0"},
		{"97639fd9-step1", "step1"},
		{"short", ""},
		{"markov-run-a3ab59e4", ""},
		{"markov-run-a3ab59e4-deploy_all-0-health_check", "deploy_all-0-health_check"},
		{"markov-run-a3ab59e4-step1", "step1"},
		{"markov-run-", ""},
	}

	for _, tt := range tests {
		got := forkID(tt.input)
		if got != tt.want {
			t.Errorf("forkID(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestEventRunID(t *testing.T) {
	for _, tt := range []struct{ root, fork, want string }{
		{"97639fd9", "", ""},
		{"97639fd9", "deploy_all-0", "97639fd9-deploy_all-0"},
		{"markov-run-a3ab59e4", "step1", "markov-run-a3ab59e4-step1"},
	} {
		if got := eventRunID(tt.root, tt.fork); got != tt.want {
			t.Errorf("eventRunID(%q, %q) = %q, want %q", tt.root, tt.fork, got, tt.want)
		}
	}
	// A fork's events must resolve back to the same root and fork.
	id := eventRunID("markov-run-a3ab59e4", "step1")
	if rootRunID(id) != "markov-run-a3ab59e4" || forkID(id) != "step1" {
		t.Errorf("round trip failed for %q", id)
	}
}

func TestParseNonNegative(t *testing.T) {
	if n, err := parseNonNegative("", 7); err != nil || n != 7 {
		t.Errorf("default = %d, %v", n, err)
	}
	if n, err := parseNonNegative("12", 0); err != nil || n != 12 {
		t.Errorf("12 = %d, %v", n, err)
	}
	for _, bad := range []string{"-1", "x", "1.5"} {
		if _, err := parseNonNegative(bad, 0); err == nil {
			t.Errorf("%q should be rejected", bad)
		}
	}
}
