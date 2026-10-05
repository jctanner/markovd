---
title: Markov job does not send callbacks despite --callback arg being set
severity: high
component: markov
---

## Problem

When markovd spawns a markov job via the Kubernetes runner, the job completes successfully but never sends callback events back to markovd. The markovd events endpoint (`POST /api/v1/events`) receives no requests.

## Evidence

The job pod args confirm `--callback` and `--callback-header` are correctly passed:

```json
[
    "run", "/etc/markov/workflow.yaml", "--verbose",
    "--namespace", "ai-pipeline",
    "--callback", "http://markovd.ai-pipeline.svc.cluster.local:8080/api/v1/events",
    "--callback-header", "Authorization=Bearer 0df4922f..."
]
```

The callback URL is reachable from inside the cluster (verified with curl from a test pod).

However, the markov job logs contain zero callback-related output — no send attempts, no errors, no warnings. It behaves as if the `--callback` flag is being ignored entirely.

For comparison, when markov runs as a subprocess inside markovd (shell runner), callbacks work correctly and markovd logs show `POST /api/v1/events` requests.

## Additional Issue: Token Mismatch on Restart

`MARKOVD_CALLBACK_TOKEN` is not set in the deployment, so markovd generates a random token on each restart. If a job was created before a restart, its callback header will contain the old token, and the events endpoint will reject it with 401. This is a secondary issue — the primary bug is that markov doesn't attempt callbacks at all.

### Fix for token mismatch

Set a stable `MARKOVD_CALLBACK_TOKEN` in the deployment or in `pipeline-secrets`:

```yaml
- name: MARKOVD_CALLBACK_TOKEN
  value: "a-stable-token-value"
```

## Debug Run Log

Full output from `markov-run-4ed302fe` with `--debug`:

```
2026/04/24 22:37:57 [debug] flags: --workflow="" --namespace="ai-pipeline" --kubeconfig="" --state-store="/tmp/markov-state.db" --forks=0 --verbose=true
2026/04/24 22:37:57 [debug] flags: --callback=[http://markovd.ai-pipeline.svc.cluster.local:8080/api/v1/events] --callback-header=[Authorization=Bearer 2de91470bd0d99fc4b2ede267026448ea4fe39efd7bb14d8ecbbe359579b840d] --callback-tls-insecure=false --callback-buffer-size=1000
2026/04/24 22:37:57 [debug] flags: --var=[]
2026/04/24 22:37:57 [debug] state store: /tmp/markov-state.db
2026/04/24 22:37:57 [debug] namespace: using workflow namespace "ai-pipeline"
2026/04/24 22:37:57 [debug] k8s client: using in-cluster config (host=https://10.43.0.1:443)
2026/04/24 22:37:57 [debug] executors: registered [shell_exec http_request k8s_job]
2026/04/24 22:37:57 [debug] callback: parsing "http://markovd.ai-pipeline.svc.cluster.local:8080/api/v1/events"
2026/04/24 22:37:57 [debug] callback: created *callback.HTTPCallback for http://markovd.ai-pipeline.svc.cluster.local:8080/api/v1/events
2026/04/24 22:37:57 [debug] callbacks: 1 created from 1 --callback flags
2026/04/24 22:37:57 [debug] k8s client: using in-cluster config (host=https://10.43.0.1:443)
2026/04/24 22:37:57 [run:5952f7ce] starting workflow "hello"
2026/04/24 22:37:57 [run:5952f7ce]   forks: 2
2026/04/24 22:37:57 [run:5952f7ce]   namespace: ai-pipeline
2026/04/24 22:37:57 [run:5952f7ce]   vars: {"greeting":"hello from markov"}
2026/04/24 22:37:57 [run:5952f7ce]   steps: 3
2026/04/24 22:37:57 [run:5952f7ce] executing step "say-hello" (type: shell_exec)
2026/04/24 22:37:57 [run:5952f7ce]   resolved type: shell_exec -> shell_exec
2026/04/24 22:37:57 [run:5952f7ce]   param command: echo 'hello from markov'
2026/04/24 22:37:57 [run:5952f7ce]   registered "hello_result": map[exit_code:0 stderr: stdout:hello from markov
]
2026/04/24 22:37:57 [run:5952f7ce] step "say-hello" completed
2026/04/24 22:37:57 [run:5952f7ce] executing step "show-date" (type: shell_exec)
2026/04/24 22:37:57 [run:5952f7ce]   resolved type: shell_exec -> shell_exec
2026/04/24 22:37:57 [run:5952f7ce]   param command: date +%Y-%m-%d
2026/04/24 22:37:57 [run:5952f7ce]   registered "date_result": map[exit_code:0 stderr: stdout:2026-04-24
]
2026/04/24 22:37:57 [run:5952f7ce] step "show-date" completed
2026/04/24 22:37:57 [run:5952f7ce] executing step "combine" (type: shell_exec)
2026/04/24 22:37:57 [run:5952f7ce]   resolved type: shell_exec -> shell_exec
2026/04/24 22:37:57 [run:5952f7ce]   param command: echo 'Message: hello from markov
 on 2026-04-24
'
2026/04/24 22:37:57 [run:5952f7ce]   registered "combined": map[exit_code:0 stderr: stdout:Message: hello from markov
 on 2026-04-24

]
2026/04/24 22:37:57 [run:5952f7ce] step "combine" completed
2026/04/24 22:37:57 [run:5952f7ce] workflow "hello" completed
run 5952f7ce completed successfully
```

## Analysis

The callback object is created successfully:
- `callback: created *callback.HTTPCallback for http://...`
- `callbacks: 1 created from 1 --callback flags`

But during workflow execution there are zero callback send attempts — no `[debug] callback: sending`, no HTTP POST errors, nothing. The engine runs all steps and completes without ever invoking the callback sender. The callbacks slice is constructed but apparently never passed to (or called by) the workflow engine.

## Investigation Needed

1. Check where the callbacks slice is passed after creation in `cmd/markov/run.go` — is it wired into the engine?
2. Verify the engine's event emitter calls the callbacks on step/run lifecycle events
3. Check if there's a missing `engine.WithCallbacks(callbacks)` option or similar wiring
