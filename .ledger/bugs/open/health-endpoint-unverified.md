---
title: Verify or implement /api/v1/health endpoint
severity: medium
component: api
---

## Problem

The Kubernetes deployment manifest (`deploy/k8s/15-markovd.yaml`) configures liveness and readiness probes pointing to `/api/v1/health` on port 8080:

```yaml
livenessProbe:
  httpGet:
    path: /api/v1/health
    port: 8080
  initialDelaySeconds: 10
  periodSeconds: 10
readinessProbe:
  httpGet:
    path: /api/v1/health
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 5
```

This endpoint has not been verified to exist in the markovd codebase. If it doesn't exist, the pod will fail health checks and be repeatedly restarted by kubelet.

## Expected Behavior

`GET /api/v1/health` should return HTTP 200 with a JSON body indicating service status. Ideally it should also check PostgreSQL connectivity so the readiness probe reflects actual availability.

## Action Required

1. Check if `/api/v1/health` is already implemented in `cmd/markovd` or `internal/` routes.
2. If not, implement a health endpoint that:
   - Returns 200 when the server is running (liveness).
   - Returns 200 only when the database connection is healthy (readiness), or split into separate endpoints (`/healthz` and `/readyz`).
3. If a health endpoint exists at a different path, update the K8s manifest to match.
