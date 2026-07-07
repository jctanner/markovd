package runner

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jctanner/markovd/internal/models"
	"github.com/jctanner/markovd/internal/workflowdef"
)

func TestShellRunnerMaterializesFileWorkflow(t *testing.T) {
	record := runShellRunnerWithDefinition(t, models.WorkflowDefinition{
		Kind: workflowdef.KindFile,
		Files: []models.WorkflowDefinitionFile{
			{Path: "workflow.yaml", Content: "entrypoint: main\n"},
		},
	})
	if record.kind != "file" {
		t.Fatalf("materialized kind = %q, want file; args=%q", record.kind, record.args)
	}
	fields := strings.Fields(record.args)
	if len(fields) == 0 || fields[0] != "run" {
		t.Fatalf("args = %q, want markov run invocation", record.args)
	}
}

func TestShellRunnerMaterializesDirectoryWorkflow(t *testing.T) {
	record := runShellRunnerWithDefinition(t, models.WorkflowDefinition{
		Kind: workflowdef.KindDirectory,
		Files: []models.WorkflowDefinitionFile{
			{Path: "meta.yaml", Content: "entrypoint: main\n"},
			{Path: "vars.yaml", Content: "{}\n"},
			{Path: "rules.yaml", Content: "[]\n"},
			{Path: "step_types.yaml", Content: "{}\n"},
			{Path: "workflows/main.yaml", Content: "name: main\nsteps: []\n"},
		},
	})
	if record.kind != "directory" {
		t.Fatalf("materialized kind = %q, want directory; args=%q", record.kind, record.args)
	}
	if record.mainExists != "yes" {
		t.Fatalf("materialized directory did not contain workflows/main.yaml; args=%q", record.args)
	}
}

func TestShellRunnerPassesWorkflowEntrypoint(t *testing.T) {
	record := runShellRunnerWithRequest(t, RunRequest{
		Workflow: models.WorkflowDefinition{
			Kind: workflowdef.KindFile,
			Files: []models.WorkflowDefinitionFile{
				{Path: "workflow.yaml", Content: "entrypoint: main\n"},
			},
		},
		WorkflowEntrypoint: "deploy-target",
	})
	fields := strings.Fields(record.args)
	if !containsArgPair(fields, "--workflow", "deploy-target") {
		t.Fatalf("args = %q, want --workflow deploy-target", record.args)
	}
}

type shellRunnerRecord struct {
	args       string
	kind       string
	mainExists string
}

func runShellRunnerWithDefinition(t *testing.T, def models.WorkflowDefinition) shellRunnerRecord {
	t.Helper()
	return runShellRunnerWithRequest(t, RunRequest{Workflow: def})
}

func runShellRunnerWithRequest(t *testing.T, req RunRequest) shellRunnerRecord {
	t.Helper()
	dir := t.TempDir()
	recordPath := filepath.Join(dir, "record")
	markovPath := filepath.Join(dir, "markov")
	script := `#!/bin/sh
printf ' %s ' "$@" > "$RECORD_PATH.args"
if [ -f "$2" ]; then echo file > "$RECORD_PATH.kind"; fi
if [ -d "$2" ]; then echo directory > "$RECORD_PATH.kind"; fi
if [ -f "$2/workflows/main.yaml" ]; then echo yes > "$RECORD_PATH.main"; fi
sleep 0.2
`
	if err := os.WriteFile(markovPath, []byte(script), 0755); err != nil {
		t.Fatalf("write fake markov: %v", err)
	}
	t.Setenv("RECORD_PATH", recordPath)

	runner := NewShellRunner(markovPath)
	if _, err := runner.Start(context.Background(), req); err != nil {
		t.Fatalf("Start() error: %v", err)
	}

	args := waitForFile(t, recordPath+".args")
	kind := waitForFile(t, recordPath+".kind")
	mainExists := readOptionalFile(t, recordPath+".main")
	return shellRunnerRecord{
		args:       strings.TrimSpace(args),
		kind:       strings.TrimSpace(kind),
		mainExists: strings.TrimSpace(mainExists),
	}
}

func containsArgPair(args []string, flag string, value string) bool {
	for i, arg := range args {
		if arg == flag && i+1 < len(args) && args[i+1] == value {
			return true
		}
	}
	return false
}

func waitForFile(t *testing.T, path string) string {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		data, err := os.ReadFile(path)
		if err == nil {
			return string(data)
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", path)
	return ""
}

func readOptionalFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}
