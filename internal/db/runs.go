package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jctanner/markovd/internal/models"
)

func (d *DB) CreateRun(ctx context.Context, runID, workflowName string, workflowID, triggeredBy int, varsJSON, volumesJSON, secretVolumesJSON string) (*models.Run, error) {
	var r models.Run
	now := time.Now()
	err := d.QueryRowContext(ctx,
		`INSERT INTO runs (run_id, workflow_id, workflow_name, status, triggered_by, vars_json, volumes_json, secret_volumes_json, started_at)
		 VALUES ($1, $2, $3, 'running', $4, $5, $6, $7, $8)
		 ON CONFLICT (run_id) DO UPDATE SET
		   workflow_id = COALESCE(EXCLUDED.workflow_id, runs.workflow_id),
		   triggered_by = COALESCE(EXCLUDED.triggered_by, runs.triggered_by),
		   vars_json = COALESCE(EXCLUDED.vars_json, runs.vars_json),
		   volumes_json = COALESCE(EXCLUDED.volumes_json, runs.volumes_json),
		   secret_volumes_json = COALESCE(EXCLUDED.secret_volumes_json, runs.secret_volumes_json)
		 RETURNING id, run_id, workflow_id, workflow_name, status, triggered_by, vars_json, volumes_json, secret_volumes_json, started_at, completed_at, created_at`,
		runID, workflowID, workflowName, triggeredBy, varsJSON, volumesJSON, secretVolumesJSON, now,
	).Scan(&r.ID, &r.RunID, &r.WorkflowID, &r.WorkflowName, &r.Status, &r.TriggeredBy, &r.VarsJSON, &r.VolumesJSON, &r.SecretVolumesJSON, &r.StartedAt, &r.CompletedAt, &r.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("creating run: %w", err)
	}
	return &r, nil
}

func (d *DB) ListRuns(ctx context.Context) ([]models.Run, error) {
	rows, err := d.QueryContext(ctx,
		`SELECT id, run_id, workflow_id, workflow_name, status, triggered_by, vars_json, volumes_json, secret_volumes_json, started_at, completed_at, created_at
		 FROM runs ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("listing runs: %w", err)
	}
	defer rows.Close()

	var runs []models.Run
	for rows.Next() {
		var r models.Run
		if err := rows.Scan(&r.ID, &r.RunID, &r.WorkflowID, &r.WorkflowName, &r.Status, &r.TriggeredBy, &r.VarsJSON, &r.VolumesJSON, &r.SecretVolumesJSON, &r.StartedAt, &r.CompletedAt, &r.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning run: %w", err)
		}
		runs = append(runs, r)
	}
	return runs, rows.Err()
}

func (d *DB) GetRunByID(ctx context.Context, runID string) (*models.Run, error) {
	var r models.Run
	err := d.QueryRowContext(ctx,
		`SELECT id, run_id, workflow_id, workflow_name, status, triggered_by, vars_json, volumes_json, secret_volumes_json, started_at, completed_at, created_at
		 FROM runs WHERE run_id = $1`, runID,
	).Scan(&r.ID, &r.RunID, &r.WorkflowID, &r.WorkflowName, &r.Status, &r.TriggeredBy, &r.VarsJSON, &r.VolumesJSON, &r.SecretVolumesJSON, &r.StartedAt, &r.CompletedAt, &r.CreatedAt)
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

func (d *DB) DeleteRun(ctx context.Context, runID string) error {
	tx, err := d.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback()

	for _, q := range []string{
		`DELETE FROM steps WHERE run_id = $1`,
		`DELETE FROM events WHERE run_id = $1`,
		`DELETE FROM runs WHERE run_id = $1`,
	} {
		if _, err := tx.ExecContext(ctx, q, runID); err != nil {
			return fmt.Errorf("deleting run data: %w", err)
		}
	}

	return tx.Commit()
}

func (d *DB) CountActiveJobs(ctx context.Context) (running, pending int, err error) {
	err = d.QueryRowContext(ctx,
		`SELECT
			COUNT(*) FILTER (WHERE status = 'running'),
			COUNT(*) FILTER (WHERE status = 'pending')
		FROM (
			SELECT status FROM runs WHERE status IN ('running', 'pending')
			UNION ALL
			SELECT status FROM steps
			WHERE status IN ('running', 'pending')
			  AND COALESCE(output_json::jsonb->>'job_name', '') != ''
		) sub`,
	).Scan(&running, &pending)
	return
}

func (d *DB) ListActiveJobs(ctx context.Context) ([]models.ActiveJob, error) {
	rows, err := d.QueryContext(ctx,
		`SELECT kind, run_id, fork_id, workflow_name, step_name, step_type, status, job_name, started_at
		FROM (
			SELECT 'run' AS kind, run_id, '' AS fork_id, workflow_name,
			       '' AS step_name, '' AS step_type, status, run_id AS job_name, started_at
			FROM runs WHERE status IN ('running', 'pending')
			UNION ALL
			SELECT 'step' AS kind, run_id, COALESCE(fork_id, ''), workflow_name,
			       step_name, COALESCE(step_type, ''), status,
			       COALESCE(output_json::jsonb->>'job_name', ''), started_at
			FROM steps
			WHERE status IN ('running', 'pending')
			  AND COALESCE(output_json::jsonb->>'job_name', '') != ''
		) sub ORDER BY started_at DESC NULLS LAST`)
	if err != nil {
		return nil, fmt.Errorf("listing active jobs: %w", err)
	}
	defer rows.Close()

	var jobs []models.ActiveJob
	for rows.Next() {
		var j models.ActiveJob
		if err := rows.Scan(&j.Kind, &j.RunID, &j.ForkID, &j.WorkflowName, &j.StepName, &j.StepType, &j.Status, &j.JobName, &j.StartedAt); err != nil {
			return nil, fmt.Errorf("scanning active job: %w", err)
		}
		jobs = append(jobs, j)
	}
	return jobs, rows.Err()
}

func (d *DB) GetConcurrencyHistory(ctx context.Context) ([]models.ConcurrencyBucket, error) {
	rows, err := d.QueryContext(ctx,
		`WITH buckets AS (
			SELECT generate_series(
				date_trunc('minute', now() - interval '24 hours'),
				now(),
				interval '15 minutes'
			) AS t
		),
		jobs AS (
			SELECT started_at, completed_at FROM runs
			WHERE started_at > now() - interval '25 hours'
			UNION ALL
			SELECT started_at, completed_at FROM steps
			WHERE started_at > now() - interval '25 hours'
			  AND COALESCE(output_json::jsonb->>'job_name', '') != ''
		)
		SELECT b.t, COUNT(j.started_at)
		FROM buckets b
		LEFT JOIN jobs j ON j.started_at < b.t + interval '15 minutes'
		  AND (j.completed_at IS NULL OR j.completed_at >= b.t)
		GROUP BY b.t
		ORDER BY b.t`)
	if err != nil {
		return nil, fmt.Errorf("concurrency history: %w", err)
	}
	defer rows.Close()

	var buckets []models.ConcurrencyBucket
	for rows.Next() {
		var b models.ConcurrencyBucket
		if err := rows.Scan(&b.T, &b.Count); err != nil {
			return nil, fmt.Errorf("scanning bucket: %w", err)
		}
		buckets = append(buckets, b)
	}
	return buckets, rows.Err()
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
