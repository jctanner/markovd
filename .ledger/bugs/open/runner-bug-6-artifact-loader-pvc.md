# Bug: load_artifact cannot read from PVCs after job completion

## Summary

`load_artifact` steps with `source: k8s` fail when the producing job pod has already terminated. The artifact loader scans running pods to find the file, but the agent job pod is complete and gone. The artifact *does* exist on the PVC — the dashboard pod (which mounts the same PVC) can browse it — but markov has no way to access it.

## Reproduction

1. Run `rfe-pipeline-with-gates.yaml` workflow
2. `rfe_speedrun` agent job completes successfully, writes `/app/artifacts/rfe-tasks/RHAIRFE-1981.md` to the `pipeline-artifacts` PVC
3. Next step `load_rfe_results` fails:

```
step "load_rfe_results": loading artifacts: artifact "task": reading /app/artifacts/rfe-tasks/RHAIRFE-1981.md: artifact not found at /app/artifacts/rfe-tasks/RHAIRFE-1981.md in any running pod in ai-pipeline
```

4. File confirmed present on the PVC via dashboard file browser.

## Root cause

The `source: k8s` artifact loader appears to resolve file paths by exec-ing into running pods. After the agent job pod terminates, there's no running pod to exec into, so the artifact is unreachable — even though it's sitting on a PVC that's still bound in the namespace.

## Expected behavior

`load_artifact` with `source: k8s` should be able to read files from PVCs directly, similar to how one would read from S3 or any other persistent storage backend. The workflow YAML already declares the PVC name and mount path in `step_types.agent_job.volumes` — the artifact loader should be able to use that information to access the data without depending on a running pod.

## Workflow reference

```yaml
step_types:
  agent_job:
    base: k8s_job
    volumes:
      - name: pipeline-artifacts
        mount: /app/artifacts
        pvc: pipeline-artifacts

# ...

- name: load_rfe_results
  type: load_artifact
  artifacts:
    task:
      path: "/app/artifacts/rfe-tasks/{{ issue }}.md"
      format: markdown
      source: k8s
```

## Environment

- k3s cluster, namespace `ai-pipeline`
- PVC `pipeline-artifacts` exists and is bound
- markov image: `markov:latest`
- markovd image: `markovd:latest`
