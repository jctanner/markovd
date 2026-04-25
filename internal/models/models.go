package models

import "time"

type User struct {
	ID        int       `json:"id"`
	Username  string    `json:"username"`
	Password  string    `json:"-"`
	CreatedAt time.Time `json:"created_at"`
}

type Workflow struct {
	ID         int       `json:"id"`
	Name       string    `json:"name"`
	YAML       string    `json:"yaml"`
	UploadedBy int       `json:"uploaded_by"`
	ProjectID  *int      `json:"project_id,omitempty"`
	SourcePath string    `json:"source_path,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type Project struct {
	ID           int        `json:"id"`
	Name         string     `json:"name"`
	URL          string     `json:"url"`
	Branch       string     `json:"branch"`
	LastSyncedAt *time.Time `json:"last_synced_at,omitempty"`
	SyncStatus   string     `json:"sync_status"`
	SyncError    string     `json:"sync_error,omitempty"`
	CreatedBy    int        `json:"created_by"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

type ProjectFile struct {
	Path     string `json:"path"`
	Imported bool   `json:"imported"`
}

type Run struct {
	ID                int        `json:"id"`
	RunID             string     `json:"run_id"`
	WorkflowID        *int       `json:"workflow_id,omitempty"`
	WorkflowName      string     `json:"workflow_name"`
	Status            string     `json:"status"`
	TriggeredBy       *int       `json:"triggered_by,omitempty"`
	VarsJSON          string     `json:"vars_json"`
	VolumesJSON       string     `json:"volumes_json"`
	SecretVolumesJSON string     `json:"secret_volumes_json"`
	StartedAt         *time.Time `json:"started_at,omitempty"`
	CompletedAt       *time.Time `json:"completed_at,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
}

type Step struct {
	ID           int        `json:"id"`
	RunID        string     `json:"run_id"`
	ForkID       string     `json:"fork_id,omitempty"`
	WorkflowName string     `json:"workflow_name"`
	StepName     string     `json:"step_name"`
	StepType     string     `json:"step_type,omitempty"`
	Status       string     `json:"status"`
	OutputJSON   string     `json:"output_json,omitempty"`
	Error        string     `json:"error,omitempty"`
	StartedAt    *time.Time `json:"started_at,omitempty"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
}

type Event struct {
	ID         int       `json:"id"`
	RunID      string    `json:"run_id"`
	EventType  string    `json:"event_type"`
	Payload    string    `json:"payload"`
	ReceivedAt time.Time `json:"received_at"`
}

type UserPreferences struct {
	ID             int       `json:"id"`
	UserID         int       `json:"user_id"`
	DefaultVolumes string    `json:"default_volumes"`
	DefaultSecrets string    `json:"default_secrets"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}
