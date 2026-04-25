package db

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jctanner/markovd/internal/models"
)

func (d *DB) GetPreferences(ctx context.Context, userID int) (*models.UserPreferences, error) {
	var p models.UserPreferences
	err := d.QueryRowContext(ctx,
		`SELECT id, user_id, default_volumes, default_secrets, created_at, updated_at
		 FROM user_preferences WHERE user_id = $1`, userID,
	).Scan(&p.ID, &p.UserID, &p.DefaultVolumes, &p.DefaultSecrets, &p.CreatedAt, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("getting preferences: %w", err)
	}
	return &p, nil
}

func (d *DB) UpsertPreferences(ctx context.Context, userID int, defaultVolumes, defaultSecrets string) (*models.UserPreferences, error) {
	var p models.UserPreferences
	err := d.QueryRowContext(ctx,
		`INSERT INTO user_preferences (user_id, default_volumes, default_secrets)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (user_id) DO UPDATE SET
		   default_volumes = EXCLUDED.default_volumes,
		   default_secrets = EXCLUDED.default_secrets,
		   updated_at = now()
		 RETURNING id, user_id, default_volumes, default_secrets, created_at, updated_at`,
		userID, defaultVolumes, defaultSecrets,
	).Scan(&p.ID, &p.UserID, &p.DefaultVolumes, &p.DefaultSecrets, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("upserting preferences: %w", err)
	}
	return &p, nil
}
