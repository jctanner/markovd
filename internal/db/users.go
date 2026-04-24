package db

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jctanner/markovd/internal/models"
)

func (d *DB) CreateUser(ctx context.Context, username, passwordHash string) (*models.User, error) {
	var u models.User
	err := d.QueryRowContext(ctx,
		`INSERT INTO users (username, password) VALUES ($1, $2)
		 RETURNING id, username, password, created_at`,
		username, passwordHash,
	).Scan(&u.ID, &u.Username, &u.Password, &u.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("creating user: %w", err)
	}
	return &u, nil
}

func (d *DB) GetUserByUsername(ctx context.Context, username string) (*models.User, error) {
	var u models.User
	err := d.QueryRowContext(ctx,
		`SELECT id, username, password, created_at FROM users WHERE username = $1`,
		username,
	).Scan(&u.ID, &u.Username, &u.Password, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("getting user: %w", err)
	}
	return &u, nil
}

func (d *DB) CountUsers(ctx context.Context) (int, error) {
	var count int
	err := d.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&count)
	return count, err
}
