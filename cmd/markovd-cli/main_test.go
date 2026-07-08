package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestParseConfigTOML(t *testing.T) {
	raw, err := parseConfigTOML(`
server = "https://markovd.local"
username = "admin"
password = "secret"
ssl_verify = false
ca_cert = "./ca.pem"

[defaults]
project = "ai-first-pipeline"
poll_interval = "3s"
timeout = "45m"
output = "json"

[[defaults.volumes]]
name = "pipeline-artifacts"
mount_path = "/app/artifacts"

[[defaults.volumes]]
name = "workspace"
pvc = "pipeline-workspace"
mount_path = "/app/workspace"
read_only = true

[[defaults.secret_volumes]]
name = "gcp-credentials"
mount_path = "/home/pipelineagent/.config/gcloud"
read_only = true
`)
	if err != nil {
		t.Fatal(err)
	}
	cfg := cliConfig{Server: defaultServer, Output: defaultOutput, Timeout: defaultTimeout, PollInterval: defaultPollInterval}
	mergeRawConfig(&cfg, raw)
	if cfg.Server != "https://markovd.local" {
		t.Fatalf("server = %q", cfg.Server)
	}
	if cfg.Username != "admin" || cfg.Password != "secret" {
		t.Fatalf("credentials not loaded: %#v", cfg)
	}
	if !cfg.InsecureSkipTLSVerify {
		t.Fatal("ssl_verify=false should set InsecureSkipTLSVerify")
	}
	if cfg.CACert != "./ca.pem" {
		t.Fatalf("ca cert = %q", cfg.CACert)
	}
	if cfg.Output != "json" || cfg.DefaultProject != "ai-first-pipeline" {
		t.Fatalf("defaults not loaded: %#v", cfg)
	}
	if cfg.PollInterval != 3*time.Second || cfg.Timeout != 45*time.Minute {
		t.Fatalf("durations not loaded: poll=%s timeout=%s", cfg.PollInterval, cfg.Timeout)
	}
	if len(cfg.DefaultVolumes) != 2 {
		t.Fatalf("default volumes = %#v", cfg.DefaultVolumes)
	}
	if cfg.DefaultVolumes[0].Name != "pipeline-artifacts" || cfg.DefaultVolumes[0].MountPath != "/app/artifacts" {
		t.Fatalf("first default volume = %#v", cfg.DefaultVolumes[0])
	}
	if cfg.DefaultVolumes[1].PVC != "pipeline-workspace" || !cfg.DefaultVolumes[1].ReadOnly {
		t.Fatalf("second default volume = %#v", cfg.DefaultVolumes[1])
	}
	if len(cfg.DefaultSecretVolumes) != 1 || cfg.DefaultSecretVolumes[0].Name != "gcp-credentials" || !cfg.DefaultSecretVolumes[0].ReadOnly {
		t.Fatalf("default secret volumes = %#v", cfg.DefaultSecretVolumes)
	}
}

func TestHelpCommandsExitSuccessfully(t *testing.T) {
	for _, args := range [][]string{{"--help"}, {"-h"}, {"help"}} {
		var stdout bytes.Buffer
		err := runCLI(context.Background(), args, &stdout, bytes.NewBuffer(nil), bytes.NewReader(nil))
		if err != nil {
			t.Fatalf("runCLI(%v) returned error: %v", args, err)
		}
		if !strings.Contains(stdout.String(), "Usage:") {
			t.Fatalf("runCLI(%v) did not print usage: %q", args, stdout.String())
		}
	}
}

func TestResolveConfigPrecedence(t *testing.T) {
	tmp := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWD) })
	t.Setenv("HOME", filepath.Join(tmp, "home"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, "xdg"))
	t.Setenv("MARKOVD_URL", "https://env.example")
	t.Setenv("MARKOVD_USERNAME", "env-user")
	if err := os.WriteFile(".markovd-cli-config.toml", []byte(`
server = "https://local.example"
username = "local-user"
password = "local-pass"
ssl_verify = false
`), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, rest, err := resolveConfig([]string{"--server", "https://flag.example", "--username", "flag-user", "health"}, bytes.NewReader(nil))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Server != "https://flag.example" || cfg.Username != "flag-user" {
		t.Fatalf("flags should win, got server=%q username=%q", cfg.Server, cfg.Username)
	}
	if cfg.Password != "local-pass" {
		t.Fatalf("local config password should be retained, got %q", cfg.Password)
	}
	if !cfg.InsecureSkipTLSVerify {
		t.Fatal("local ssl_verify=false should apply")
	}
	if len(rest) != 1 || rest[0] != "health" {
		t.Fatalf("rest = %#v", rest)
	}
}

func TestParseCommandFlagsAllowsFlagsAfterPositionals(t *testing.T) {
	fs := flagSetForTest()
	var wait bool
	var workflow string
	fs.BoolVar(&wait, "wait", false, "")
	fs.StringVar(&workflow, "workflow", "", "")
	pos, err := parseCommandFlags(fs, []string{"demo", "--workflow", "pipeline", "--wait"})
	if err != nil {
		t.Fatal(err)
	}
	if len(pos) != 1 || pos[0] != "demo" {
		t.Fatalf("positionals = %#v", pos)
	}
	if !wait || workflow != "pipeline" {
		t.Fatalf("flags not parsed: wait=%t workflow=%q", wait, workflow)
	}
}

func flagSetForTest() *flag.FlagSet {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	fs.SetOutput(bytes.NewBuffer(nil))
	return fs
}

func TestRunCreatePayloadOmitsBlankEntrypointAndSendsMounts(t *testing.T) {
	var got runCreateRequest
	client := fakeClient(func(r *http.Request) (int, any) {
		switch r.URL.Path {
		case "/api/v1/runs":
			if r.Header.Get("Authorization") != "Bearer token-1" {
				t.Fatalf("missing auth header: %q", r.Header.Get("Authorization"))
			}
			if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
				t.Fatal(err)
			}
			return http.StatusOK, run{RunID: "markov-run-1", WorkflowName: got.WorkflowName, Status: "pending"}
		default:
			return http.StatusNotFound, map[string]string{"error": "not found"}
		}
	})

	var stdout bytes.Buffer
	err := runRuns(context.Background(), client, cliConfig{Output: "json", Timeout: time.Second, PollInterval: time.Millisecond}, []string{
		"create", "graph-boundary-noop",
		"--var", "k=v",
		"--volume", "workspace-pvc:/workspace",
		"--secret-volume", "api-keys:/secrets/api-keys",
	}, &stdout)
	if err != nil {
		t.Fatal(err)
	}
	if got.WorkflowName != "graph-boundary-noop" {
		t.Fatalf("workflow name = %q", got.WorkflowName)
	}
	if got.WorkflowEntrypoint != "" {
		t.Fatalf("blank workflow entrypoint should be omitted/empty, got %q", got.WorkflowEntrypoint)
	}
	if got.Vars["k"] != "v" {
		t.Fatalf("vars = %#v", got.Vars)
	}
	if len(got.Volumes) != 1 || got.Volumes[0].PVC != "workspace-pvc" || got.Volumes[0].MountPath != "/workspace" {
		t.Fatalf("volumes = %#v", got.Volumes)
	}
	if len(got.SecretVolumes) != 1 || got.SecretVolumes[0].Secret != "api-keys" || got.SecretVolumes[0].MountPath != "/secrets/api-keys" {
		t.Fatalf("secret volumes = %#v", got.SecretVolumes)
	}
}

func TestRunCreatePayloadIncludesConfigDefaultMounts(t *testing.T) {
	var got runCreateRequest
	client := fakeClient(func(r *http.Request) (int, any) {
		switch r.URL.Path {
		case "/api/v1/runs":
			if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
				t.Fatal(err)
			}
			return http.StatusOK, run{RunID: "markov-run-1", WorkflowName: got.WorkflowName, Status: "pending"}
		default:
			return http.StatusNotFound, map[string]string{"error": "not found"}
		}
	})

	cfg := cliConfig{
		Output:       "json",
		Timeout:      time.Second,
		PollInterval: time.Millisecond,
		DefaultVolumes: []pvcMount{
			{Name: "pipeline-artifacts", MountPath: "/app/artifacts"},
		},
		DefaultSecretVolumes: []secretMount{
			{Name: "gcp-credentials", MountPath: "/home/pipelineagent/.config/gcloud", ReadOnly: true},
		},
	}
	var stdout bytes.Buffer
	err := runRuns(context.Background(), client, cfg, []string{
		"create", "graph-boundary-noop",
		"--volume", "pipeline-context:/app/.context",
	}, &stdout)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Volumes) != 2 {
		t.Fatalf("volumes = %#v", got.Volumes)
	}
	if got.Volumes[0].Name != "pipeline-artifacts" || got.Volumes[0].PVC != "pipeline-artifacts" || got.Volumes[0].MountPath != "/app/artifacts" {
		t.Fatalf("default volume = %#v", got.Volumes[0])
	}
	if got.Volumes[1].PVC != "pipeline-context" || got.Volumes[1].MountPath != "/app/.context" {
		t.Fatalf("cli volume = %#v", got.Volumes[1])
	}
	if len(got.SecretVolumes) != 1 || got.SecretVolumes[0].Secret != "gcp-credentials" || !got.SecretVolumes[0].ReadOnly {
		t.Fatalf("secret volumes = %#v", got.SecretVolumes)
	}
}

func TestRunCreateRejectsDuplicateConfigAndCLIMountPath(t *testing.T) {
	client := fakeClient(func(r *http.Request) (int, any) {
		t.Fatalf("unexpected API request to %s", r.URL.Path)
		return http.StatusInternalServerError, nil
	})
	cfg := cliConfig{
		Output:       "json",
		Timeout:      time.Second,
		PollInterval: time.Millisecond,
		DefaultVolumes: []pvcMount{
			{Name: "pipeline-artifacts", MountPath: "/app/artifacts"},
		},
	}
	var stdout bytes.Buffer
	err := runRuns(context.Background(), client, cfg, []string{
		"create", "graph-boundary-noop",
		"--volume", "other-pvc:/app/artifacts",
	}, &stdout)
	if err == nil || !strings.Contains(err.Error(), `duplicate mount path "/app/artifacts"`) {
		t.Fatalf("expected duplicate mount path error, got %v", err)
	}
}

func TestRunCreateWaitWithEntrypoint(t *testing.T) {
	getCount := 0
	var got runCreateRequest
	client := fakeClient(func(r *http.Request) (int, any) {
		switch r.URL.Path {
		case "/api/v1/runs":
			if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
				t.Fatal(err)
			}
			return http.StatusOK, run{RunID: "markov-run-2", WorkflowName: got.WorkflowName, Status: "pending"}
		case "/api/v1/runs/markov-run-2":
			getCount++
			return http.StatusOK, run{RunID: "markov-run-2", WorkflowName: got.WorkflowName, Status: "completed"}
		default:
			return http.StatusNotFound, map[string]string{"error": "not found"}
		}
	})

	var stdout bytes.Buffer
	err := runRuns(context.Background(), client, cliConfig{Output: "table", Timeout: time.Second, PollInterval: time.Millisecond}, []string{
		"create", "graph-boundary-noop", "--workflow", "pipeline", "--wait",
	}, &stdout)
	if err != nil {
		t.Fatal(err)
	}
	if got.WorkflowEntrypoint != "pipeline" {
		t.Fatalf("workflow entrypoint = %q", got.WorkflowEntrypoint)
	}
	if getCount == 0 {
		t.Fatal("wait did not poll run detail")
	}
}

func TestProjectSyncWaitByName(t *testing.T) {
	getCount := 0
	client := fakeClient(func(r *http.Request) (int, any) {
		switch r.URL.Path {
		case "/api/v1/projects":
			return http.StatusOK, []project{{ID: 7, Name: "ai-first-pipeline", Branch: "main", SyncStatus: "synced"}}
		case "/api/v1/projects/7/sync":
			return http.StatusOK, project{ID: 7, Name: "ai-first-pipeline", Branch: "main", SyncStatus: "syncing"}
		case "/api/v1/projects/7":
			getCount++
			return http.StatusOK, project{ID: 7, Name: "ai-first-pipeline", Branch: "main", SyncStatus: "synced"}
		default:
			return http.StatusNotFound, map[string]string{"error": "not found"}
		}
	})

	var stdout bytes.Buffer
	err := runProjects(context.Background(), client, cliConfig{Output: "table", Timeout: time.Second, PollInterval: time.Millisecond}, []string{
		"sync", "ai-first-pipeline", "--wait",
	}, &stdout)
	if err != nil {
		t.Fatal(err)
	}
	if getCount == 0 {
		t.Fatal("wait did not poll project detail")
	}
}

func TestAuthLoginGlobalPasswordStdin(t *testing.T) {
	var got map[string]string
	client := fakeClient(func(r *http.Request) (int, any) {
		switch r.URL.Path {
		case "/api/v1/auth/login":
			if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
				t.Fatal(err)
			}
			return http.StatusOK, map[string]string{"token": "token-stdin"}
		default:
			return http.StatusNotFound, map[string]string{"error": "not found"}
		}
	})

	var stdout bytes.Buffer
	err := runAuth(context.Background(), client, cliConfig{Output: "json", Username: "admin", Password: "admin", PasswordStdin: true}, []string{"login"}, &stdout, bytes.NewReader(nil))
	if err != nil {
		t.Fatal(err)
	}
	if got["username"] != "admin" || got["password"] != "admin" {
		t.Fatalf("login payload = %#v", got)
	}
}

func TestPreferencesSetParsesJSONValues(t *testing.T) {
	var got map[string]any
	client := fakeClient(func(r *http.Request) (int, any) {
		switch r.URL.Path {
		case "/api/v1/preferences":
			if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
				t.Fatal(err)
			}
			return http.StatusOK, got
		default:
			return http.StatusNotFound, map[string]string{"error": "not found"}
		}
	})

	var stdout bytes.Buffer
	err := runPreferences(context.Background(), client, cliConfig{Output: "json"}, []string{
		"set",
		`default_volumes=[{"name":"pipeline-artifacts","mount_path":"/app/artifacts"}]`,
	}, &stdout)
	if err != nil {
		t.Fatal(err)
	}
	volumes, ok := got["default_volumes"].([]any)
	if !ok || len(volumes) != 1 {
		t.Fatalf("default_volumes = %#v", got["default_volumes"])
	}
	first, ok := volumes[0].(map[string]any)
	if !ok || first["name"] != "pipeline-artifacts" || first["mount_path"] != "/app/artifacts" {
		t.Fatalf("first volume = %#v", volumes[0])
	}
}

func TestTLSConfigAllowsSelfSignedMode(t *testing.T) {
	c, err := newAPIClient(cliConfig{Server: "https://markovd.local", InsecureSkipTLSVerify: true})
	if err != nil {
		t.Fatal(err)
	}
	transport, ok := c.client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("transport type %T", c.client.Transport)
	}
	if transport.TLSClientConfig == nil || !transport.TLSClientConfig.InsecureSkipVerify {
		t.Fatal("expected InsecureSkipVerify to be enabled")
	}
}

func writeTestJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func fakeClient(fn func(*http.Request) (int, any)) *apiClient {
	return &apiClient{
		baseURL: "http://markovd.test",
		token:   "token-1",
		client:  &http.Client{Transport: fakeRoundTripper(fn)},
	}
}

type fakeRoundTripper func(*http.Request) (int, any)

func (f fakeRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	status, body := f(r)
	b, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(bytes.NewReader(b)),
		Request:    r,
	}, nil
}
