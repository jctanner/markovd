package runner

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jctanner/markovd/internal/models"
	"github.com/jctanner/markovd/internal/workflowdef"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type KubernetesRunner struct {
	client              kubernetes.Interface
	image               string
	imagePullPolicy     corev1.PullPolicy
	namespace           string
	serviceAccount      string
	secrets             []string
	defaultVolumes      []PVCMount
	defaultSecretMounts []SecretMount
	dataPVC             string
	dataDir             string
}

func NewKubernetesRunner(image, imagePullPolicy, namespace, serviceAccount string, secrets []string, defaultVolumes []PVCMount, defaultSecretMounts []SecretMount) (*KubernetesRunner, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("k8s in-cluster config: %w", err)
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("k8s client: %w", err)
	}

	pullPolicy := corev1.PullPolicy(imagePullPolicy)
	if pullPolicy == "" {
		pullPolicy = corev1.PullIfNotPresent
	}

	return &KubernetesRunner{
		client:              client,
		image:               image,
		imagePullPolicy:     pullPolicy,
		namespace:           namespace,
		serviceAccount:      serviceAccount,
		secrets:             secrets,
		defaultVolumes:      defaultVolumes,
		defaultSecretMounts: defaultSecretMounts,
	}, nil
}

// Run data. With a data volume configured (SetDataVolume), each run gets a directory on it,
// runs/<run-id>, holding the workflow files and Markov's state database. The runner pod mounts
// that directory, so a run has no size cap beyond the volume, and a later `markov resume` Job
// finds the same files and state. Without a data volume, the workflow goes into a ConfigMap
// (about 1 MiB at most) and state lives only in the pod, so runs can't be resumed.
const runDataMount = "/run-data"

// SetDataVolume makes runs keep their files and state on a PVC: pvc is the claim the runner pods
// mount, and dir is where markovd itself has that claim mounted.
func (r *KubernetesRunner) SetDataVolume(pvc, dir string) {
	r.dataPVC = pvc
	r.dataDir = dir
}

func (r *KubernetesRunner) runDataDir(runID string) string {
	return filepath.Join(r.dataDir, "runs", runID)
}

func (r *KubernetesRunner) Start(ctx context.Context, req RunRequest) (string, error) {
	runID := generateRunID()
	def, err := workflowdef.RuntimeCompatibleDefinition(req.WorkflowDefinition())
	if err != nil {
		return "", fmt.Errorf("invalid workflow definition: %w", err)
	}

	var workflowPath string
	var mounts runMounts
	if r.dataPVC != "" {
		workflowPath, err = writeRunFiles(r.runDataDir(runID), def)
		if err != nil {
			return "", err
		}
		mounts = r.dataMounts(runID)
	} else {
		cmName := runID + "-workflow"
		cmData, cmItems, path := configMapWorkflowData(def)
		workflowPath = path
		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: cmName, Labels: runLabels(runID)},
			Data:       cmData,
		}
		if _, err := r.client.CoreV1().ConfigMaps(r.namespace).Create(ctx, cm, metav1.CreateOptions{}); err != nil {
			return "", fmt.Errorf("creating workflow configmap: %w", err)
		}
		mountPath := "/etc/markov"
		if def.Kind == workflowdef.KindDirectory {
			mountPath = "/etc/markov/workflow"
		}
		mounts.volumes = []corev1.Volume{{
			Name: "workflow",
			VolumeSource: corev1.VolumeSource{ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{Name: cmName},
				Items:                cmItems,
			}},
		}}
		mounts.mounts = []corev1.VolumeMount{{Name: "workflow", MountPath: mountPath, ReadOnly: true}}
	}

	args := []string{"run", workflowPath, "--verbose", "--run-id", runID, "--namespace", r.namespace}
	if req.WorkflowEntrypoint != "" {
		args = append(args, "--workflow", req.WorkflowEntrypoint)
	}
	if req.Debug {
		args = append(args, "--debug")
	}
	args = append(args, callbackArgs(req.CallbackURL, req.CallbackToken)...)
	args = append(args, varArgs(req.Vars)...)

	if err := r.createJob(ctx, runID, runID, args, mounts, req.Volumes, req.SecretVolumes); err != nil {
		if r.dataPVC == "" {
			_ = r.client.CoreV1().ConfigMaps(r.namespace).Delete(ctx, runID+"-workflow", metav1.DeleteOptions{})
		}
		return "", err
	}
	return runID, nil
}

// Resume starts `markov resume` for a paused or failed run in a new Job, with the run's files
// and state from the data volume. It returns the Job's name.
func (r *KubernetesRunner) Resume(ctx context.Context, req ResumeRequest) (string, error) {
	if r.dataPVC == "" {
		return "", ErrResumeUnsupported
	}
	if _, err := os.Stat(filepath.Join(r.runDataDir(req.RunID), "markov-state.db")); err != nil {
		return "", fmt.Errorf("%w: run %s has no saved state (it started before runs were kept on the data volume)", ErrResumeUnsupported, req.RunID)
	}
	b := make([]byte, 2)
	rand.Read(b)
	jobName := fmt.Sprintf("%s-resume-%s", req.RunID, hex.EncodeToString(b))

	args := []string{"resume", req.RunID, "--verbose", "--namespace", r.namespace}
	args = append(args, callbackArgs(req.CallbackURL, req.CallbackToken)...)
	args = append(args, varArgs(req.Vars)...)
	if err := r.createJob(ctx, jobName, req.RunID, args, r.dataMounts(req.RunID), req.Volumes, req.SecretVolumes); err != nil {
		return "", err
	}
	return jobName, nil
}

type runMounts struct {
	volumes []corev1.Volume
	mounts  []corev1.VolumeMount
	env     []corev1.EnvVar
}

func (r *KubernetesRunner) dataMounts(runID string) runMounts {
	return runMounts{
		volumes: []corev1.Volume{{
			Name: "run-data",
			VolumeSource: corev1.VolumeSource{PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
				ClaimName: r.dataPVC,
			}},
		}},
		mounts: []corev1.VolumeMount{{Name: "run-data", MountPath: runDataMount, SubPath: "runs/" + runID}},
		env:    []corev1.EnvVar{{Name: "MARKOV_STATE_STORE", Value: runDataMount + "/markov-state.db"}},
	}
}

// writeRunFiles writes a definition under dir and returns the path the runner pod passes to
// markov: <mount>/workflow for a directory workflow, <mount>/workflow.yaml for a single file.
func writeRunFiles(dir string, def models.WorkflowDefinition) (string, error) {
	root := filepath.Join(dir, "workflow")
	podPath := runDataMount + "/workflow"
	if def.Kind != workflowdef.KindDirectory {
		root = dir
		podPath = runDataMount + "/workflow.yaml"
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return "", fmt.Errorf("creating run directory: %w", err)
	}
	for _, f := range def.Files {
		rel := f.Path
		if def.Kind != workflowdef.KindDirectory {
			rel = "workflow.yaml"
		}
		clean := filepath.Clean(rel)
		if filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
			return "", fmt.Errorf("workflow file path %q escapes the run directory", f.Path)
		}
		target := filepath.Join(root, clean)
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return "", fmt.Errorf("creating %s: %w", filepath.Dir(target), err)
		}
		if err := os.WriteFile(target, []byte(f.Content), 0o644); err != nil {
			return "", fmt.Errorf("writing %s: %w", target, err)
		}
	}
	return podPath, nil
}

func runLabels(runID string) map[string]string {
	return map[string]string{"app": "markov", "markov/run-id": runID}
}

func callbackArgs(url, token string) []string {
	var args []string
	if url != "" {
		args = append(args, "--callback", url)
	}
	if token != "" {
		args = append(args, "--callback-header", fmt.Sprintf("Authorization=Bearer %s", token))
	}
	return args
}

func varArgs(vars map[string]string) []string {
	keys := make([]string, 0, len(vars))
	for k := range vars {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var args []string
	for _, k := range keys {
		args = append(args, "--var", fmt.Sprintf("%s=%s", k, vars[k]))
	}
	return args
}

// createJob creates the runner Job: the markov image with args, the run's own mounts, the
// default and requested PVCs and secrets.
func (r *KubernetesRunner) createJob(ctx context.Context, jobName, runID string, args []string, run runMounts, pvcs []PVCMount, secretMounts []SecretMount) error {
	var envFrom []corev1.EnvFromSource
	for _, s := range r.secrets {
		envFrom = append(envFrom, corev1.EnvFromSource{
			SecretRef: &corev1.SecretEnvSource{LocalObjectReference: corev1.LocalObjectReference{Name: s}},
		})
	}

	volumes := append([]corev1.Volume{}, run.volumes...)
	volumeMounts := append([]corev1.VolumeMount{}, run.mounts...)

	seen := map[string]bool{}
	for _, pvc := range append(append([]PVCMount{}, r.defaultVolumes...), pvcs...) {
		if seen[pvc.Name] {
			continue
		}
		seen[pvc.Name] = true
		volumes = append(volumes, corev1.Volume{
			Name: pvc.Name,
			VolumeSource: corev1.VolumeSource{PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
				ClaimName: pvc.PVC,
				ReadOnly:  pvc.ReadOnly,
			}},
		})
		volumeMounts = append(volumeMounts, corev1.VolumeMount{Name: pvc.Name, MountPath: pvc.MountPath, ReadOnly: pvc.ReadOnly})
	}

	seenSecrets := map[string]bool{}
	for _, sm := range append(append([]SecretMount{}, r.defaultSecretMounts...), secretMounts...) {
		if seenSecrets[sm.Name] {
			continue
		}
		seenSecrets[sm.Name] = true
		volumes = append(volumes, corev1.Volume{
			Name:         sm.Name,
			VolumeSource: corev1.VolumeSource{Secret: &corev1.SecretVolumeSource{SecretName: sm.Secret}},
		})
		volumeMounts = append(volumeMounts, corev1.VolumeMount{Name: sm.Name, MountPath: sm.MountPath, ReadOnly: sm.ReadOnly})
	}

	var backoffLimit int32
	var ttl int32 = 86400
	labels := runLabels(runID)

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{Name: jobName, Labels: labels},
		Spec: batchv1.JobSpec{
			BackoffLimit:            &backoffLimit,
			TTLSecondsAfterFinished: &ttl,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels},
				Spec: corev1.PodSpec{
					RestartPolicy:      corev1.RestartPolicyNever,
					ServiceAccountName: r.serviceAccount,
					SecurityContext: &corev1.PodSecurityContext{
						FSGroup: func() *int64 { v := int64(1000); return &v }(),
					},
					Containers: []corev1.Container{{
						Name:            "markov",
						Image:           r.image,
						ImagePullPolicy: r.imagePullPolicy,
						Command:         []string{"markov"},
						Args:            args,
						Env:             run.env,
						EnvFrom:         envFrom,
						VolumeMounts:    volumeMounts,
					}},
					Volumes: volumes,
				},
			},
		},
	}
	if _, err := r.client.BatchV1().Jobs(r.namespace).Create(ctx, job, metav1.CreateOptions{}); err != nil {
		return fmt.Errorf("creating job: %w", err)
	}
	return nil
}

func configMapWorkflowData(def models.WorkflowDefinition) (map[string]string, []corev1.KeyToPath, string) {
	data := map[string]string{}
	var items []corev1.KeyToPath
	workflowPath := "/etc/markov/workflow.yaml"
	if def.Kind == workflowdef.KindDirectory {
		workflowPath = "/etc/markov/workflow"
	}
	for i, f := range def.Files {
		key := "workflow.yaml"
		path := "workflow.yaml"
		if def.Kind == workflowdef.KindDirectory {
			key = fmt.Sprintf("f-%03d", i)
			path = f.Path
		}
		data[key] = f.Content
		items = append(items, corev1.KeyToPath{Key: key, Path: path})
	}
	return data, items, workflowPath
}

// Cancel stops a run: every Job labelled with its run ID (the first run and any resumes). It
// returns an error when the run has no Job.
func (r *KubernetesRunner) Cancel(runID string) error {
	ctx := context.Background()
	propagation := metav1.DeletePropagationBackground
	_ = r.client.CoreV1().ConfigMaps(r.namespace).Delete(ctx, runID+"-workflow", metav1.DeleteOptions{})

	jobs, err := r.client.BatchV1().Jobs(r.namespace).List(ctx, metav1.ListOptions{LabelSelector: "markov/run-id=" + runID})
	if err != nil {
		return fmt.Errorf("listing jobs: %w", err)
	}
	names := map[string]bool{runID: true}
	for _, j := range jobs.Items {
		names[j.Name] = true
	}
	deleted := 0
	var firstErr error
	for name := range names {
		err := r.client.BatchV1().Jobs(r.namespace).Delete(ctx, name, metav1.DeleteOptions{PropagationPolicy: &propagation})
		switch {
		case err == nil:
			deleted++
		case apierrors.IsNotFound(err):
		case firstErr == nil:
			firstErr = err
		}
	}
	if firstErr != nil {
		return fmt.Errorf("deleting job: %w", firstErr)
	}
	if deleted == 0 {
		return fmt.Errorf("deleting job: no job found for run %s", runID)
	}
	return nil
}

// Delete cancels a run and removes its files and state from the data volume.
func (r *KubernetesRunner) Delete(runID string) error {
	err := r.Cancel(runID)
	if r.dataPVC != "" && runID != "" && !strings.ContainsAny(runID, "/.") {
		if rmErr := os.RemoveAll(r.runDataDir(runID)); rmErr != nil && err == nil {
			err = rmErr
		}
	}
	return err
}

func (r *KubernetesRunner) ListPVCs(ctx context.Context) ([]PVCInfo, error) {
	list, err := r.client.CoreV1().PersistentVolumeClaims(r.namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("listing PVCs: %w", err)
	}
	var pvcs []PVCInfo
	for _, pvc := range list.Items {
		pvcs = append(pvcs, PVCInfo{
			Name:   pvc.Name,
			Status: string(pvc.Status.Phase),
		})
	}
	return pvcs, nil
}

func (r *KubernetesRunner) ListSecrets(ctx context.Context) ([]SecretInfo, error) {
	list, err := r.client.CoreV1().Secrets(r.namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("listing Secrets: %w", err)
	}
	var secrets []SecretInfo
	for _, s := range list.Items {
		sType := string(s.Type)
		if sType == "" {
			sType = "Opaque"
		}
		secrets = append(secrets, SecretInfo{
			Name: s.Name,
			Type: sType,
		})
	}
	return secrets, nil
}

func (r *KubernetesRunner) GetJobLogs(ctx context.Context, jobName string) (string, error) {
	pods, err := r.client.CoreV1().Pods(r.namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("job-name=%s", jobName),
	})
	if err != nil {
		return "", fmt.Errorf("listing pods for job %s: %w", jobName, err)
	}
	if len(pods.Items) == 0 {
		return "", fmt.Errorf("no pods found for job %s", jobName)
	}

	podName := pods.Items[0].Name
	req := r.client.CoreV1().Pods(r.namespace).GetLogs(podName, &corev1.PodLogOptions{})
	stream, err := req.Stream(ctx)
	if err != nil {
		return "", fmt.Errorf("streaming logs for pod %s: %w", podName, err)
	}
	defer stream.Close()

	var buf strings.Builder
	if _, err := io.Copy(&buf, stream); err != nil {
		return "", fmt.Errorf("reading logs for pod %s: %w", podName, err)
	}
	return buf.String(), nil
}

func (r *KubernetesRunner) StreamJobLogs(ctx context.Context, jobName string) (io.ReadCloser, error) {
	pods, err := r.client.CoreV1().Pods(r.namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("job-name=%s", jobName),
	})
	if err != nil {
		return nil, fmt.Errorf("listing pods for job %s: %w", jobName, err)
	}
	if len(pods.Items) == 0 {
		return nil, fmt.Errorf("no pods found for job %s", jobName)
	}

	podName := pods.Items[0].Name
	req := r.client.CoreV1().Pods(r.namespace).GetLogs(podName, &corev1.PodLogOptions{Follow: true})
	return req.Stream(ctx)
}

func (r *KubernetesRunner) AuditJobStatuses(ctx context.Context) (map[string]string, error) {
	jobs, err := r.client.BatchV1().Jobs(r.namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("listing jobs: %w", err)
	}

	statuses := make(map[string]string, len(jobs.Items))
	for _, j := range jobs.Items {
		switch {
		case j.Status.Succeeded > 0:
			statuses[j.Name] = "completed"
		case j.Status.Failed > 0:
			statuses[j.Name] = "failed"
		case j.Status.Active > 0:
			statuses[j.Name] = "running"
		default:
			statuses[j.Name] = "pending"
		}
	}
	return statuses, nil
}

func generateRunID() string {
	b := make([]byte, 4)
	rand.Read(b)
	return "markov-run-" + hex.EncodeToString(b)
}

func ParseSecrets(s string) []string {
	if s == "" {
		return nil
	}
	var secrets []string
	for _, name := range strings.Split(s, ",") {
		name = strings.TrimSpace(name)
		if name != "" {
			secrets = append(secrets, name)
		}
	}
	return secrets
}
