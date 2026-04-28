package runner

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"strings"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
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

func (r *KubernetesRunner) Start(ctx context.Context, req RunRequest) (string, error) {
	runID := generateRunID()
	cmName := runID + "-workflow"

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: cmName,
			Labels: map[string]string{
				"app":            "markov",
				"markov/run-id":  runID,
			},
		},
		Data: map[string]string{
			"workflow.yaml": req.WorkflowYAML,
		},
	}

	_, err := r.client.CoreV1().ConfigMaps(r.namespace).Create(ctx, cm, metav1.CreateOptions{})
	if err != nil {
		return "", fmt.Errorf("creating workflow configmap: %w", err)
	}

	args := []string{
		"run", "/etc/markov/workflow.yaml", "--verbose",
		"--run-id", runID,
		"--namespace", r.namespace,
	}
	if req.Debug {
		args = append(args, "--debug")
	}
	if req.CallbackURL != "" {
		args = append(args, "--callback", req.CallbackURL)
	}
	if req.CallbackToken != "" {
		args = append(args, "--callback-header", fmt.Sprintf("Authorization=Bearer %s", req.CallbackToken))
	}
	for k, v := range req.Vars {
		args = append(args, "--var", fmt.Sprintf("%s=%s", k, v))
	}

	var envFrom []corev1.EnvFromSource
	for _, s := range r.secrets {
		envFrom = append(envFrom, corev1.EnvFromSource{
			SecretRef: &corev1.SecretEnvSource{
				LocalObjectReference: corev1.LocalObjectReference{Name: s},
			},
		})
	}

	volumeMounts := []corev1.VolumeMount{
		{
			Name:      "workflow",
			MountPath: "/etc/markov",
			ReadOnly:  true,
		},
	}
	volumes := []corev1.Volume{
		{
			Name: "workflow",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{Name: cmName},
				},
			},
		},
	}

	allPVCs := append(r.defaultVolumes, req.Volumes...)
	seen := map[string]bool{}
	for _, pvc := range allPVCs {
		if seen[pvc.Name] {
			continue
		}
		seen[pvc.Name] = true
		volumes = append(volumes, corev1.Volume{
			Name: pvc.Name,
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: pvc.PVC,
					ReadOnly:  pvc.ReadOnly,
				},
			},
		})
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      pvc.Name,
			MountPath: pvc.MountPath,
			ReadOnly:  pvc.ReadOnly,
		})
	}

	allSecretMounts := append(r.defaultSecretMounts, req.SecretVolumes...)
	seenSecrets := map[string]bool{}
	for _, sm := range allSecretMounts {
		if seenSecrets[sm.Name] {
			continue
		}
		seenSecrets[sm.Name] = true
		volumes = append(volumes, corev1.Volume{
			Name: sm.Name,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: sm.Secret,
				},
			},
		})
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      sm.Name,
			MountPath: sm.MountPath,
			ReadOnly:  sm.ReadOnly,
		})
	}

	var backoffLimit int32
	var ttl int32 = 86400

	labels := map[string]string{
		"app":           "markov",
		"markov/run-id": runID,
	}

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:   runID,
			Labels: labels,
		},
		Spec: batchv1.JobSpec{
			BackoffLimit:            &backoffLimit,
			TTLSecondsAfterFinished: &ttl,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					RestartPolicy:      corev1.RestartPolicyNever,
					ServiceAccountName: r.serviceAccount,
					Containers: []corev1.Container{
						{
							Name:            "markov",
							Image:           r.image,
							ImagePullPolicy: r.imagePullPolicy,
							Command:         []string{"markov"},
							Args:            args,
							EnvFrom:         envFrom,
							VolumeMounts:    volumeMounts,
						},
					},
					Volumes: volumes,
				},
			},
		},
	}

	_, err = r.client.BatchV1().Jobs(r.namespace).Create(ctx, job, metav1.CreateOptions{})
	if err != nil {
		_ = r.client.CoreV1().ConfigMaps(r.namespace).Delete(ctx, cmName, metav1.DeleteOptions{})
		return "", fmt.Errorf("creating job: %w", err)
	}

	return runID, nil
}

func (r *KubernetesRunner) Cancel(runID string) error {
	ctx := context.Background()
	propagation := metav1.DeletePropagationBackground

	err := r.client.BatchV1().Jobs(r.namespace).Delete(ctx, runID, metav1.DeleteOptions{
		PropagationPolicy: &propagation,
	})
	if err != nil {
		return fmt.Errorf("deleting job: %w", err)
	}

	cmName := runID + "-workflow"
	_ = r.client.CoreV1().ConfigMaps(r.namespace).Delete(ctx, cmName, metav1.DeleteOptions{})

	return nil
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
