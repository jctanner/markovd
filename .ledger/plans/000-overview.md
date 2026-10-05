# markovd

Long-running API + frontend daemon for Markov. Separate project (`jctanner/markovd`), separate repo.

## Relationship to markov

- `markovd` invokes the `markov` CLI as a container image — no Go import, no shared code
- Contract between them: CLI interface (flags, exit codes) + shared PostgreSQL schema
- `markov` remains fully usable standalone without `markovd`

## Stack

- **Language:** Go (unified with markov; better JWT/OIDC ecosystem than Flask)
- **API framework:** Chi or Echo
- **Frontend:** React + React Flow (graph visualization) + Dagre (auto-layout)
- **Database:** PostgreSQL (shared with markov engine processes)
- **Auth:** External identity provider (Keycloak/Dex/Okta) — markovd validates JWTs, never implements auth itself
- **RBAC:** Casbin for policy evaluation
- **Real-time:** WebSocket or SSE for streaming run status and events

## Core Responsibilities

- User authentication (OIDC/SSO via external IdP)
- RBAC and multi-tenancy (org/team scoping, namespace isolation)
- Workflow catalog (upload, validate, version workflow YAML files)
- Run management (trigger runs, cancel, view status/history)
- Job scheduling (create K8s Jobs or podman containers that run `markov run`)
- Human gate approvals (approve/reject via UI or API)
- Diagram/visualization (interactive React Flow graphs, replacing static Mermaid)
- Event streaming (real-time step progress via WebSocket/SSE)

## API Surface

- `POST /api/v1/runs` — trigger a workflow run
- `GET /api/v1/runs/:id` — run status + steps
- `DELETE /api/v1/runs/:id` — cancel a run
- `POST /api/v1/runs/:id/resume` — resume a failed run
- `PATCH /api/v1/runs/:id/gates/:name` — approve/reject a human gate
- `GET /api/v1/runs/:id/events` — WebSocket/SSE event stream
- `GET /api/v1/workflows` — list workflow definitions
- `POST /api/v1/workflows` — upload a workflow YAML
- `GET /api/v1/workflows/:name/diagram` — workflow graph data

## Deployment

- **K8s (production):** markovd runs as a Deployment; spawns `markov run` as K8s Jobs; PostgreSQL + Keycloak as separate services
- **podman-compose (dev):** markovd + PostgreSQL + Keycloak containers; markov runs as spawned containers
- **Multi-tenancy:** K8s namespaces as tenancy boundary; OIDC group claims map to namespace-level roles

## What markovd Does NOT Do

- Run workflows itself (delegates to `markov` binary)
- Implement its own auth (delegates to external IdP)
- Share Go packages with markov (contract is CLI + database)

## Open Questions

- PostgreSQL schema ownership — does markov or markovd own migrations?
- Event storage — separate events table or derive from run/step state?
- Workflow file storage — database blobs, git repo, or filesystem?
- Container image registry for markov — how does markovd know which image to launch?
