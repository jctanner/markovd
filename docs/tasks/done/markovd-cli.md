# Task: Add Markovd API CLI

## Goal

Add `./bin/markovd-cli`, a command-line client that provides full markovd API
CRUD coverage and makes the main automation flows easy to script.

Plan: [Markovd CLI Plan](../../plans/003-markovd-cli.md)

## Primary Use Cases

- Sync a project and wait for the result.
- Trigger a workflow, optionally set `--workflow`, optionally set PVC or Secret
  volume mounts, and wait for the result.

## Acceptance Criteria

- [x] `./bin/markovd-cli` exists and can be run from a checkout.
- [x] CLI authentication works with token, environment variables, and explicit
      username/password login.
- [x] Username/password login supports flags, environment variables,
      interactive prompt, and `--password-stdin`.
- [x] CLI config supports `./.markovd-cli-config.toml` for project-local
      settings.
- [x] Config precedence is flags, environment, explicit `--config`, local
      `.markovd-cli-config.toml`, user config, then defaults.
- [x] Local TOML config can define `server`, `username`, `password`, `token`,
      `ssl_verify`, and `ca_cert`.
- [x] HTTPS connections verify certificates by default.
- [x] Self-signed development instances work with
      `--insecure-skip-tls-verify` or `MARKOVD_INSECURE_SKIP_TLS_VERIFY=true`.
- [x] Custom local CA trust works with `--ca-cert` or `MARKOVD_CA_CERT`.
- [x] Project CRUD commands cover list, get, create, delete, sync, files, and
      import.
- [x] `projects sync <id-or-name> --wait` polls until `synced` or `error` and
      exits with the correct status code.
- [x] Workflow CRUD commands cover list, get, create, update, delete, validate,
      and diagram.
- [x] Run commands cover list, get, create, wait, cancel, delete, and logs.
- [x] `runs create` omits `workflow_entrypoint` when `--workflow` is blank.
- [x] `runs create --workflow ENTRYPOINT` sends `workflow_entrypoint`.
- [x] `runs create --volume PVC:/path` sends a PVC mount payload.
- [x] `runs create --secret-volume SECRET:/path` sends a Secret mount payload.
- [x] `runs create --wait` waits until terminal status and returns non-zero for
      failed, cancelled, timeout, or API errors.
- [x] Machine-readable JSON output is available for automation.
- [x] Verification is recorded before moving this task to done.

## Files Likely Involved

- `bin/markovd-cli`
- `cmd/markovd-cli/`
- `internal/api/`
- `docs/plans/003-markovd-cli.md`

## Verification

- `env GOCACHE=/tmp/go-build-cache go test ./cmd/markovd-cli`
- `env GOCACHE=/tmp/go-build-cache go test ./...`
- `env GOCACHE=/tmp/go-build-cache ./bin/markovd-cli` printed CLI usage.
- `env GOCACHE=/tmp/go-build-cache ./bin/markovd-cli --server https://markovd.local --insecure-skip-tls-verify --output json health`
  returned `{"status":"healthy"}`.
- `env GOCACHE=/tmp/go-build-cache ./bin/markovd-cli --server https://markovd.local --insecure-skip-tls-verify --username admin --password admin --output json projects list`
  authenticated and listed projects from the running instance.
- `printf '%s\n' admin | env GOCACHE=/tmp/go-build-cache ./bin/markovd-cli --server https://markovd.local --insecure-skip-tls-verify --username admin --password-stdin --output json auth login`
  authenticated through `--password-stdin`.
- `env GOCACHE=/tmp/go-build-cache ./bin/markovd-cli --server https://markovd.local --insecure-skip-tls-verify --username admin --password admin --timeout 2m --poll-interval 2s runs create graph-boundary-noop --workflow pipeline --wait`
  triggered `markov-run-cfb29a83` and waited until it completed.

## Status

Done
