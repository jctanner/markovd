package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jctanner/markovd/internal/models"
)

func (d *DB) CreateProject(ctx context.Context, name, url, branch, clonePath string, createdBy int) (*models.Project, error) {
	var p models.Project
	err := d.QueryRowContext(ctx,
		`INSERT INTO projects (name, url, branch, clone_path, created_by)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, name, url, branch, clone_path, last_synced_at, sync_status, sync_error, created_by, created_at, updated_at`,
		name, url, branch, clonePath, createdBy,
	).Scan(&p.ID, &p.Name, &p.URL, &p.Branch, &clonePath, &p.LastSyncedAt, &p.SyncStatus, &p.SyncError, &p.CreatedBy, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("creating project: %w", err)
	}
	return &p, nil
}

func (d *DB) ListProjects(ctx context.Context) ([]models.Project, error) {
	rows, err := d.QueryContext(ctx,
		`SELECT id, name, url, branch, last_synced_at, sync_status, sync_error, created_by, created_at, updated_at
		 FROM projects ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("listing projects: %w", err)
	}
	defer rows.Close()

	var projects []models.Project
	for rows.Next() {
		var p models.Project
		if err := rows.Scan(&p.ID, &p.Name, &p.URL, &p.Branch, &p.LastSyncedAt, &p.SyncStatus, &p.SyncError, &p.CreatedBy, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning project: %w", err)
		}
		projects = append(projects, p)
	}
	return projects, rows.Err()
}

func (d *DB) GetProject(ctx context.Context, id int) (*models.Project, error) {
	var p models.Project
	var clonePath string
	err := d.QueryRowContext(ctx,
		`SELECT id, name, url, branch, clone_path, last_synced_at, sync_status, sync_error, created_by, created_at, updated_at
		 FROM projects WHERE id = $1`, id,
	).Scan(&p.ID, &p.Name, &p.URL, &p.Branch, &clonePath, &p.LastSyncedAt, &p.SyncStatus, &p.SyncError, &p.CreatedBy, &p.CreatedAt, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("getting project: %w", err)
	}
	return &p, nil
}

func (d *DB) GetProjectClonePath(ctx context.Context, id int) (string, error) {
	var clonePath string
	err := d.QueryRowContext(ctx, `SELECT clone_path FROM projects WHERE id = $1`, id).Scan(&clonePath)
	if err != nil {
		return "", fmt.Errorf("getting project clone path: %w", err)
	}
	return clonePath, nil
}

func (d *DB) DeleteProject(ctx context.Context, id int) error {
	result, err := d.ExecContext(ctx, `DELETE FROM projects WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("deleting project: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (d *DB) UpdateProjectSyncStatus(ctx context.Context, id int, status, syncError string, syncedAt *time.Time) error {
	_, err := d.ExecContext(ctx,
		`UPDATE projects SET sync_status = $2, sync_error = $3, last_synced_at = COALESCE($4, last_synced_at), updated_at = now()
		 WHERE id = $1`,
		id, status, syncError, syncedAt,
	)
	if err != nil {
		return fmt.Errorf("updating project sync status: %w", err)
	}
	return nil
}

func (d *DB) ImportProjectWorkflow(ctx context.Context, projectID int, name, yaml, sourcePath string, uploadedBy int) (*models.Workflow, error) {
	var w models.Workflow
	err := d.QueryRowContext(ctx,
		`INSERT INTO workflows (name, yaml, uploaded_by, project_id, source_path)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (name) DO UPDATE SET yaml = $2, project_id = $4, source_path = $5, updated_at = now()
		 RETURNING id, name, yaml, uploaded_by, project_id, source_path, created_at, updated_at`,
		name, yaml, uploadedBy, projectID, sourcePath,
	).Scan(&w.ID, &w.Name, &w.YAML, &w.UploadedBy, &w.ProjectID, &w.SourcePath, &w.CreatedAt, &w.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("importing project workflow: %w", err)
	}
	return &w, nil
}

func (d *DB) GetWorkflowsByProjectID(ctx context.Context, projectID int) ([]models.Workflow, error) {
	rows, err := d.QueryContext(ctx,
		`SELECT id, name, yaml, uploaded_by, project_id, source_path, created_at, updated_at
		 FROM workflows WHERE project_id = $1 ORDER BY name`, projectID,
	)
	if err != nil {
		return nil, fmt.Errorf("listing project workflows: %w", err)
	}
	defer rows.Close()

	var workflows []models.Workflow
	for rows.Next() {
		var w models.Workflow
		if err := rows.Scan(&w.ID, &w.Name, &w.YAML, &w.UploadedBy, &w.ProjectID, &w.SourcePath, &w.CreatedAt, &w.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning project workflow: %w", err)
		}
		workflows = append(workflows, w)
	}
	return workflows, rows.Err()
}
