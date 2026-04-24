package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jctanner/markovd/internal/models"
)

func (d *DB) CreateRun(ctx context.Context, runID, workflowName string, workflowID, triggeredBy int, varsJSON string) (*models.Run, error) {
	var r models.Run
	now := time.Now()
	err := d.QueryRowContext(ctx,
		`INSERT INTO runs (run_id, workflow_id, workflow_name, status, triggered_by, vars_json, started_at)
		 VALUES ($1, $2, $3, 'running', $4, $5, $6)
		 ON CONFLICT (run_id) DO UPDATE SET
		   workflow_id = COALESCE(EXCLUDED.workflow_id, runs.workflow_id),
		   triggered_by = COALESCE(EXCLUDED.triggered_by, runs.triggered_by),
		   vars_json = COALESCE(EXCLUDED.vars_json, runs.vars_json)
		 RETURNING id, run_id, workflow_id, workflow_name, status, triggered_by, vars_json, started_at, completed_at, created_at`,
		runID, workflowID, workflowName, triggeredBy, varsJSON, now,
	).Scan(&r.ID, &r.RunID, &r.WorkflowID, &r.WorkflowName, &r.Status, &r.TriggeredBy, &r.VarsJSON, &r.StartedAt, &r.CompletedAt, &r.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("creating run: %w", err)
	}
	return &r, nil
}

func (d *DB) ListRuns(ctx context.Context) ([]models.Run, error) {
	rows, err := d.QueryContext(ctx,
		`SELECT id, run_id, workflow_id, workflow_name, status, triggered_by, vars_json, started_at, completed_at, created_at
		 FROM runs ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("listing runs: %w", err)
	}
	defer rows.Close()

	var runs []models.Run
	for rows.Next() {
		var r models.Run
		if err := rows.Scan(&r.ID, &r.RunID, &r.WorkflowID, &r.WorkflowName, &r.Status, &r.TriggeredBy, &r.VarsJSON, &r.StartedAt, &r.CompletedAt, &r.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning run: %w", err)
		}
		runs = append(runs, r)
	}
	return runs, rows.Err()
}

func (d *DB) GetRunByID(ctx context.Context, runID string) (*models.Run, error) {
	var r models.Run
	err := d.QueryRowContext(ctx,
		`SELECT id, run_id, workflow_id, workflow_name, status, triggered_by, vars_json, started_at, completed_at, created_at
		 FROM runs WHERE run_id = $1`, runID,
	).Scan(&r.ID, &r.RunID, &r.WorkflowID, &r.WorkflowName, &r.Status, &r.TriggeredBy, &r.VarsJSON, &r.StartedAt, &r.CompletedAt, &r.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("getting run: %w", err)
	}
	return &r, nil
}

func (d *DB) UpdateRunStatus(ctx context.Context, runID, status string, completedAt *time.Time) error {
	_, err := d.ExecContext(ctx,
		`UPDATE runs SET status = $1, completed_at = $2 WHERE run_id = $3`,
		status, completedAt, runID,
	)
	return err
}

func (d *DB) UpsertRunFromEvent(ctx context.Context, runID, workflowName, status string, startedAt, completedAt *time.Time) error {
	_, err := d.ExecContext(ctx,
		`INSERT INTO runs (run_id, workflow_name, status, started_at, completed_at)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (run_id) DO UPDATE SET
		   status = EXCLUDED.status,
		   started_at = COALESCE(EXCLUDED.started_at, runs.started_at),
		   completed_at = COALESCE(EXCLUDED.completed_at, runs.completed_at)`,
		runID, workflowName, status, startedAt, completedAt,
	)
	return err
}
