package db

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

type DB struct {
	*sql.DB
}

func New(connStr string) (*DB, error) {
	sqlDB, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("connecting to database: %w", err)
	}
	d := &DB{sqlDB}
	if err := d.migrate(); err != nil {
		return nil, fmt.Errorf("running migrations: %w", err)
	}
	return d, nil
}

func (d *DB) migrate() error {
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id          SERIAL PRIMARY KEY,
			username    TEXT UNIQUE NOT NULL,
			password    TEXT NOT NULL,
			created_at  TIMESTAMPTZ DEFAULT now()
		)`,
		`CREATE TABLE IF NOT EXISTS workflows (
			id          SERIAL PRIMARY KEY,
			name        TEXT UNIQUE NOT NULL,
			yaml        TEXT NOT NULL,
			uploaded_by INTEGER REFERENCES users(id),
			created_at  TIMESTAMPTZ DEFAULT now(),
			updated_at  TIMESTAMPTZ DEFAULT now()
		)`,
		`CREATE TABLE IF NOT EXISTS runs (
			id              SERIAL PRIMARY KEY,
			run_id          TEXT UNIQUE NOT NULL,
			workflow_id     INTEGER REFERENCES workflows(id),
			workflow_name   TEXT NOT NULL,
			status          TEXT NOT NULL DEFAULT 'pending',
			triggered_by    INTEGER REFERENCES users(id),
			vars_json       TEXT DEFAULT '{}',
			started_at      TIMESTAMPTZ,
			completed_at    TIMESTAMPTZ,
			created_at      TIMESTAMPTZ DEFAULT now()
		)`,
		`CREATE TABLE IF NOT EXISTS steps (
			id              SERIAL PRIMARY KEY,
			run_id          TEXT NOT NULL REFERENCES runs(run_id),
			fork_id         TEXT NOT NULL DEFAULT '',
			workflow_name   TEXT NOT NULL,
			step_name       TEXT NOT NULL,
			step_type       TEXT,
			status          TEXT NOT NULL DEFAULT 'pending',
			output_json     TEXT,
			error           TEXT,
			started_at      TIMESTAMPTZ,
			completed_at    TIMESTAMPTZ,
			UNIQUE(run_id, fork_id, workflow_name, step_name)
		)`,
		`DO $$ BEGIN
			IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='steps' AND column_name='fork_id') THEN
				ALTER TABLE steps ADD COLUMN fork_id TEXT NOT NULL DEFAULT '';
				ALTER TABLE steps DROP CONSTRAINT IF EXISTS steps_run_id_workflow_name_step_name_key;
				ALTER TABLE steps ADD CONSTRAINT steps_run_id_fork_id_workflow_name_step_name_key UNIQUE (run_id, fork_id, workflow_name, step_name);
			END IF;
		END $$`,
		`CREATE TABLE IF NOT EXISTS events (
			id          SERIAL PRIMARY KEY,
			run_id      TEXT NOT NULL,
			event_type  TEXT NOT NULL,
			payload     JSONB NOT NULL,
			received_at TIMESTAMPTZ DEFAULT now()
		)`,
		`CREATE TABLE IF NOT EXISTS projects (
			id              SERIAL PRIMARY KEY,
			name            TEXT UNIQUE NOT NULL,
			url             TEXT NOT NULL,
			branch          TEXT NOT NULL DEFAULT 'main',
			clone_path      TEXT NOT NULL DEFAULT '',
			last_synced_at  TIMESTAMPTZ,
			sync_status     TEXT NOT NULL DEFAULT 'idle',
			sync_error      TEXT NOT NULL DEFAULT '',
			created_by      INTEGER REFERENCES users(id),
			created_at      TIMESTAMPTZ DEFAULT now(),
			updated_at      TIMESTAMPTZ DEFAULT now()
		)`,
		`DO $$ BEGIN
			IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='workflows' AND column_name='project_id') THEN
				ALTER TABLE workflows ADD COLUMN project_id INTEGER REFERENCES projects(id) ON DELETE SET NULL;
				ALTER TABLE workflows ADD COLUMN source_path TEXT NOT NULL DEFAULT '';
			END IF;
		END $$`,
	}
	for _, m := range migrations {
		if _, err := d.Exec(m); err != nil {
			return fmt.Errorf("migration failed: %w\nSQL: %s", err, m)
		}
	}
	return nil
}
