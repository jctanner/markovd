# Accessing a Running markovd Instance

## Base URL

markovd is exposed via ingress at `https://markovd.local`. TLS certificates are self-signed, so curl requires `-k` to skip verification.

## Authentication

markovd uses JWT bearer tokens. Obtain a token by posting credentials to the login endpoint:

```bash
curl -sk -X POST https://markovd.local/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin"}'
```

Response:

```json
{"token":"eyJhbGciOiJIUzI1NiIs..."}
```

Use the token in subsequent requests via the `Authorization: Bearer` header. Note that `-u user:pass` and `Authorization: Basic` do **not** work — the API requires JWT.

## Example: Fetch a Run

```bash
TOKEN=$(curl -sk -X POST https://markovd.local/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin"}' \
  | python3 -c "import json,sys; print(json.load(sys.stdin)['token'])")

curl -sk -H "Authorization: Bearer $TOKEN" \
  https://markovd.local/api/v1/runs/markov-run-19ab5533
```

## Common API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/v1/runs` | List all runs |
| GET | `/api/v1/runs/{run_id}` | Get run details including all steps |
| GET | `/api/v1/health` | Health check (no auth required) |

## Run Response Structure

Each run contains:

- `id` — numeric database ID
- `run_id` — string identifier (e.g. `markov-run-19ab5533`)
- `status` — `running`, `completed`, or `failed`
- `started_at`, `completed_at` — ISO 8601 timestamps
- `steps` — flat list of all step executions across all workflows

Each step has:

- `step_name` — the name from the workflow YAML
- `workflow_name` — which workflow this step belongs to
- `status` — `completed` or `skipped`
- `output_json` — JSON string with step output (e.g. set_fact values, shell_exec stdout)

## Web UI

markovd also serves a web UI at `https://markovd.local/` with a dashboard for viewing runs, steps, and logs.
