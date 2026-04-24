# markovd

Web dashboard and API server for the [markov](https://github.com/jctanner/markov) workflow engine.

markovd provides an HTTP API and React UI on top of markov, letting users trigger workflow runs, monitor step execution in real time, and visualize pipeline structure through a browser.

## Screenshots

**Workflow graph** — fan-out visualization with status-colored nodes

![Graph view](docs/images/graph-view.png)

**Gantt chart** — step durations and concurrency timeline

![Gantt view](docs/images/gantt-view.png)

**Step table** — tabular view with fork context, status, and durations

![Table view](docs/images/table-view.png)

## Features

- **Run management** — trigger, list, and inspect workflow runs
- **Callback event receiver** — ingests real-time step lifecycle events from markov's HTTP callback system
- **Sub-workflow visibility** — full execution tree for fan-out (`for_each`), recursive, and nested sub-workflows
- **Workflow graph** — React Flow DAG visualization with status-colored nodes and step type icons
- **Gantt chart** — timeline view showing step durations and concurrency
- **Step table** — tabular view with fork context, status badges, and error details
- **Dark/light theme** — toggle with localStorage persistence
- **Auth** — local username/password (bcrypt + JWT), structured for future OIDC

## Quick start

Requires `podman-compose` (or `docker-compose`).

```bash
# Build and start all services (PostgreSQL, API, UI)
make compose-build
make compose-up

# Get the auto-generated admin password
make admin-password

# Open the UI
open http://localhost:5173
```

Log in with username `admin` and the password from `make admin-password`.

## Architecture

```
┌─────────────┐    callbacks     ┌──────────────┐     SQL      ┌────────────┐
│   markov    │ ──────────────▶ │   markovd    │ ──────────▶ │ PostgreSQL │
│  (CLI/bin)  │  POST /events   │  (Go API)    │             │            │
└─────────────┘                 └──────┬───────┘             └────────────┘
                                       │ REST API
                                       ▼
                                ┌──────────────┐
                                │  React UI    │
                                │  (Vite/TS)   │
                                └──────────────┘
```

When a run is triggered via the API, markovd:

1. Writes the workflow YAML to a temp file
2. Starts `markov run` as a subprocess with `--callback http://api:8080/api/v1/events`
3. markov POSTs step lifecycle events (`step_started`, `step_completed`, `gate_evaluated`, etc.) back to markovd
4. markovd upserts run/step state in PostgreSQL
5. The React UI polls the API and renders the execution graph, Gantt chart, or step table

## Services

| Service    | Port  | Description                          |
|------------|-------|--------------------------------------|
| `postgres` | 15432 | PostgreSQL 16 (data in Docker volume)|
| `api`      | 8082  | Go API server                        |
| `ui`       | 5173  | Vite dev server (React)              |

## API endpoints

All under `/api/v1/`, JWT-protected except login/register and the event receiver.

| Method | Path                   | Description                    |
|--------|------------------------|--------------------------------|
| POST   | `/auth/login`          | Login, returns JWT             |
| POST   | `/auth/register`       | Register new user              |
| GET    | `/runs`                | List runs                      |
| GET    | `/runs/:id`            | Run detail with all steps      |
| POST   | `/runs`                | Trigger a new run              |
| GET    | `/workflows`           | List workflows                 |
| GET    | `/workflows/:name`     | Get workflow by name           |
| POST   | `/workflows`           | Upload workflow YAML           |
| POST   | `/events`              | Callback event receiver        |

## Configuration

Environment variables (with defaults):

| Variable                | Default                          | Description                         |
|-------------------------|----------------------------------|-------------------------------------|
| `MARKOVD_PORT`          | `8080`                           | API listen port                     |
| `MARKOVD_DB_URL`        | `postgres://markovd:markovd@...` | PostgreSQL connection string        |
| `MARKOVD_JWT_SECRET`    | *(random)*                       | JWT signing secret                  |
| `MARKOVD_MARKOV_BIN`    | `markov`                         | Path to markov binary               |
| `MARKOVD_CALLBACK_TOKEN`| *(empty)*                        | Shared token for callback auth      |
| `MARKOVD_CALLBACK_URL`  | `http://localhost:PORT/api/v1/events` | URL markov posts callbacks to |

## Make targets

```
make help             Show all targets
make compose-build    Build container images
make compose-up       Start all services
make compose-down     Stop all services
make admin-password   Print the admin password from API logs
make admin-reset      Wipe all data and regenerate admin credentials
make build            Build the markovd Go binary
make deps             Install Go and JS dependencies
make dev              Instructions for local (non-container) development
```

## Project structure

```
markovd/
├── cmd/markovd/          Go entrypoint, config, server startup
├── internal/
│   ├── api/              Chi router, auth, run, workflow, event handlers
│   ├── auth/             AuthProvider interface, local bcrypt, JWT
│   ├── db/               PostgreSQL connection, migrations, queries
│   ├── models/           Shared types (Run, Step, Workflow, User, Event)
│   └── runner/           Runner interface, shell exec implementation
├── ui/
│   └── src/
│       ├── components/   WorkflowGraph, GanttChart, StepTable, Layout
│       ├── pages/        Login, Runs, RunDetail, Workflows, TriggerRun
│       ├── api.ts        Typed fetch wrapper with JWT
│       ├── auth.tsx      Auth context + protected routes
│       ├── theme.ts      Dark/light theme hook
│       └── index.css     Design system (CSS custom properties)
├── Dockerfile            Multi-stage build (Go API + React static assets)
├── podman-compose.yml    Full stack: PostgreSQL, API, UI dev server
└── Makefile
```
