package runner

import "context"

type RunRequest struct {
	WorkflowYAML  string
	Vars          map[string]string
	CallbackURL   string
	CallbackToken string
	Debug         bool
}

type Runner interface {
	Start(ctx context.Context, req RunRequest) (runID string, err error)
	Cancel(runID string) error
}
