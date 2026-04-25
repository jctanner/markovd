package runner

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func newTestRunner(secrets []string) *KubernetesRunner {
	return newTestRunnerWithVolumes(secrets, nil)
}

func newTestRunnerWithVolumes(secrets []string, volumes []PVCMount) *KubernetesRunner {
	return newTestRunnerFull(secrets, volumes, nil)
}

func newTestRunnerFull(secrets []string, volumes []PVCMount, secretMounts []SecretMount) *KubernetesRunner {
	return &KubernetesRunner{
		client:              fake.NewSimpleClientset(),
		image:               "ghcr.io/jctanner/markov:latest",
		imagePullPolicy:     corev1.PullIfNotPresent,
		namespace:           "ai-pipeline",
		serviceAccount:      "pipeline-agent",
		secrets:             secrets,
		defaultVolumes:      volumes,
		defaultSecretMounts: secretMounts,
	}
}

func TestStartCreatesConfigMapAndJob(t *testing.T) {
	r := newTestRunner(nil)
	ctx := context.Background()

	req := RunRequest{
		WorkflowYAML:  "entrypoint: main\nworkflows:\n  - name: main\n    steps: []\n",
		Vars:          map[string]string{"env": "staging"},
		CallbackURL:   "http://markovd:8080/api/v1/events",
		CallbackToken: "test-token-123",
	}

	runID, err := r.Start(ctx, req)
	if err != nil {
		t.Fatalf("Start() error: %v", err)
	}
	if runID == "" {
		t.Fatal("Start() returned empty runID")
	}

	cm, err := r.client.CoreV1().ConfigMaps("ai-pipeline").Get(ctx, runID+"-workflow", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("ConfigMap not created: %v", err)
	}
	if cm.Data["workflow.yaml"] != req.WorkflowYAML {
		t.Errorf("ConfigMap workflow.yaml = %q, want %q", cm.Data["workflow.yaml"], req.WorkflowYAML)
	}
	if cm.Labels["markov/run-id"] != runID {
		t.Errorf("ConfigMap label markov/run-id = %q, want %q", cm.Labels["markov/run-id"], runID)
	}

	job, err := r.client.BatchV1().Jobs("ai-pipeline").Get(ctx, runID, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Job not created: %v", err)
	}
	if job.Labels["app"] != "markov" {
		t.Errorf("Job label app = %q, want %q", job.Labels["app"], "markov")
	}
	if job.Labels["markov/run-id"] != runID {
		t.Errorf("Job label markov/run-id = %q, want %q", job.Labels["markov/run-id"], runID)
	}

	spec := job.Spec.Template.Spec
	if spec.ServiceAccountName != "pipeline-agent" {
		t.Errorf("ServiceAccountName = %q, want %q", spec.ServiceAccountName, "pipeline-agent")
	}
	if len(spec.Containers) != 1 {
		t.Fatalf("expected 1 container, got %d", len(spec.Containers))
	}

	container := spec.Containers[0]
	if container.Image != "ghcr.io/jctanner/markov:latest" {
		t.Errorf("Image = %q, want %q", container.Image, "ghcr.io/jctanner/markov:latest")
	}
	if len(container.VolumeMounts) != 1 || container.VolumeMounts[0].MountPath != "/etc/markov" {
		t.Errorf("expected volume mount at /etc/markov, got %v", container.VolumeMounts)
	}

	hasCallback := false
	hasCallbackHeader := false
	hasVar := false
	hasNamespace := false
	hasRunID := false
	for i, arg := range container.Args {
		switch arg {
		case "--callback":
			if i+1 < len(container.Args) && container.Args[i+1] == "http://markovd:8080/api/v1/events" {
				hasCallback = true
			}
		case "--callback-header":
			if i+1 < len(container.Args) && container.Args[i+1] == "Authorization=Bearer test-token-123" {
				hasCallbackHeader = true
			}
		case "--var":
			if i+1 < len(container.Args) && container.Args[i+1] == "env=staging" {
				hasVar = true
			}
		case "--run-id":
			if i+1 < len(container.Args) && container.Args[i+1] == runID {
				hasRunID = true
			}
		case "--namespace":
			if i+1 < len(container.Args) && container.Args[i+1] == "ai-pipeline" {
				hasNamespace = true
			}
		}
	}
	if !hasCallback {
		t.Error("args missing --callback")
	}
	if !hasCallbackHeader {
		t.Error("args missing --callback-header with bearer token")
	}
	if !hasVar {
		t.Error("args missing --var env=staging")
	}
	if !hasNamespace {
		t.Error("args missing --namespace ai-pipeline")
	}
	if !hasRunID {
		t.Error("args missing --run-id matching run ID")
	}
}

func TestStartInjectsSecrets(t *testing.T) {
	r := newTestRunner([]string{"pipeline-credentials", "jira-token"})
	ctx := context.Background()

	req := RunRequest{WorkflowYAML: "entrypoint: main"}

	runID, err := r.Start(ctx, req)
	if err != nil {
		t.Fatalf("Start() error: %v", err)
	}

	job, err := r.client.BatchV1().Jobs("ai-pipeline").Get(ctx, runID, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Job not created: %v", err)
	}

	envFrom := job.Spec.Template.Spec.Containers[0].EnvFrom
	if len(envFrom) != 2 {
		t.Fatalf("expected 2 envFrom entries, got %d", len(envFrom))
	}

	names := map[string]bool{}
	for _, ef := range envFrom {
		if ef.SecretRef != nil {
			names[ef.SecretRef.Name] = true
		}
	}
	if !names["pipeline-credentials"] {
		t.Error("missing envFrom for pipeline-credentials")
	}
	if !names["jira-token"] {
		t.Error("missing envFrom for jira-token")
	}
}

func TestStartNoCallbackWhenEmpty(t *testing.T) {
	r := newTestRunner(nil)
	ctx := context.Background()

	req := RunRequest{WorkflowYAML: "entrypoint: main"}

	runID, err := r.Start(ctx, req)
	if err != nil {
		t.Fatalf("Start() error: %v", err)
	}

	job, err := r.client.BatchV1().Jobs("ai-pipeline").Get(ctx, runID, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Job not created: %v", err)
	}

	for _, arg := range job.Spec.Template.Spec.Containers[0].Args {
		if arg == "--callback" || arg == "--callback-header" {
			t.Errorf("unexpected arg %q when callback is empty", arg)
		}
	}
}

func TestCancelDeletesJobAndConfigMap(t *testing.T) {
	r := newTestRunner(nil)
	ctx := context.Background()

	req := RunRequest{WorkflowYAML: "entrypoint: main"}
	runID, err := r.Start(ctx, req)
	if err != nil {
		t.Fatalf("Start() error: %v", err)
	}

	// Verify both exist before cancel
	_, err = r.client.BatchV1().Jobs("ai-pipeline").Get(ctx, runID, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Job should exist before cancel: %v", err)
	}
	_, err = r.client.CoreV1().ConfigMaps("ai-pipeline").Get(ctx, runID+"-workflow", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("ConfigMap should exist before cancel: %v", err)
	}

	if err := r.Cancel(runID); err != nil {
		t.Fatalf("Cancel() error: %v", err)
	}

	_, err = r.client.BatchV1().Jobs("ai-pipeline").Get(ctx, runID, metav1.GetOptions{})
	if err == nil {
		t.Error("Job should be deleted after cancel")
	}
	_, err = r.client.CoreV1().ConfigMaps("ai-pipeline").Get(ctx, runID+"-workflow", metav1.GetOptions{})
	if err == nil {
		t.Error("ConfigMap should be deleted after cancel")
	}
}

func TestCancelNonexistentJobReturnsError(t *testing.T) {
	r := newTestRunner(nil)
	if err := r.Cancel("nonexistent-run"); err == nil {
		t.Error("Cancel() should return error for nonexistent job")
	}
}

func TestGenerateRunID(t *testing.T) {
	id1 := generateRunID()
	id2 := generateRunID()

	if id1 == id2 {
		t.Error("generateRunID() returned same value twice")
	}
	if len(id1) != len("markov-run-")+8 {
		t.Errorf("generateRunID() length = %d, want %d", len(id1), len("markov-run-")+8)
	}
}

func TestParseSecrets(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"", 0},
		{"one", 1},
		{"one,two,three", 3},
		{" one , two , ", 2},
		{",,,", 0},
	}

	for _, tt := range tests {
		got := ParseSecrets(tt.input)
		if len(got) != tt.want {
			t.Errorf("ParseSecrets(%q) = %v (len %d), want len %d", tt.input, got, len(got), tt.want)
		}
	}
}

func TestStartWithDebugFlag(t *testing.T) {
	r := newTestRunner(nil)
	ctx := context.Background()

	runID, err := r.Start(ctx, RunRequest{WorkflowYAML: "test", Debug: true})
	if err != nil {
		t.Fatalf("Start() error: %v", err)
	}

	job, _ := r.client.BatchV1().Jobs("ai-pipeline").Get(ctx, runID, metav1.GetOptions{})
	args := job.Spec.Template.Spec.Containers[0].Args
	hasDebug := false
	for _, arg := range args {
		if arg == "--debug" {
			hasDebug = true
			break
		}
	}
	if !hasDebug {
		t.Errorf("args %v missing --debug", args)
	}
}

func TestStartWithoutDebugFlag(t *testing.T) {
	r := newTestRunner(nil)
	ctx := context.Background()

	runID, err := r.Start(ctx, RunRequest{WorkflowYAML: "test", Debug: false})
	if err != nil {
		t.Fatalf("Start() error: %v", err)
	}

	job, _ := r.client.BatchV1().Jobs("ai-pipeline").Get(ctx, runID, metav1.GetOptions{})
	for _, arg := range job.Spec.Template.Spec.Containers[0].Args {
		if arg == "--debug" {
			t.Error("args should not contain --debug when Debug is false")
		}
	}
}

func TestImagePullPolicy(t *testing.T) {
	r := newTestRunner(nil)
	ctx := context.Background()

	runID, err := r.Start(ctx, RunRequest{WorkflowYAML: "test"})
	if err != nil {
		t.Fatalf("Start() error: %v", err)
	}

	job, _ := r.client.BatchV1().Jobs("ai-pipeline").Get(ctx, runID, metav1.GetOptions{})
	policy := job.Spec.Template.Spec.Containers[0].ImagePullPolicy
	if policy != corev1.PullIfNotPresent {
		t.Errorf("ImagePullPolicy = %q, want %q", policy, corev1.PullIfNotPresent)
	}

	// Test with Never
	r.imagePullPolicy = corev1.PullNever
	runID2, err := r.Start(ctx, RunRequest{WorkflowYAML: "test"})
	if err != nil {
		t.Fatalf("Start() error: %v", err)
	}

	job2, _ := r.client.BatchV1().Jobs("ai-pipeline").Get(ctx, runID2, metav1.GetOptions{})
	policy2 := job2.Spec.Template.Spec.Containers[0].ImagePullPolicy
	if policy2 != corev1.PullNever {
		t.Errorf("ImagePullPolicy = %q, want %q", policy2, corev1.PullNever)
	}
}

func TestJobSpecDetails(t *testing.T) {
	r := newTestRunner(nil)
	ctx := context.Background()

	runID, err := r.Start(ctx, RunRequest{WorkflowYAML: "test"})
	if err != nil {
		t.Fatalf("Start() error: %v", err)
	}

	job, _ := r.client.BatchV1().Jobs("ai-pipeline").Get(ctx, runID, metav1.GetOptions{})

	if *job.Spec.BackoffLimit != 0 {
		t.Errorf("BackoffLimit = %d, want 0", *job.Spec.BackoffLimit)
	}
	if *job.Spec.TTLSecondsAfterFinished != 86400 {
		t.Errorf("TTLSecondsAfterFinished = %d, want 86400", *job.Spec.TTLSecondsAfterFinished)
	}
	if job.Spec.Template.Spec.RestartPolicy != "Never" {
		t.Errorf("RestartPolicy = %q, want %q", job.Spec.Template.Spec.RestartPolicy, "Never")
	}

	vols := job.Spec.Template.Spec.Volumes
	if len(vols) != 1 {
		t.Fatalf("expected 1 volume, got %d", len(vols))
	}
	if vols[0].ConfigMap == nil || vols[0].ConfigMap.Name != runID+"-workflow" {
		t.Errorf("volume should reference configmap %s-workflow", runID)
	}
}

func TestStartWithDefaultPVCVolumes(t *testing.T) {
	r := newTestRunnerWithVolumes(nil, []PVCMount{
		{Name: "pipeline-artifacts", PVC: "pipeline-artifacts", MountPath: "/app/artifacts"},
	})
	ctx := context.Background()

	runID, err := r.Start(ctx, RunRequest{WorkflowYAML: "test"})
	if err != nil {
		t.Fatalf("Start() error: %v", err)
	}

	job, _ := r.client.BatchV1().Jobs("ai-pipeline").Get(ctx, runID, metav1.GetOptions{})
	spec := job.Spec.Template.Spec

	if len(spec.Volumes) != 2 {
		t.Fatalf("expected 2 volumes (workflow + pvc), got %d", len(spec.Volumes))
	}
	pvcVol := spec.Volumes[1]
	if pvcVol.Name != "pipeline-artifacts" {
		t.Errorf("PVC volume name = %q, want %q", pvcVol.Name, "pipeline-artifacts")
	}
	if pvcVol.PersistentVolumeClaim == nil || pvcVol.PersistentVolumeClaim.ClaimName != "pipeline-artifacts" {
		t.Errorf("PVC claim name mismatch")
	}

	mounts := spec.Containers[0].VolumeMounts
	if len(mounts) != 2 {
		t.Fatalf("expected 2 volume mounts, got %d", len(mounts))
	}
	if mounts[1].MountPath != "/app/artifacts" {
		t.Errorf("PVC mount path = %q, want %q", mounts[1].MountPath, "/app/artifacts")
	}
}

func TestStartWithPerRunPVCVolumes(t *testing.T) {
	r := newTestRunner(nil)
	ctx := context.Background()

	runID, err := r.Start(ctx, RunRequest{
		WorkflowYAML: "test",
		Volumes: []PVCMount{
			{Name: "data-vol", PVC: "my-data-pvc", MountPath: "/data"},
		},
	})
	if err != nil {
		t.Fatalf("Start() error: %v", err)
	}

	job, _ := r.client.BatchV1().Jobs("ai-pipeline").Get(ctx, runID, metav1.GetOptions{})
	spec := job.Spec.Template.Spec

	if len(spec.Volumes) != 2 {
		t.Fatalf("expected 2 volumes, got %d", len(spec.Volumes))
	}
	if spec.Volumes[1].PersistentVolumeClaim.ClaimName != "my-data-pvc" {
		t.Errorf("PVC claim = %q, want %q", spec.Volumes[1].PersistentVolumeClaim.ClaimName, "my-data-pvc")
	}
	if spec.Containers[0].VolumeMounts[1].MountPath != "/data" {
		t.Errorf("mount path = %q, want %q", spec.Containers[0].VolumeMounts[1].MountPath, "/data")
	}
}

func TestStartMergesDefaultAndPerRunVolumes(t *testing.T) {
	r := newTestRunnerWithVolumes(nil, []PVCMount{
		{Name: "default-vol", PVC: "default-pvc", MountPath: "/default"},
	})
	ctx := context.Background()

	runID, err := r.Start(ctx, RunRequest{
		WorkflowYAML: "test",
		Volumes: []PVCMount{
			{Name: "extra-vol", PVC: "extra-pvc", MountPath: "/extra"},
		},
	})
	if err != nil {
		t.Fatalf("Start() error: %v", err)
	}

	job, _ := r.client.BatchV1().Jobs("ai-pipeline").Get(ctx, runID, metav1.GetOptions{})
	spec := job.Spec.Template.Spec

	if len(spec.Volumes) != 3 {
		t.Fatalf("expected 3 volumes (workflow + 2 PVCs), got %d", len(spec.Volumes))
	}
	if len(spec.Containers[0].VolumeMounts) != 3 {
		t.Fatalf("expected 3 mounts, got %d", len(spec.Containers[0].VolumeMounts))
	}
}

func TestParseVolumes(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"", 0},
		{"pipeline-artifacts:/app/artifacts", 1},
		{"pvc1:/mnt/a,pvc2:/mnt/b", 2},
		{" pvc1:/mnt/a , pvc2:/mnt/b , ", 2},
		{",,,", 0},
		{"invalid", 0},
	}

	for _, tt := range tests {
		got := ParseVolumes(tt.input)
		if len(got) != tt.want {
			t.Errorf("ParseVolumes(%q) len = %d, want %d", tt.input, len(got), tt.want)
		}
	}

	vols := ParseVolumes("pipeline-artifacts:/app/artifacts")
	if vols[0].PVC != "pipeline-artifacts" || vols[0].MountPath != "/app/artifacts" {
		t.Errorf("ParseVolumes parsed incorrectly: %+v", vols[0])
	}
}

func TestStartWithSecretVolumes(t *testing.T) {
	r := newTestRunnerFull(nil, nil, []SecretMount{
		{Name: "gcp-creds", Secret: "gcp-credentials", MountPath: "/app/gcloud", ReadOnly: true},
	})
	ctx := context.Background()

	runID, err := r.Start(ctx, RunRequest{WorkflowYAML: "test"})
	if err != nil {
		t.Fatalf("Start() error: %v", err)
	}

	job, _ := r.client.BatchV1().Jobs("ai-pipeline").Get(ctx, runID, metav1.GetOptions{})
	spec := job.Spec.Template.Spec

	if len(spec.Volumes) != 2 {
		t.Fatalf("expected 2 volumes (workflow + secret), got %d", len(spec.Volumes))
	}
	secretVol := spec.Volumes[1]
	if secretVol.Name != "gcp-creds" {
		t.Errorf("secret volume name = %q, want %q", secretVol.Name, "gcp-creds")
	}
	if secretVol.Secret == nil || secretVol.Secret.SecretName != "gcp-credentials" {
		t.Errorf("secret volume source mismatch: %+v", secretVol)
	}

	mounts := spec.Containers[0].VolumeMounts
	if len(mounts) != 2 {
		t.Fatalf("expected 2 mounts, got %d", len(mounts))
	}
	if mounts[1].MountPath != "/app/gcloud" {
		t.Errorf("mount path = %q, want %q", mounts[1].MountPath, "/app/gcloud")
	}
	if !mounts[1].ReadOnly {
		t.Error("secret mount should be read-only")
	}
}

func TestStartMergesDefaultAndPerRunSecretVolumes(t *testing.T) {
	r := newTestRunnerFull(nil, nil, []SecretMount{
		{Name: "default-secret", Secret: "default-secret", MountPath: "/etc/default"},
	})
	ctx := context.Background()

	runID, err := r.Start(ctx, RunRequest{
		WorkflowYAML: "test",
		SecretVolumes: []SecretMount{
			{Name: "extra-secret", Secret: "extra-secret", MountPath: "/etc/extra"},
			{Name: "default-secret", Secret: "default-secret", MountPath: "/etc/override"},
		},
	})
	if err != nil {
		t.Fatalf("Start() error: %v", err)
	}

	job, _ := r.client.BatchV1().Jobs("ai-pipeline").Get(ctx, runID, metav1.GetOptions{})
	spec := job.Spec.Template.Spec

	if len(spec.Volumes) != 3 {
		t.Fatalf("expected 3 volumes (workflow + 2 secrets, deduped), got %d", len(spec.Volumes))
	}
	if len(spec.Containers[0].VolumeMounts) != 3 {
		t.Fatalf("expected 3 mounts, got %d", len(spec.Containers[0].VolumeMounts))
	}
}

func TestParseSecretMounts(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"", 0},
		{"gcp-creds:/app/gcloud", 1},
		{"secret1:/mnt/a,secret2:/mnt/b", 2},
		{" s1:/a , s2:/b , ", 2},
		{",,,", 0},
		{"invalid", 0},
	}

	for _, tt := range tests {
		got := ParseSecretMounts(tt.input)
		if len(got) != tt.want {
			t.Errorf("ParseSecretMounts(%q) len = %d, want %d", tt.input, len(got), tt.want)
		}
	}

	mounts := ParseSecretMounts("gcp-creds:/app/gcloud")
	if mounts[0].Secret != "gcp-creds" || mounts[0].MountPath != "/app/gcloud" {
		t.Errorf("ParseSecretMounts parsed incorrectly: %+v", mounts[0])
	}
}
