package db

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jctanner/markovd/internal/models"
)

func (d *DB) GetStepsByRunID(ctx context.Context, runID string) ([]models.Step, error) {
	rows, err := d.QueryContext(ctx,
		`SELECT id, run_id, COALESCE(fork_id, ''), workflow_name, step_name, COALESCE(step_type, ''), status,
		        COALESCE(output_json, ''), COALESCE(error, ''), started_at, completed_at, updated_at
		 FROM steps WHERE run_id = $1 ORDER BY id`, runID,
	)
	if err != nil {
		return nil, fmt.Errorf("listing steps: %w", err)
	}
	defer rows.Close()

	var steps []models.Step
	for rows.Next() {
		var s models.Step
		if err := rows.Scan(&s.ID, &s.RunID, &s.ForkID, &s.WorkflowName, &s.StepName, &s.StepType, &s.Status,
			&s.OutputJSON, &s.Error, &s.StartedAt, &s.CompletedAt, &s.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning step: %w", err)
		}
		steps = append(steps, s)
	}
	return steps, rows.Err()
}

func (d *DB) GetStepsUpdatedSince(ctx context.Context, runID string, since time.Time) ([]models.Step, error) {
	rows, err := d.QueryContext(ctx,
		`SELECT id, run_id, COALESCE(fork_id, ''), workflow_name, step_name, COALESCE(step_type, ''), status,
		        COALESCE(output_json, ''), COALESCE(error, ''), started_at, completed_at, updated_at
		 FROM steps WHERE run_id = $1 AND updated_at > $2 ORDER BY id`, runID, since,
	)
	if err != nil {
		return nil, fmt.Errorf("listing updated steps: %w", err)
	}
	defer rows.Close()

	var steps []models.Step
	for rows.Next() {
		var s models.Step
		if err := rows.Scan(&s.ID, &s.RunID, &s.ForkID, &s.WorkflowName, &s.StepName, &s.StepType, &s.Status,
			&s.OutputJSON, &s.Error, &s.StartedAt, &s.CompletedAt, &s.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning step: %w", err)
		}
		steps = append(steps, s)
	}
	return steps, rows.Err()
}

func (d *DB) UpsertStep(ctx context.Context, runID, forkID, workflowName, stepName, stepType, status, outputJSON, stepError string, startedAt, completedAt *time.Time) error {
	_, err := d.ExecContext(ctx,
		`INSERT INTO steps (run_id, fork_id, workflow_name, step_name, step_type, status, output_json, error, started_at, completed_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, now())
		 ON CONFLICT (run_id, fork_id, workflow_name, step_name) DO UPDATE SET
		   step_type = COALESCE(EXCLUDED.step_type, steps.step_type),
		   status = EXCLUDED.status,
		   output_json = COALESCE(EXCLUDED.output_json, steps.output_json),
		   error = COALESCE(EXCLUDED.error, steps.error),
		   started_at = COALESCE(EXCLUDED.started_at, steps.started_at),
		   completed_at = COALESCE(EXCLUDED.completed_at, steps.completed_at),
		   updated_at = now()`,
		runID, forkID, workflowName, stepName, stepType, status, nullIfEmpty(outputJSON), nullIfEmpty(stepError), startedAt, completedAt,
	)
	return err
}

func (d *DB) FindCachedJobLogs(ctx context.Context, jobName string) (string, bool) {
	var outputJSON string
	err := d.QueryRowContext(ctx,
		`SELECT COALESCE(output_json, '') FROM steps
		 WHERE output_json LIKE $1 LIMIT 1`,
		fmt.Sprintf("%%\"job_name\":\"%s\"%%", jobName),
	).Scan(&outputJSON)
	if err != nil || outputJSON == "" {
		return "", false
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(outputJSON), &parsed); err != nil {
		return "", false
	}
	if logs, ok := parsed["logs"].(string); ok && logs != "" {
		return logs, true
	}
	return "", false
}

func nullIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
