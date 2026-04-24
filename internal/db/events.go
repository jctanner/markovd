package db

import (
	"context"
	"fmt"

	"github.com/jctanner/markovd/internal/models"
)

func (d *DB) InsertEvent(ctx context.Context, runID, eventType, payload string) (*models.Event, error) {
	var e models.Event
	err := d.QueryRowContext(ctx,
		`INSERT INTO events (run_id, event_type, payload)
		 VALUES ($1, $2, $3::jsonb)
		 RETURNING id, run_id, event_type, payload, received_at`,
		runID, eventType, payload,
	).Scan(&e.ID, &e.RunID, &e.EventType, &e.Payload, &e.ReceivedAt)
	if err != nil {
		return nil, fmt.Errorf("inserting event: %w", err)
	}
	return &e, nil
}

func (d *DB) GetEventsByRunID(ctx context.Context, runID string) ([]models.Event, error) {
	rows, err := d.QueryContext(ctx,
		`SELECT id, run_id, event_type, payload, received_at
		 FROM events WHERE run_id = $1 ORDER BY received_at`, runID,
	)
	if err != nil {
		return nil, fmt.Errorf("listing events: %w", err)
	}
	defer rows.Close()

	var events []models.Event
	for rows.Next() {
		var e models.Event
		if err := rows.Scan(&e.ID, &e.RunID, &e.EventType, &e.Payload, &e.ReceivedAt); err != nil {
			return nil, fmt.Errorf("scanning event: %w", err)
		}
		events = append(events, e)
	}
	return events, rows.Err()
}
