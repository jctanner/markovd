package runner

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jctanner/markovd/internal/models"
	"github.com/jctanner/markovd/internal/workflowdef"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func dirWorkflow() RunRequest {
	return RunRequest{
		Workflow: models.WorkflowDefinition{
			Kind: workflowdef.KindDirectory,
			Files: []models.WorkflowDefinitionFile{
				{Path: "meta.yaml", Content: "entrypoint: main\n"},
				{Path: "vars.yaml", Content: "{}\n"},
				{Path: "rules.yaml", Content: "[]\n"},
				{Path: "step_types.yaml", Content: "{}\n"},
				{Path: "workflows/main.yaml", Content: "name: main\nsteps: []\n"},
				{Path: "scripts/check.py", Content: "print('ok')\n"},
			},
		},
		Vars:        map[string]string{"b": "2", "a": "1"},
		CallbackURL: "http://markovd/api/v1/events",
	}
}

func TestStartWithDataVolumeWritesFilesAndMountsRunDirectory(t *testing.T) {
	r := newTestRunner(nil)
	dataDir := t.TempDir()
	r.SetDataVolume("markovd-data", dataDir)
	ctx := context.Background()

	runID, err := r.Start(ctx, dirWorkflow())
	if err != nil {
		t.Fatalf("Start() error: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(dataDir, "runs", runID, "workflow", "scripts", "check.py"))
	if err != nil || string(got) != "print('ok')\n" {
		t.Fatalf("run file = %q, %v", got, err)
	}
	if _, err := r.client.CoreV1().ConfigMaps("ai-pipeline").Get(ctx, runID+"-workflow", metav1.GetOptions{}); err == nil {
		t.Fatal("no ConfigMap expected with a data volume")
	}

	job, err := r.client.BatchV1().Jobs("ai-pipeline").Get(ctx, runID, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Job not created: %v", err)
	}
	c := job.Spec.Template.Spec.Containers[0]
	if c.Args[0] != "run" || c.Args[1] != "/run-data/workflow" {
		t.Fatalf("args = %v", c.Args)
	}
	if !strings.Contains(strings.Join(c.Args, " "), "--var a=1 --var b=2") {
		t.Fatalf("vars not passed in order: %v", c.Args)
	}
	if len(c.VolumeMounts) != 1 || c.VolumeMounts[0].SubPath != "runs/"+runID || c.VolumeMounts[0].MountPath != "/run-data" {
		t.Fatalf("mounts = %#v", c.VolumeMounts)
	}
	if v := job.Spec.Template.Spec.Volumes[0]; v.PersistentVolumeClaim == nil || v.PersistentVolumeClaim.ClaimName != "markovd-data" {
		t.Fatalf("volume = %#v", v)
	}
	if len(c.Env) != 1 || c.Env[0].Name != "MARKOV_STATE_STORE" || c.Env[0].Value != "/run-data/markov-state.db" {
		t.Fatalf("env = %#v", c.Env)
	}
}

func TestResumeStartsResumeJobWithSameRunData(t *testing.T) {
	r := newTestRunner(nil)
	dataDir := t.TempDir()
	r.SetDataVolume("markovd-data", dataDir)
	ctx := context.Background()
	runID, err := r.Start(ctx, dirWorkflow())
	if err != nil {
		t.Fatal(err)
	}

	if _, err := r.Resume(ctx, ResumeRequest{RunID: runID}); !errors.Is(err, ErrResumeUnsupported) {
		t.Fatalf("resume without saved state: err = %v", err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "runs", runID, "markov-state.db"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	jobName, err := r.Resume(ctx, ResumeRequest{RunID: runID, Vars: map[string]string{"reviewed": "true"}})
	if err != nil {
		t.Fatalf("Resume() error: %v", err)
	}
	if !strings.HasPrefix(jobName, runID+"-resume-") {
		t.Fatalf("job name = %s", jobName)
	}
	job, err := r.client.BatchV1().Jobs("ai-pipeline").Get(ctx, jobName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("resume Job not created: %v", err)
	}
	c := job.Spec.Template.Spec.Containers[0]
	if strings.Join(c.Args[:3], " ") != "resume "+runID+" --verbose" || !strings.Contains(strings.Join(c.Args, " "), "--var reviewed=true") {
		t.Fatalf("args = %v", c.Args)
	}
	if c.VolumeMounts[0].SubPath != "runs/"+runID || job.Labels["markov/run-id"] != runID {
		t.Fatalf("mount %#v labels %v", c.VolumeMounts, job.Labels)
	}

	if err := r.Delete(runID); err != nil {
		t.Fatalf("Delete() error: %v", err)
	}
	for _, name := range []string{runID, jobName} {
		if _, err := r.client.BatchV1().Jobs("ai-pipeline").Get(ctx, name, metav1.GetOptions{}); err == nil {
			t.Fatalf("job %s not deleted", name)
		}
	}
	if _, err := os.Stat(filepath.Join(dataDir, "runs", runID)); !os.IsNotExist(err) {
		t.Fatalf("run directory not removed: %v", err)
	}
}

func TestResumeWithoutDataVolumeIsUnsupported(t *testing.T) {
	if _, err := newTestRunner(nil).Resume(context.Background(), ResumeRequest{RunID: "x"}); !errors.Is(err, ErrResumeUnsupported) {
		t.Fatalf("err = %v", err)
	}
}

func TestWriteRunFilesRejectsEscapingPaths(t *testing.T) {
	_, err := writeRunFiles(t.TempDir(), models.WorkflowDefinition{
		Kind:  workflowdef.KindDirectory,
		Files: []models.WorkflowDefinitionFile{{Path: "../evil", Content: "x"}},
	})
	if err == nil {
		t.Fatal("expected an error for ../evil")
	}
}
