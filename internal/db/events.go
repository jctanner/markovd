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

// ListStepProgress returns step_progress events for one step of a run, oldest
// first. rootRunID matches events from the run and its forks; when eventRunID
// is non-empty only events from that exact run/fork ID are returned.
func (d *DB) ListStepProgress(ctx context.Context, rootRunID, eventRunID, workflow, step string, afterID, limit int) ([]models.Event, error) {
	rows, err := d.QueryContext(ctx,
		`SELECT id, run_id, event_type, payload, received_at
		 FROM events
		 WHERE event_type = 'step_progress'
		   AND (run_id = $1 OR left(run_id, length($1) + 1) = $1 || '-')
		   AND ($2 = '' OR run_id = $2)
		   AND payload->>'step_name' = $3
		   AND ($4 = '' OR payload->>'workflow_name' = $4)
		   AND id > $5
		 ORDER BY id
		 LIMIT $6`,
		rootRunID, eventRunID, step, workflow, afterID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("listing step progress: %w", err)
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
