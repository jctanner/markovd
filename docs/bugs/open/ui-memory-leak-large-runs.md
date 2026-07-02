# Bug: RunDetail UI consumes 11GB+ RAM on large workflow runs

## Summary

The RunDetail page (run view with Graph/Gantt/Table tabs) causes Firefox to consume 11+ GB of RAM when viewing a large batch workflow run. The tab becomes unresponsive and monopolizes CPU. The root cause is a combination of an oversized API response polled at high frequency, unbounded DOM rendering, and object churn from React state updates.

## Observed Behavior

- Firefox content process (single tab) consuming **11.4 GB RAM** and **14.5% CPU**
- Process has accumulated **~114 minutes of CPU time** over ~24 hours of the tab being open
- The tab is viewing a batch-rfe-pipeline run with **11,524 steps**
- The tab remained at 11 GB even after extended idle time — memory is not being released

## Root Cause Analysis

### 1. Massive API response polled every 3 seconds

`RunDetail.tsx` (line 82) polls the full run detail every 3 seconds:

```typescript
const interval = setInterval(loadRun, 3000);
```

The `GET /api/v1/runs/{runID}` endpoint returns the run metadata plus **all steps** in a single response. For the active batch run:

- **11,524 steps** in the response
- **18.5 MB** per response (JSON)
- Step breakdown: 8,525 completed, 2,963 skipped, 36 running

At a 3-second poll interval, the browser is:
- Downloading **~370 MB/minute** of JSON
- Parsing 18.5 MB of JSON into objects every 3 seconds
- Triggering a full React re-render cycle with each `setRun()` call

### 2. Full component tree rebuilt on every poll

Each `setRun()` call replaces the entire `run` state object, which triggers a full re-render of whichever view is active:

- **Graph view** (`WorkflowGraph.tsx`): Calls `buildGraph(steps)` inside `useMemo([steps])`, but since `steps` is a new array reference on every poll, the memo cache is invalidated every time. `buildGraph` creates a `Node` and `Edge` object for each step (or a `ForkSummaryNode` for collapsed groups), plus a `Map` entry in `stepMap`. ReactFlow then diffs and reconciles thousands of DOM elements.

- **Gantt view** (`GanttChart.tsx`): Similarly rebuilds `buildRows()` and `buildDisplayRows()` on every poll via `useMemo([steps])`. Renders an SVG with potentially 11,524 `<rect>` elements plus labels.

- **Table view** (`StepTable.tsx`): Renders all 11,524 rows in a single `<table>`.

None of these views implement virtualization — all rows/nodes are in the DOM simultaneously.

### 3. No object reuse between polls

The API returns a fresh JSON array every 3 seconds. Even though most steps haven't changed (8,525 completed + 2,963 skipped = 11,488 static steps), the entire array is deserialized into new objects. React sees new references and re-renders everything.

The previous `run` object (18.5 MB of parsed JSON + all the derived Node/Edge/Row objects) becomes garbage. With a 3-second interval, the GC has to collect ~6 copies per 18 seconds. If GC can't keep up with the allocation rate, memory grows unboundedly.

### 4. ReactFlow overhead at scale

ReactFlow (used in Graph view) maintains its own internal state for every node and edge — positions, dimensions, selection state, intersection data. At 11,524 steps, even with collapsed fork summaries, this internal state is substantial. The MiniMap component also renders a scaled-down version of every node.

## Impact

- **User experience**: The run detail page becomes unusable for batch runs, which are the primary use case
- **System resources**: 11 GB consumed by a single browser tab on a 64 GB host. This competes with the Vagrant VM running k3s (32 GB allocated)
- **Polling continues indefinitely**: Even when the tab is in the background, the 3-second poll continues, accumulating memory

## Reproduction

1. Start a batch workflow with `for_each` over 100+ items (e.g., `batch-rfe-pipeline` with ~160 RFE tickets)
2. Open the run detail page in the markovd UI
3. Leave the tab open for several hours
4. Observe memory growth via `ps aux --sort=-%mem | grep firefox`

The current active run (`markov-run-032f47ca`) has 11,524 steps and reproduces immediately.

## Suggested Fixes

### Short-term (reduce severity)

1. **Increase poll interval for large runs**: If `steps.length > 1000`, poll every 15-30 seconds instead of 3 seconds. Most steps are static — the user doesn't need sub-second freshness.

2. **Pause polling when tab is hidden**: Use `document.visibilityState` to stop polling when the tab is in the background.

3. **Add step count to the runs list endpoint**: Let the UI show step count in the run list without fetching all step data.

### Medium-term (fix the core problem)

4. **Server-side pagination for steps**: Add `?offset=N&limit=M` to the run detail endpoint. Only fetch the steps needed for the current view/scroll position.

5. **Delta polling**: Add `?since=<timestamp>` to only return steps that changed since the last poll. For a run with 11,524 steps where 36 are running, this would reduce the response from 18.5 MB to ~60 KB.

6. **Virtual scrolling**: Replace the full DOM render with a virtualized list (e.g., `react-window` or `@tanstack/virtual`). Only render the ~30 visible rows, not all 11,524.

### Long-term (architectural)

7. **WebSocket/SSE for step updates**: Push only changed steps instead of polling the full state. This eliminates both the bandwidth waste and the full-state replacement that breaks React memo caching.

8. **Separate summary and detail endpoints**: `GET /runs/{id}` returns run metadata + step summary counts. `GET /runs/{id}/steps?fork_id=X` returns steps for a specific fork branch on demand.

## Files Involved

| File | Issue |
|------|-------|
| `ui/src/pages/RunDetail.tsx:82` | 3-second poll interval with `setInterval(loadRun, 3000)` |
| `ui/src/pages/RunDetail.tsx:35` | `setRun()` replaces entire state, invalidating all memos |
| `ui/src/components/WorkflowGraph.tsx:668` | `useMemo(() => buildGraph(steps), [steps])` — cache always misses |
| `ui/src/components/GanttChart.tsx:267` | `useMemo` on `steps` — same cache invalidation issue |
| `ui/src/components/GanttChart.tsx:406-487` | Renders SVG rect for every step — no virtualization |
| `ui/src/components/WorkflowGraph.tsx:506-521` | Creates a ReactFlow Node for every step |
| API: `GET /api/v1/runs/{runID}` | Returns all steps inline — 18.5 MB for 11,524 steps |

## Post-Rebuild Observations (2026-04-27)

### Round 1: No code fixes

After killing the 11 GB Firefox tab and rebuilding/redeploying markovd, a fresh
tab was opened on the same run. No code fixes were applied — the rebuild only
confirmed the issue persists with a clean starting state.

- **T+0 min**: Tab opened, content process at ~1.4 GB
- **T+5 min**: 1.5 GB (steady, GC keeping up)
- **T+10 min**: 1.54 GB and climbing, ~100 MB growth over a few minutes
- **Trend**: Memory grows continuously, confirming the leak is not a one-time
  rendering cost but ongoing accumulation from the 3-second poll cycle.

### Round 2: With patches deployed

Patches were applied to the markovd UI codebase, rebuilt (`npm run build`
produced new bundle `index-D8S-48qJ.js`), and redeployed. Verified the new
image is running in the pod (UI assets dated Apr 27 16:38).

Initial confusion during deployment: `docker inspect` and `k3s crictl images`
report different digest formats (config digest vs unpacked image ID), making it
appear the new image wasn't picked up. Verified by checking file timestamps
inside the running container — the new build IS deployed.

However, memory growth continues after the patches:

- **T+0 min**: Fresh tab opened after hard-refresh, ~1.4 GB
- **T+10 min**: 1.5 GB
- **T+20 min**: 1.6 GB, still climbing at ~100 MB per few minutes

The patches did not resolve the memory growth. The core issue — polling 18.5 MB
of JSON every 3 seconds and rebuilding the full React component tree — requires
the medium-term fixes (server-side pagination, delta polling, or virtual
scrolling) to fully address. Incremental improvements to the client-side
rendering path cannot overcome the fundamental data volume problem.

## Environment

- markovd UI: React + ReactFlow + Vite
- Browser: Firefox 125 on Fedora 42
- Run: `markov-run-032f47ca` (batch-rfe-pipeline, ~160 RFE tickets)
- Steps: 11,524 (8,525 completed, 2,963 skipped, 36 running)
- API response: 18.5 MB per poll, polled every 3 seconds
