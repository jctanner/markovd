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
