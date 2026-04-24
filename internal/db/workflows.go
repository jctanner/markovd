package db

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jctanner/markovd/internal/models"
)

func (d *DB) CreateWorkflow(ctx context.Context, name, yaml string, uploadedBy int) (*models.Workflow, error) {
	var w models.Workflow
	err := d.QueryRowContext(ctx,
		`INSERT INTO workflows (name, yaml, uploaded_by)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (name) DO UPDATE SET yaml = $2, updated_at = now()
		 RETURNING id, name, yaml, uploaded_by, created_at, updated_at`,
		name, yaml, uploadedBy,
	).Scan(&w.ID, &w.Name, &w.YAML, &w.UploadedBy, &w.CreatedAt, &w.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("creating workflow: %w", err)
	}
	return &w, nil
}

func (d *DB) ListWorkflows(ctx context.Context) ([]models.Workflow, error) {
	rows, err := d.QueryContext(ctx,
		`SELECT id, name, yaml, uploaded_by, created_at, updated_at
		 FROM workflows ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("listing workflows: %w", err)
	}
	defer rows.Close()

	var workflows []models.Workflow
	for rows.Next() {
		var w models.Workflow
		if err := rows.Scan(&w.ID, &w.Name, &w.YAML, &w.UploadedBy, &w.CreatedAt, &w.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning workflow: %w", err)
		}
		workflows = append(workflows, w)
	}
	return workflows, rows.Err()
}

func (d *DB) GetWorkflowByName(ctx context.Context, name string) (*models.Workflow, error) {
	var w models.Workflow
	err := d.QueryRowContext(ctx,
		`SELECT id, name, yaml, uploaded_by, created_at, updated_at
		 FROM workflows WHERE name = $1`, name,
	).Scan(&w.ID, &w.Name, &w.YAML, &w.UploadedBy, &w.CreatedAt, &w.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("getting workflow: %w", err)
	}
	return &w, nil
}
