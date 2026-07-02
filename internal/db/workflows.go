package db

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jctanner/markovd/internal/models"
	"github.com/jctanner/markovd/internal/workflowdef"
)

func (d *DB) CreateWorkflow(ctx context.Context, name, yaml string, uploadedBy int) (*models.Workflow, error) {
	def := workflowdef.FromLegacyYAML(yaml)
	return d.CreateWorkflowDefinition(ctx, name, def, uploadedBy)
}

func (d *DB) CreateWorkflowDefinition(ctx context.Context, name string, def models.WorkflowDefinition, uploadedBy int) (*models.Workflow, error) {
	def, err := workflowdef.Normalize(def.Kind, def.Files)
	if err != nil {
		return nil, err
	}
	defJSON, err := workflowdef.MarshalFiles(def.Files)
	if err != nil {
		return nil, err
	}
	yaml := workflowdef.LegacyYAML(def)
	var w models.Workflow
	var rawFiles string
	err = d.QueryRowContext(ctx,
		`INSERT INTO workflows (name, yaml, definition_kind, definition_json, source_kind, source_root, uploaded_by)
		 VALUES ($1, $2, $3, $4::jsonb, 'manual', '', $5)
		 ON CONFLICT (name) DO UPDATE SET
		   yaml = $2,
		   definition_kind = $3,
		   definition_json = $4::jsonb,
		   source_kind = 'manual',
		   source_root = '',
		   project_id = NULL,
		   source_path = '',
		   updated_at = now()
		 RETURNING id, name, yaml, definition_kind, definition_json::text, uploaded_by, project_id, source_path, source_kind, source_root, created_at, updated_at`,
		name, yaml, def.Kind, defJSON, uploadedBy,
	).Scan(&w.ID, &w.Name, &w.YAML, &w.DefinitionKind, &rawFiles, &w.UploadedBy, &w.ProjectID, &w.SourcePath, &w.SourceKind, &w.SourceRoot, &w.CreatedAt, &w.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("creating workflow: %w", err)
	}
	return hydrateWorkflow(&w, rawFiles)
}

func (d *DB) ListWorkflows(ctx context.Context) ([]models.Workflow, error) {
	rows, err := d.QueryContext(ctx,
		`SELECT id, name, yaml, definition_kind, definition_json::text, uploaded_by, project_id, source_path, source_kind, source_root, created_at, updated_at
		 FROM workflows ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("listing workflows: %w", err)
	}
	defer rows.Close()

	var workflows []models.Workflow
	for rows.Next() {
		var w models.Workflow
		var rawFiles string
		if err := rows.Scan(&w.ID, &w.Name, &w.YAML, &w.DefinitionKind, &rawFiles, &w.UploadedBy, &w.ProjectID, &w.SourcePath, &w.SourceKind, &w.SourceRoot, &w.CreatedAt, &w.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning workflow: %w", err)
		}
		hydrated, err := hydrateWorkflow(&w, rawFiles)
		if err != nil {
			return nil, fmt.Errorf("hydrating workflow: %w", err)
		}
		workflows = append(workflows, *hydrated)
	}
	return workflows, rows.Err()
}

func (d *DB) UpdateWorkflow(ctx context.Context, name, yaml string) (*models.Workflow, error) {
	def := workflowdef.FromLegacyYAML(yaml)
	return d.UpdateWorkflowDefinition(ctx, name, def)
}

func (d *DB) UpdateWorkflowDefinition(ctx context.Context, name string, def models.WorkflowDefinition) (*models.Workflow, error) {
	def, err := workflowdef.Normalize(def.Kind, def.Files)
	if err != nil {
		return nil, err
	}
	defJSON, err := workflowdef.MarshalFiles(def.Files)
	if err != nil {
		return nil, err
	}
	yaml := workflowdef.LegacyYAML(def)
	var w models.Workflow
	var rawFiles string
	err = d.QueryRowContext(ctx,
		`UPDATE workflows
		 SET yaml = $2, definition_kind = $3, definition_json = $4::jsonb, updated_at = now()
		 WHERE name = $1
		 RETURNING id, name, yaml, definition_kind, definition_json::text, uploaded_by, project_id, source_path, source_kind, source_root, created_at, updated_at`,
		name, yaml, def.Kind, defJSON,
	).Scan(&w.ID, &w.Name, &w.YAML, &w.DefinitionKind, &rawFiles, &w.UploadedBy, &w.ProjectID, &w.SourcePath, &w.SourceKind, &w.SourceRoot, &w.CreatedAt, &w.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("updating workflow: %w", err)
	}
	return hydrateWorkflow(&w, rawFiles)
}

func (d *DB) DeleteWorkflow(ctx context.Context, name string) error {
	result, err := d.ExecContext(ctx, `DELETE FROM workflows WHERE name = $1`, name)
	if err != nil {
		return fmt.Errorf("deleting workflow: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (d *DB) GetWorkflowByName(ctx context.Context, name string) (*models.Workflow, error) {
	var w models.Workflow
	var rawFiles string
	err := d.QueryRowContext(ctx,
		`SELECT id, name, yaml, definition_kind, definition_json::text, uploaded_by, project_id, source_path, source_kind, source_root, created_at, updated_at
		 FROM workflows WHERE name = $1`, name,
	).Scan(&w.ID, &w.Name, &w.YAML, &w.DefinitionKind, &rawFiles, &w.UploadedBy, &w.ProjectID, &w.SourcePath, &w.SourceKind, &w.SourceRoot, &w.CreatedAt, &w.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("getting workflow: %w", err)
	}
	return hydrateWorkflow(&w, rawFiles)
}

func hydrateWorkflow(w *models.Workflow, rawFiles string) (*models.Workflow, error) {
	files, err := workflowdef.UnmarshalFiles(rawFiles)
	if err != nil {
		return nil, err
	}
	if w.DefinitionKind == "" {
		w.DefinitionKind = workflowdef.KindFile
	}
	if len(files) == 0 && w.YAML != "" {
		files = workflowdef.FromLegacyYAML(w.YAML).Files
	}
	w.Files = files
	if w.SourceKind == "" {
		if w.ProjectID != nil {
			w.SourceKind = "project"
		} else {
			w.SourceKind = "manual"
		}
	}
	return w, nil
}
