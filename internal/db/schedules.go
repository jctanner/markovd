package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/jctanner/markovd/internal/models"
)

var ErrScheduleBusy = errors.New("schedule is dispatching or its last run is still active")

const scheduleColumns = `id,name,kind,cron,timezone,run_at,next_run_at,state,request_json,workflow_json,source_commit,created_by,last_run_id,last_error,created_at,updated_at`

type scanner interface{ Scan(...any) error }

func scanSchedule(row scanner) (*models.Schedule, error) {
	var s models.Schedule
	var wf []byte
	err := row.Scan(&s.ID, &s.Name, &s.Kind, &s.Cron, &s.Timezone, &s.RunAt, &s.NextRunAt, &s.State, &s.Request, &wf, &s.SourceCommit, &s.CreatedBy, &s.LastRunID, &s.LastError, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if err = json.Unmarshal(wf, &s.Workflow); err != nil {
		return nil, err
	}
	return &s, nil
}

func (d *DB) ListSchedules(ctx context.Context) ([]models.Schedule, error) {
	rows, err := d.QueryContext(ctx, `SELECT `+scheduleColumns+` FROM schedules ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.Schedule{}
	for rows.Next() {
		s, err := scanSchedule(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *s)
	}
	return out, rows.Err()
}

func (d *DB) GetSchedule(ctx context.Context, id int) (*models.Schedule, error) {
	s, err := scanSchedule(d.QueryRowContext(ctx, `SELECT `+scheduleColumns+` FROM schedules WHERE id=$1`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return s, err
}

func (d *DB) SaveSchedule(ctx context.Context, s *models.Schedule) (*models.Schedule, error) {
	wf, err := json.Marshal(s.Workflow)
	if err != nil {
		return nil, err
	}
	if s.ID == 0 {
		return scanSchedule(d.QueryRowContext(ctx, `INSERT INTO schedules(name,kind,cron,timezone,run_at,next_run_at,state,request_json,workflow_json,source_commit,created_by)
 VALUES($1,$2,$3,$4,$5,$6,'active',$7,$8,$9,$10) RETURNING `+scheduleColumns,
			s.Name, s.Kind, s.Cron, s.Timezone, s.RunAt, s.NextRunAt, []byte(s.Request), wf, s.SourceCommit, s.CreatedBy))
	}
	saved, err := scanSchedule(d.QueryRowContext(ctx, `UPDATE schedules SET name=$2,kind=$3,cron=$4,timezone=$5,run_at=$6,next_run_at=$7,request_json=$8,workflow_json=$9,source_commit=$10,last_error='',updated_at=now()
 WHERE id=$1 AND state IN ('active','paused') RETURNING `+scheduleColumns,
		s.ID, s.Name, s.Kind, s.Cron, s.Timezone, s.RunAt, s.NextRunAt, []byte(s.Request), wf, s.SourceCommit))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrScheduleBusy
	}
	return saved, err
}

func (d *DB) SetScheduleState(ctx context.Context, id int, state string, next *time.Time) error {
	res, err := d.ExecContext(ctx, `UPDATE schedules SET state=$2,next_run_at=COALESCE($3,next_run_at),updated_at=now()
 WHERE id=$1 AND state IN ('active','paused')`, id, state, next)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrScheduleBusy
	}
	return nil
}

func (d *DB) DeleteSchedule(ctx context.Context, id int) error {
	res, err := d.ExecContext(ctx, `DELETE FROM schedules WHERE id=$1 AND state != 'dispatching'`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrScheduleBusy
	}
	return nil
}

// ClaimDueSchedule commits the occurrence and stable run identity before any external
// effect. Concurrent scheduler processes cannot claim the same occurrence.
func (d *DB) ClaimDueSchedule(ctx context.Context, now time.Time, runID string) (*models.Schedule, error) {
	tx, err := d.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	s, err := scanSchedule(tx.QueryRowContext(ctx, `SELECT `+scheduleColumns+` FROM schedules s
 WHERE state='active' AND next_run_at <= $1
 AND NOT EXISTS (SELECT 1 FROM runs r WHERE r.run_id=s.last_run_id AND r.status IN ('pending','running','paused'))
		ORDER BY next_run_at,id LIMIT 1 FOR UPDATE SKIP LOCKED`, now))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var req struct {
		Vars    map[string]string `json:"vars"`
		Volumes json.RawMessage   `json:"volumes"`
		Secrets json.RawMessage   `json:"secret_volumes"`
	}
	if err = json.Unmarshal(s.Request, &req); err != nil {
		return nil, err
	}
	vars, _ := json.Marshal(req.Vars)
	if len(req.Volumes) == 0 {
		req.Volumes = json.RawMessage(`[]`)
	}
	if len(req.Secrets) == 0 {
		req.Secrets = json.RawMessage(`[]`)
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO runs(run_id,workflow_id,workflow_name,workflow_source_commit,status,triggered_by,vars_json,volumes_json,secret_volumes_json)
 VALUES($1,(SELECT id FROM workflows WHERE id=$2),$3,$4,'pending',$5,$6,$7,$8)`, runID, s.Workflow.ID, s.Workflow.Name, s.SourceCommit, s.CreatedBy, string(vars), string(req.Volumes), string(req.Secrets))
	if err != nil {
		return nil, err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO schedule_occurrences(schedule_id,scheduled_for,run_id,state) VALUES($1,$2,$3,'dispatching')`, s.ID, s.NextRunAt, runID)
	if err != nil {
		return nil, err
	}
	_, err = tx.ExecContext(ctx, `UPDATE schedules SET state='dispatching',last_run_id=$2,last_error='',updated_at=now() WHERE id=$1`, s.ID, runID)
	if err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	s.State = "dispatching"
	s.LastRunID = runID
	return s, nil
}

func (d *DB) FinishScheduleDispatch(ctx context.Context, s *models.Schedule, next *time.Time, launchErr string) error {
	tx, err := d.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	state := "completed"
	occurrence := "dispatched"
	if s.Kind == "cron" {
		state = "active"
	}
	if launchErr != "" {
		state = "failed"
		occurrence = "uncertain"
	}
	res, err := tx.ExecContext(ctx, `UPDATE schedules SET state=$3,next_run_at=$4,last_error=$5,updated_at=now() WHERE id=$1 AND last_run_id=$2 AND state='dispatching'`, s.ID, s.LastRunID, state, next, launchErr)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrScheduleBusy
	}
	_, err = tx.ExecContext(ctx, `UPDATE schedule_occurrences SET state=$2,error=$3 WHERE run_id=$1`, s.LastRunID, occurrence, launchErr)
	if err != nil {
		return err
	}
	// Preserve callback statuses, including fast completion before Start returns.
	if launchErr == "" {
		_, err = tx.ExecContext(ctx, `UPDATE runs SET status='running',started_at=COALESCE(started_at,now()) WHERE run_id=$1 AND status='pending'`, s.LastRunID)
	}
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (d *DB) StaleScheduleDispatches(ctx context.Context, before time.Time) ([]models.Schedule, error) {
	rows, err := d.QueryContext(ctx, `SELECT `+scheduleColumns+` FROM schedules WHERE state='dispatching' AND updated_at < $1`, before)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.Schedule{}
	for rows.Next() {
		s, err := scanSchedule(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *s)
	}
	return out, rows.Err()
}

func (d *DB) ScheduleOccurrences(ctx context.Context, id int) ([]models.ScheduleOccurrence, error) {
	rows, err := d.QueryContext(ctx, `SELECT id,schedule_id,scheduled_for,run_id,state,error,created_at FROM schedule_occurrences WHERE schedule_id=$1 ORDER BY id DESC LIMIT 100`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.ScheduleOccurrence{}
	for rows.Next() {
		var o models.ScheduleOccurrence
		if err = rows.Scan(&o.ID, &o.ScheduleID, &o.ScheduledFor, &o.RunID, &o.State, &o.Error, &o.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, rows.Err()
}
