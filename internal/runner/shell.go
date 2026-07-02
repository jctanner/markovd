package runner

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"sync"

	"github.com/jctanner/markovd/internal/workflowdef"
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
	m, err := workflowdef.Materialize(req.WorkflowDefinition())
	if err != nil {
		return "", fmt.Errorf("materializing workflow: %w", err)
	}

	runID := generateRunID()
	args := []string{"run", m.Path, "--verbose", "--run-id", runID}
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

	cmd := exec.CommandContext(context.Background(), r.markovBin, args...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		m.Cleanup()
		return "", fmt.Errorf("creating stdout pipe: %w", err)
	}
	cmd.Stderr = cmd.Stdout

	if err := cmd.Start(); err != nil {
		m.Cleanup()
		return "", fmt.Errorf("starting markov: %w", err)
	}

	r.mu.Lock()
	r.procs[runID] = cmd.Process
	r.mu.Unlock()

	go func() {
		defer m.Cleanup()
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

func (r *ShellRunner) ListPVCs(ctx context.Context) ([]PVCInfo, error) {
	return nil, nil
}

func (r *ShellRunner) ListSecrets(ctx context.Context) ([]SecretInfo, error) {
	return nil, nil
}

func (r *ShellRunner) GetJobLogs(ctx context.Context, jobName string) (string, error) {
	return "", fmt.Errorf("job logs not available in shell mode")
}

func (r *ShellRunner) StreamJobLogs(ctx context.Context, jobName string) (io.ReadCloser, error) {
	return nil, fmt.Errorf("log streaming not available in shell mode")
}

func (r *ShellRunner) AuditJobStatuses(ctx context.Context) (map[string]string, error) {
	return nil, nil
}
