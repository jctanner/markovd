# Markovd CLI Reference

`./bin/markovd-cli` is a command-line client for the markovd HTTP API. It is
intended for local terminal use, CI jobs, and automation that needs to sync
projects, import workflows, trigger runs, and wait for results without using the
browser UI.

## Build

Build the CLI binary from the markovd checkout:

```bash
make markovd-cli
```

This writes `bin/markovd-cli`. The `bin/` directory is ignored by Git, so the
CLI binary is a local build artifact rather than a tracked wrapper script.

Build both the API server and CLI:

```bash
make build
```

## Quick Start

Check the API health endpoint:

```bash
./bin/markovd-cli --server https://markovd.local \
  --insecure-skip-tls-verify \
  health
```

List projects using username/password authentication:

```bash
./bin/markovd-cli --server https://markovd.local \
  --insecure-skip-tls-verify \
  --username admin \
  --password admin \
  projects list
```

Sync a project and wait:

```bash
./bin/markovd-cli --server https://markovd.local \
  --insecure-skip-tls-verify \
  --username admin \
  --password admin \
  projects sync ai-first-pipeline --wait
```

Trigger a workflow entrypoint and wait:

```bash
./bin/markovd-cli --server https://markovd.local \
  --insecure-skip-tls-verify \
  --username admin \
  --password admin \
  runs create graph-boundary-noop \
  --workflow pipeline \
  --wait
```

Trigger a workflow with variable and volume payloads:

```bash
./bin/markovd-cli runs create var-demos-end-to-end \
  --workflow run-pipeline \
  --var run_pipeline=true \
  --volume pipeline-artifacts:/app/artifacts \
  --secret-volume gcp-credentials:/home/pipelineagent/.config/gcloud \
  --wait
```

## Global Flags

Global flags must appear before the resource command.

```text
--server URL
--username USER
--password PASS
--password-stdin
--token TOKEN
--config PATH
--output table|json|yaml
--timeout DURATION
--poll-interval DURATION
--insecure-skip-tls-verify
--ca-cert PATH
```

Defaults:

- `--server`: `http://localhost:8080`
- `--output`: `table`
- `--timeout`: `30m`
- `--poll-interval`: `2s`
- TLS certificate verification is enabled unless explicitly disabled.

Durations use Go duration syntax, such as `500ms`, `2s`, `10m`, or `1h`.

## Authentication

The CLI uses bearer tokens for API requests. It can get a token directly or log
in with username/password.

Token authentication:

```bash
./bin/markovd-cli --server https://markovd.local \
  --token "$MARKOVD_TOKEN" \
  projects list
```

Username/password flags:

```bash
./bin/markovd-cli --server https://markovd.local \
  --username admin \
  --password admin \
  projects list
```

Password from standard input:

```bash
printf '%s\n' "$MARKOVD_PASSWORD" | ./bin/markovd-cli \
  --server https://markovd.local \
  --username admin \
  --password-stdin \
  projects list
```

Interactive password prompt is used when a username is provided, no password is
available, and stdin is a terminal.

Login and print a token:

```bash
./bin/markovd-cli --server https://markovd.local \
  --username admin \
  --password-stdin \
  auth login
```

Login and save a token to the user config file:

```bash
./bin/markovd-cli --server https://markovd.local \
  --username admin \
  --password admin \
  --insecure-skip-tls-verify \
  auth login --save
```

`auth login --save` writes `~/.config/markovd/cli-config.toml`. It does not
write the project-local `./.markovd-cli-config.toml`.

## Environment Variables

The CLI reads these environment variables:

```text
MARKOVD_URL
MARKOVD_USERNAME
MARKOVD_PASSWORD
MARKOVD_TOKEN
MARKOVD_OUTPUT
MARKOVD_TIMEOUT
MARKOVD_POLL_INTERVAL
MARKOVD_INSECURE_SKIP_TLS_VERIFY
MARKOVD_CA_CERT
```

Example:

```bash
export MARKOVD_URL=https://markovd.local
export MARKOVD_USERNAME=admin
export MARKOVD_PASSWORD=admin
export MARKOVD_INSECURE_SKIP_TLS_VERIFY=true

./bin/markovd-cli projects list
```

## TLS and Self-Signed Certificates

TLS verification is enabled by default.

For local development instances using self-signed certificates:

```bash
./bin/markovd-cli --server https://markovd.local \
  --insecure-skip-tls-verify \
  health
```

For scripts:

```bash
MARKOVD_INSECURE_SKIP_TLS_VERIFY=true ./bin/markovd-cli \
  --server https://markovd.local \
  health
```

To trust a local CA without disabling TLS verification:

```bash
./bin/markovd-cli --server https://markovd.local \
  --ca-cert ./certs/local-ca.pem \
  health
```

Or:

```bash
export MARKOVD_CA_CERT=./certs/local-ca.pem
./bin/markovd-cli --server https://markovd.local health
```

## Config Files

The CLI supports TOML config files.

Lookup precedence:

1. CLI flags
2. Environment variables
3. Explicit `--config PATH`
4. `./.markovd-cli-config.toml`
5. `${XDG_CONFIG_HOME}/markovd/cli-config.toml`
6. `~/.config/markovd/cli-config.toml`
7. Built-in defaults

Project-local config:

```toml
server = "https://markovd.local"
username = "admin"
password = "admin"
ssl_verify = false

[defaults]
project = "ai-first-pipeline"
poll_interval = "2s"
timeout = "30m"
output = "table"

[[defaults.volumes]]
name = "pipeline-artifacts"
mount_path = "/app/artifacts"

[[defaults.volumes]]
name = "pipeline-context"
mount_path = "/app/.context"

[[defaults.secret_volumes]]
name = "gcp-credentials"
mount_path = "/home/pipelineagent/.config/gcloud"
read_only = true
```

Token-based config:

```toml
server = "https://markovd.local"
token = "eyJ..."
ssl_verify = false
```

Custom CA config:

```toml
server = "https://markovd.local"
ssl_verify = true
ca_cert = "./certs/local-ca.pem"
```

Supported top-level fields:

- `server`
- `username`
- `password`
- `token`
- `ssl_verify`
- `ca_cert`

Supported `[defaults]` fields:

- `output`
- `poll_interval`
- `timeout`
- `project`

`defaults.project` is parsed and retained for future project-oriented command
defaults. Current project commands still require an explicit project ID or
name.

Supported default mount sections:

```toml
[[defaults.volumes]]
name = "pipeline-artifacts"
mount_path = "/app/artifacts"

[[defaults.volumes]]
name = "workspace-volume"
pvc = "pipeline-workspace"
mount_path = "/app/workspace"
read_only = true

[[defaults.secret_volumes]]
name = "gcp-credentials"
mount_path = "/home/pipelineagent/.config/gcloud"
read_only = true

[[defaults.secret_volumes]]
name = "cloud-creds-volume"
secret = "gcp-credentials"
mount_path = "/var/run/gcp"
```

For `defaults.volumes`, `name` is used as the PVC name when `pvc` is omitted.
For `defaults.secret_volumes`, `name` is used as the Secret name when `secret`
is omitted. `mount_path` must be absolute. `read_only` is optional.

`runs create` sends configured default mounts automatically. Explicit
`--volume` and `--secret-volume` flags are appended for that run. Duplicate
mount paths are rejected so a config default cannot silently collide with a
command-line mount.

Security notes:

- `.markovd-cli-config.toml` is ignored by this repository.
- Do not commit plaintext passwords or tokens.
- Prefer `--password-stdin`, `MARKOVD_TOKEN`, or a local untracked config file
  for automation credentials.
- `ssl_verify = false` is useful for local development, but should not be used
  for production-like environments when a CA certificate is available.

## Output Formats

Use `--output table`, `--output json`, or `--output yaml`.

Human-oriented output:

```bash
./bin/markovd-cli --output table projects list
```

Machine-readable output:

```bash
./bin/markovd-cli --output json runs get markov-run-cfb29a83
```

For wait commands in JSON or YAML mode, the CLI returns the final object plus
wait metadata:

```json
{
  "status": "completed",
  "elapsed_seconds": 2.0,
  "run": {}
}
```

## Projects

List projects:

```bash
./bin/markovd-cli projects list
```

Get a project by ID or exact name:

```bash
./bin/markovd-cli projects get 1
./bin/markovd-cli projects get ai-first-pipeline
```

Create a project:

```bash
./bin/markovd-cli projects create \
  --name ai-first-pipeline \
  --url https://github.com/jctanner/ai-first-pipeline \
  --branch main
```

Delete a project:

```bash
./bin/markovd-cli projects delete ai-first-pipeline
```

Sync a project:

```bash
./bin/markovd-cli projects sync ai-first-pipeline
```

Sync and wait for terminal status:

```bash
./bin/markovd-cli projects sync ai-first-pipeline --wait
```

`projects sync --wait` exits `0` when the project reaches `synced`. It exits
non-zero if the project reaches `error`, times out, or the API request fails.

List importable workflow definitions from a synced project:

```bash
./bin/markovd-cli projects files ai-first-pipeline
```

Import a workflow file:

```bash
./bin/markovd-cli projects import ai-first-pipeline \
  var/markov-workflows/rfe-to-strategy.yaml \
  --kind file
```

Import a directory workflow:

```bash
./bin/markovd-cli projects import ai-first-pipeline \
  var/demos/end-to-end \
  --kind directory
```

## Workflows

List workflows:

```bash
./bin/markovd-cli workflows list
```

Get a workflow:

```bash
./bin/markovd-cli workflows get graph-boundary-noop
```

Create a workflow from a single file:

```bash
./bin/markovd-cli workflows create \
  --name my-workflow \
  --file ./workflow.yaml
```

Create a workflow from a directory:

```bash
./bin/markovd-cli workflows create \
  --name graph-boundary-noop \
  --directory docs/fixtures/workflows/graph-boundary-noop
```

Update a workflow:

```bash
./bin/markovd-cli workflows update graph-boundary-noop \
  --directory docs/fixtures/workflows/graph-boundary-noop
```

Validate a workflow payload without saving it:

```bash
./bin/markovd-cli workflows validate \
  --directory docs/fixtures/workflows/graph-boundary-noop
```

Generate a workflow diagram:

```bash
./bin/markovd-cli workflows diagram graph-boundary-noop --output json
```

Delete a workflow:

```bash
./bin/markovd-cli workflows delete graph-boundary-noop
```

## Runs

List runs:

```bash
./bin/markovd-cli runs list
```

Get run details:

```bash
./bin/markovd-cli runs get markov-run-cfb29a83
```

Trigger a workflow:

```bash
./bin/markovd-cli runs create graph-boundary-noop
```

Trigger a specific Markov workflow entrypoint:

```bash
./bin/markovd-cli runs create graph-boundary-noop --workflow pipeline
```

When `--workflow` is omitted or blank, the CLI omits `workflow_entrypoint` from
the API payload and markovd lets Markov use the workflow definition default.

Set variables:

```bash
./bin/markovd-cli runs create graph-boundary-noop \
  --var key=value \
  --var another=value
```

Load variables from JSON or YAML:

```bash
./bin/markovd-cli runs create graph-boundary-noop \
  --vars-file ./vars.yaml
```

Mount PVCs and Secrets:

```bash
./bin/markovd-cli runs create var-demos-end-to-end \
  --volume pipeline-artifacts:/app/artifacts \
  --volume pipeline-context:/app/.context \
  --secret-volume gcp-credentials:/home/pipelineagent/.config/gcloud
```

Mount syntax is strict:

- exactly one `:` separator
- non-empty PVC or Secret name
- absolute mount path
- duplicate mount paths are rejected

Default mounts from config are included automatically:

```toml
[[defaults.volumes]]
name = "pipeline-artifacts"
mount_path = "/app/artifacts"

[[defaults.secret_volumes]]
name = "gcp-credentials"
mount_path = "/home/pipelineagent/.config/gcloud"
read_only = true
```

With that config, this command sends both default mounts:

```bash
./bin/markovd-cli runs create var-demos-end-to-end --wait
```

You can still add run-specific mounts:

```bash
./bin/markovd-cli runs create var-demos-end-to-end \
  --volume pipeline-context:/app/.context \
  --wait
```

Trigger and wait:

```bash
./bin/markovd-cli runs create graph-boundary-noop \
  --workflow pipeline \
  --wait
```

Wait for an existing run:

```bash
./bin/markovd-cli runs wait markov-run-cfb29a83
```

Run wait behavior:

- exits `0` for `completed`
- exits non-zero for `failed` or `cancelled`
- exits `124` for timeout
- prints status transitions in table mode

Cancel a run:

```bash
./bin/markovd-cli runs cancel markov-run-cfb29a83
```

Delete a run:

```bash
./bin/markovd-cli runs delete markov-run-cfb29a83
```

Read run logs:

```bash
./bin/markovd-cli runs logs markov-run-cfb29a83
```

Follow streamed run logs:

```bash
./bin/markovd-cli runs logs markov-run-cfb29a83 --follow
```

## PVCs and Secrets

List PVCs:

```bash
./bin/markovd-cli pvcs list
```

List Secrets:

```bash
./bin/markovd-cli secrets list
```

These commands use markovd's API; the local machine does not need direct
Kubernetes credentials.

## Preferences

Get preferences:

```bash
./bin/markovd-cli preferences get
```

Set preferences:

```bash
./bin/markovd-cli preferences set default_volumes='[{"name":"pipeline-artifacts","mount_path":"/app/artifacts"}]'
```

The preferences command is a low-level API wrapper. Prefer using the UI for
complex preference edits until the CLI grows structured preference flags.

## Exit Codes

- `0`: command succeeded
- `1`: local validation, command usage, run failure, project sync failure, or
  other non-API error
- `3`: API returned a non-2xx response
- `124`: wait timeout

## Common Recipes

Use a local config file for a development instance:

```bash
cat > .markovd-cli-config.toml <<'EOF'
server = "https://markovd.local"
username = "admin"
password = "admin"
ssl_verify = false

[defaults]
timeout = "30m"
poll_interval = "2s"
output = "table"

[[defaults.volumes]]
name = "pipeline-artifacts"
mount_path = "/app/artifacts"

[[defaults.secret_volumes]]
name = "gcp-credentials"
mount_path = "/home/pipelineagent/.config/gcloud"
read_only = true
EOF

./bin/markovd-cli projects list
```

Sync, import, trigger, and wait:

```bash
./bin/markovd-cli projects sync ai-first-pipeline --wait

./bin/markovd-cli projects import ai-first-pipeline \
  var/demos/end-to-end \
  --kind directory

./bin/markovd-cli runs create var-demos-end-to-end \
  --workflow run-pipeline \
  --volume pipeline-artifacts:/app/artifacts \
  --wait
```

Use JSON output in automation:

```bash
./bin/markovd-cli --output json runs create graph-boundary-noop \
  --workflow pipeline \
  --wait
```

Use environment variables in CI:

```bash
export MARKOVD_URL=https://markovd.local
export MARKOVD_TOKEN="$CI_MARKOVD_TOKEN"
export MARKOVD_INSECURE_SKIP_TLS_VERIFY=true

./bin/markovd-cli --output json projects sync ai-first-pipeline --wait
```

## Current Limitations

- Project and workflow name lookup uses exact matches.
- Project names must be unique for name-based commands.
- `auth login --save` writes user config, not project-local config.
- `preferences set` is intentionally minimal and does not yet provide
  structured flags for default PVC and Secret mount lists.
- `runs logs --follow` streams the API's current server-sent events format and
  prints `data:` lines.
