package runner

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type KubernetesRunner struct {
	client          kubernetes.Interface
	image           string
	imagePullPolicy corev1.PullPolicy
	namespace       string
	serviceAccount  string
	secrets         []string
}

func NewKubernetesRunner(image, imagePullPolicy, namespace, serviceAccount string, secrets []string) (*KubernetesRunner, error) {
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
		client:          client,
		image:           image,
		imagePullPolicy: pullPolicy,
		namespace:       namespace,
		serviceAccount:  serviceAccount,
		secrets:         secrets,
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
							Args:    args,
							EnvFrom: envFrom,
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "workflow",
									MountPath: "/etc/markov",
									ReadOnly:  true,
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "workflow",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{Name: cmName},
								},
							},
						},
					},
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
