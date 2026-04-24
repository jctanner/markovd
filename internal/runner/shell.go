package runner

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"
)

type ShellRunner struct {
	markovBin string
	mu        sync.Mutex
	procs     map[string]*os.Process
}

func NewShellRunner(markovBin string) *ShellRunner {
	return &ShellRunner{
		markovBin: markovBin,
		procs:     make(map[string]*os.Process),
	}
}

func (r *ShellRunner) Start(ctx context.Context, req RunRequest) (string, error) {
	tmpFile, err := os.CreateTemp("", "markov-workflow-*.yaml")
	if err != nil {
		return "", fmt.Errorf("creating temp file: %w", err)
	}
	if _, err := tmpFile.WriteString(req.WorkflowYAML); err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return "", fmt.Errorf("writing workflow: %w", err)
	}
	tmpFile.Close()

	runID := generateRunID()
	args := []string{"run", tmpFile.Name(), "--verbose", "--run-id", runID}
	if req.Debug {
		args = append(args, "--debug")
	}
	for k, v := range req.Vars {
		args = append(args, "--var", fmt.Sprintf("%s=%s", k, v))
	}
	if req.CallbackURL != "" {
		args = append(args, "--callback", req.CallbackURL)
	}
	if req.CallbackToken != "" {
		args = append(args, "--callback-header", fmt.Sprintf("Authorization=Bearer %s", req.CallbackToken))
	}

	cmd := exec.CommandContext(ctx, r.markovBin, args...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		os.Remove(tmpFile.Name())
		return "", fmt.Errorf("creating stdout pipe: %w", err)
	}
	cmd.Stderr = cmd.Stdout

	if err := cmd.Start(); err != nil {
		os.Remove(tmpFile.Name())
		return "", fmt.Errorf("starting markov: %w", err)
	}

	r.mu.Lock()
	r.procs[runID] = cmd.Process
	r.mu.Unlock()

	go func() {
		defer os.Remove(tmpFile.Name())
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			log.Printf("[markov] %s", scanner.Text())
		}
		if err := cmd.Wait(); err != nil {
			log.Printf("[markov] process exited with error: %v", err)
		}
	}()

	return runID, nil
}

func (r *ShellRunner) Cancel(runID string) error {
	r.mu.Lock()
	proc, ok := r.procs[runID]
	r.mu.Unlock()
	if !ok {
		return fmt.Errorf("no process found for run %s", runID)
	}
	return proc.Kill()
}

// sanitizeForShell is unused for now but reserved for future container runner.
func sanitizeForShell(s string) string {
	return strings.ReplaceAll(s, "'", "'\\''")
}
