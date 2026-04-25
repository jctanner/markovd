package runner

import (
	"context"
	"io"
	"strings"
)

type PVCMount struct {
	Name      string `json:"name"`
	PVC       string `json:"pvc"`
	MountPath string `json:"mount_path"`
	ReadOnly  bool   `json:"read_only,omitempty"`
}

type SecretMount struct {
	Name      string `json:"name"`
	Secret    string `json:"secret"`
	MountPath string `json:"mount_path"`
	ReadOnly  bool   `json:"read_only,omitempty"`
}

type RunRequest struct {
	WorkflowYAML  string
	Vars          map[string]string
	CallbackURL   string
	CallbackToken string
	Debug         bool
	Volumes       []PVCMount
	SecretVolumes []SecretMount
}

type PVCInfo struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

type SecretInfo struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type Runner interface {
	Start(ctx context.Context, req RunRequest) (runID string, err error)
	Cancel(runID string) error
	ListPVCs(ctx context.Context) ([]PVCInfo, error)
	ListSecrets(ctx context.Context) ([]SecretInfo, error)
	GetJobLogs(ctx context.Context, jobName string) (string, error)
	StreamJobLogs(ctx context.Context, jobName string) (io.ReadCloser, error)
}

func ParseVolumes(s string) []PVCMount {
	if s == "" {
		return nil
	}
	var volumes []PVCMount
	for _, entry := range strings.Split(s, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		parts := strings.SplitN(entry, ":", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			continue
		}
		volumes = append(volumes, PVCMount{
			Name:      strings.ReplaceAll(parts[0], "/", "-"),
			PVC:       parts[0],
			MountPath: parts[1],
		})
	}
	return volumes
}

func ParseSecretMounts(s string) []SecretMount {
	if s == "" {
		return nil
	}
	var mounts []SecretMount
	for _, entry := range strings.Split(s, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		parts := strings.SplitN(entry, ":", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			continue
		}
		mounts = append(mounts, SecretMount{
			Name:      strings.ReplaceAll(parts[0], "/", "-"),
			Secret:    parts[0],
			MountPath: parts[1],
		})
	}
	return mounts
}
