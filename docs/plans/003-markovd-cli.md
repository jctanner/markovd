# Markovd CLI Plan

## Goal

Add `./bin/markovd-cli`, a command-line client for the markovd API.

The CLI should cover full API CRUD for automation, but the first-class use
cases are:

- sync a project and wait for it to finish
- trigger a workflow, optionally pass a Markov workflow entrypoint override,
  optionally mount a PVC or Secret volume, then wait for the run result

The CLI should make markovd usable from scripts, CI jobs, and local terminal
workflows without requiring browser interaction.

## Primary Workflows

### Sync Project and Wait

Target command shape:

```bash
./bin/markovd-cli projects sync <project-id-or-name> --wait
```

Expected behavior:

- authenticate to markovd
- find the project by ID or name
- call `POST /api/v1/projects/{id}/sync`
- if `--wait` is set, poll `GET /api/v1/projects/{id}` until
  `sync_status` is terminal
- exit `0` when the project reaches `synced`
- exit non-zero when the project reaches `error`, times out, or the API fails
- print a human-readable status stream by default
- support `--output json` for scripting

### Trigger Workflow and Wait

Target command shape:

```bash
./bin/markovd-cli runs create <workflow-name> \
  --workflow <entrypoint> \
  --var key=value \
  --volume workspace-pvc:/workspace \
  --secret-volume api-keys:/secrets/api-keys \
  --wait
```

Expected behavior:

- authenticate to markovd
- call `POST /api/v1/runs`
- include `workflow_entrypoint` only when `--workflow` is non-empty
- include `volumes` and `secret_volumes` only when specified
- if `--wait` is set, poll `GET /api/v1/runs/{run_id}` until terminal
- exit `0` for completed runs
- exit non-zero for failed, cancelled, timeout, or API errors
- optionally stream run logs with `--logs` once the API can support this
  reliably enough for CLI use

## Existing API Surface

The current API already exposes most of the required operations:

- `POST /api/v1/auth/login`
- `GET /api/v1/health`
- `GET /api/v1/projects`
- `POST /api/v1/projects`
- `GET /api/v1/projects/{id}`
- `DELETE /api/v1/projects/{id}`
- `POST /api/v1/projects/{id}/sync`
- `GET /api/v1/projects/{id}/files`
- `POST /api/v1/projects/{id}/import`
- `GET /api/v1/workflows`
- `POST /api/v1/workflows`
- `POST /api/v1/workflows/validate`
- `GET /api/v1/workflows/{name}`
- `PUT /api/v1/workflows/{name}`
- `DELETE /api/v1/workflows/{name}`
- `GET /api/v1/workflows/{name}/diagram`
- `GET /api/v1/runs`
- `POST /api/v1/runs`
- `GET /api/v1/runs/{run_id}`
- `POST /api/v1/runs/{run_id}/cancel`
- `DELETE /api/v1/runs/{run_id}`
- `GET /api/v1/runs/{run_id}/logs`
- `GET /api/v1/runs/{run_id}/logs/stream`
- `GET /api/v1/pvcs`
- `GET /api/v1/secrets`
- `GET /api/v1/preferences`
- `PUT /api/v1/preferences`
- job and active-job endpoints under `/api/v1/jobs/*`

The CLI should wrap existing endpoints first. Add API changes only where the
current surface cannot provide reliable automation behavior.

## Command Layout

Use noun-first commands that map closely to API resources:

```text
markovd-cli auth login [--username USER] [--password PASS | --password-stdin]
markovd-cli health

markovd-cli projects list
markovd-cli projects get <id-or-name>
markovd-cli projects create --name NAME --url URL [--branch BRANCH]
markovd-cli projects delete <id-or-name>
markovd-cli projects sync <id-or-name> [--wait]
markovd-cli projects files <id-or-name>
markovd-cli projects import <id-or-name> <path> [--kind file|directory]

markovd-cli workflows list
markovd-cli workflows get <name>
markovd-cli workflows create --name NAME --file PATH
markovd-cli workflows create --name NAME --directory PATH
markovd-cli workflows update <name> --file PATH
markovd-cli workflows update <name> --directory PATH
markovd-cli workflows delete <name>
markovd-cli workflows validate --file PATH
markovd-cli workflows validate --directory PATH
markovd-cli workflows diagram <name>

markovd-cli runs list
markovd-cli runs get <run-id>
markovd-cli runs create <workflow-name> [flags]
markovd-cli runs wait <run-id>
markovd-cli runs cancel <run-id>
markovd-cli runs delete <run-id>
markovd-cli runs logs <run-id> [--follow]

markovd-cli pvcs list
markovd-cli secrets list

markovd-cli preferences get
markovd-cli preferences set KEY=VALUE
```

The initial implementation can support the primary workflows and skeletons for
the rest, then fill in CRUD commands incrementally.

## Global Flags and Configuration

Global flags:

- `--server URL`, default from `MARKOVD_URL`, then `http://localhost:8080`
- `--username USER`, default from `MARKOVD_USERNAME`
- `--password PASS`, default from `MARKOVD_PASSWORD`
- `--password-stdin`, read the password from standard input
- `--token TOKEN`, default from `MARKOVD_TOKEN`
- `--config PATH`, explicit config file path
- `--output table|json|yaml`, default `table`
- `--timeout DURATION`, default command-specific
- `--poll-interval DURATION`, default `2s`
- `--insecure-skip-tls-verify`, accept self-signed or otherwise untrusted TLS
  certificates
- `--ca-cert PATH`, trust an additional PEM CA certificate for markovd

Authentication behavior:

- prefer explicit `--token`
- otherwise use environment variables
- otherwise load config values
- otherwise login with username/password when both are available
- support username/password through flags, `MARKOVD_USERNAME` and
  `MARKOVD_PASSWORD`, interactive prompt, and `--password-stdin`
- `auth login` may save token, server config, and TLS settings for later
  commands

The CLI must avoid printing passwords or tokens in normal output.

TLS behavior:

- verify TLS certificates by default
- allow `--insecure-skip-tls-verify` for local clusters and development
  instances with self-signed certificates
- allow `MARKOVD_INSECURE_SKIP_TLS_VERIFY=true` for non-interactive scripts
- support `--ca-cert PATH` and `MARKOVD_CA_CERT` for trusting a local CA
  without disabling verification
- persist the selected TLS behavior in config only when the user explicitly
  logs in or saves configuration

## Configuration Files

Support TOML configuration so local projects can keep their markovd connection
settings close to the workflows they operate on.

Config lookup precedence:

1. explicit CLI flags
2. environment variables
3. `--config PATH`
4. `./.markovd-cli-config.toml`
5. `${XDG_CONFIG_HOME}/markovd/cli-config.toml`
6. `~/.config/markovd/cli-config.toml`
7. built-in defaults

Project-local config should be intentionally simple:

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
```

Equivalent token-based config:

```toml
server = "https://markovd.local"
token = "eyJ..."
ssl_verify = false
```

TLS config:

```toml
server = "https://markovd.local"
ssl_verify = true
ca_cert = "./certs/local-ca.pem"
```

Config field mapping:

- `server`: same as `--server`
- `username`: same as `--username`
- `password`: same as `--password`
- `token`: same as `--token`
- `ssl_verify = false`: same behavior as `--insecure-skip-tls-verify`
- `ca_cert`: same as `--ca-cert`
- `defaults.output`: same as `--output`
- `defaults.poll_interval`: same as `--poll-interval`
- `defaults.timeout`: same as `--timeout`
- `defaults.project`: default project name for project-oriented commands when
  the command can safely infer it

Security expectations:

- never print config-loaded credentials in normal output
- warn when `auth login --save` would write a password or token into
  `./.markovd-cli-config.toml`
- recommend adding `.markovd-cli-config.toml` to `.gitignore` when it contains
  credentials
- allow local config to contain non-secret connection settings, such as
  `server`, `ssl_verify`, and `ca_cert`
- prefer tokens or `--password-stdin` over committed plaintext passwords

## Run Flags

`runs create` should accept:

- `--workflow ENTRYPOINT`: maps to `workflow_entrypoint`; omit the JSON field
  when blank
- `--var KEY=VALUE`: repeatable, maps to `vars`
- `--vars-file PATH`: JSON or YAML file merged into `vars`
- `--debug`: maps to `debug`
- `--volume PVC:MOUNT_PATH`: repeatable, maps to `volumes`
- `--secret-volume SECRET:MOUNT_PATH`: repeatable, maps to `secret_volumes`
- `--wait`: wait for terminal run status
- `--logs`: print logs while waiting when supported
- `--timeout DURATION`: wait timeout override

Volume parsing should be strict:

- require exactly one `:`
- require non-empty volume or secret name
- require absolute mount path
- reject duplicate mount paths unless an explicit override behavior is added

## Wait Semantics

Terminal run statuses:

- success: `completed`
- failure: `failed`, `cancelled`

Terminal project sync statuses:

- success: `synced`
- failure: `error`

While waiting, the CLI should:

- print status transitions and elapsed time in table/text mode
- suppress progress noise in `--output json` unless `--verbose` is set
- return a non-zero exit status for failure terminal states
- return a distinct non-zero exit status for timeout

Initial wait implementation can poll existing GET endpoints. A later
implementation can switch to SSE if the API grows a stable event stream for run
and project state.

## Output Contract

Human output should be concise and scriptable enough for terminals:

```text
project 12 ai-first-pipeline syncing
project 12 ai-first-pipeline synced in 4.2s
```

```text
run markov-run-abc123 running
run markov-run-abc123 completed in 31.8s
```

JSON output should emit the API object for non-waiting commands. For waiting
commands, emit the final object plus minimal wait metadata:

```json
{
  "status": "completed",
  "elapsed_seconds": 31.8,
  "run": {}
}
```

## Implementation Notes

Place the executable at `./bin/markovd-cli`.

Preferred implementation options:

- Go command under `cmd/markovd-cli` with `bin/markovd-cli` as a small wrapper
  that runs or builds it.
- Go keeps dependency and distribution behavior aligned with the backend.
- Use the standard `net/http` client unless a lightweight CLI package is
  already present or deliberately introduced.

The client package should centralize:

- base URL handling
- auth token injection
- JSON request and response handling
- consistent error decoding from `{"error":"..."}`
- pagination support if future APIs add pagination

The command package should keep parsing, output formatting, and exit-code logic
separate from the HTTP client.

## API Gaps and Follow-Up Candidates

The first implementation can work against existing endpoints. Useful follow-up
API improvements:

- add project lookup by name, or document that CLI resolves names by listing
  projects first
- add `POST /api/v1/projects/{id}/sync?wait=true` only if server-side waits
  become preferable to client polling
- add a consistent run/job event stream for CLI `--logs --wait`
- include `workflow_entrypoint` in the persisted run record for auditability
- return structured validation errors for workflow import failures

## Verification Plan

Unit tests:

- flag parsing for project IDs/names, variables, PVC mounts, Secret mounts,
  output format, timeout, and wait flags
- API client request construction, including omission of blank
  `workflow_entrypoint`
- wait loop behavior for success, failure, and timeout
- output formatting for table and JSON modes

Integration tests:

- start markovd with a temporary database and shell runner
- login through the CLI
- create or sync a test project
- import a fast fixture workflow
- trigger a no-op workflow with and without `--workflow`
- trigger with a PVC mount payload using a fake runner or API-level assertion
- wait for completion and assert exit status

Manual verification:

```bash
./bin/markovd-cli --server https://markovd.local auth login \
  --username admin --password admin --insecure-skip-tls-verify

printf '%s\n' admin | ./bin/markovd-cli --server https://markovd.local \
  auth login --username admin --password-stdin --insecure-skip-tls-verify

./bin/markovd-cli --server https://markovd.local projects sync ai-first-pipeline --wait

./bin/markovd-cli --server https://markovd.local runs create graph-boundary-noop \
  --workflow pipeline \
  --volume workspace-pvc:/workspace \
  --wait
```

## Open Questions

- Should `./bin/markovd-cli` be a compiled binary committed by release
  packaging, or a wrapper that executes `go run ./cmd/markovd-cli` in the repo?
- Should `runs create --logs --wait` stream Kubernetes job logs, Markov callback
  logs, or both?
- Should workflow import have a dedicated command that combines project file
  discovery, import, and optional run triggering?
- Should config support multiple named profiles for different markovd
  instances?

## Non-Goals

- Do not replace the browser UI.
- Do not add direct database access.
- Do not shell out to `curl` internally.
- Do not require Kubernetes access from the client machine; the CLI should use
  the markovd API for cluster interactions.
