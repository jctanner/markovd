import { useState, useEffect, useRef, useCallback } from 'react';
import { useParams, Link, useNavigate } from 'react-router-dom';
import { api } from '../api';
import type { RunDetail as RunDetailType, Step } from '../api';
import StepTable from '../components/StepTable';
import WorkflowGraph from '../components/WorkflowGraph';
import GanttChart from '../components/GanttChart';
import StepDetailModal from '../components/StepDetailModal';
import RerunModal from '../components/RerunModal';

function badgeClass(status: string): string {
  const map: Record<string, string> = {
    pending: 'badge-pending',
    running: 'badge-running',
    completed: 'badge-completed',
    failed: 'badge-failed',
    cancelled: 'badge-failed',
  };
  return `badge ${map[status] || 'badge-pending'}`;
}

function pollInterval(stepCount: number, status: string): number {
  if (status === 'completed' || status === 'failed' || status === 'cancelled') return 30000;
  if (stepCount > 5000) return 30000;
  if (stepCount > 1000) return 15000;
  if (stepCount > 500) return 10000;
  return 3000;
}

function maxUpdatedAt(steps: Step[]): string | null {
  let max: string | null = null;
  for (const s of steps) {
    if (s.updated_at && (!max || s.updated_at > max)) max = s.updated_at;
  }
  return max;
}

function mergeSteps(existing: Step[], delta: Step[]): Step[] {
  const map = new Map<number, Step>();
  for (const s of existing) map.set(s.id, s);
  for (const s of delta) map.set(s.id, s);
  return [...map.values()].sort((a, b) => a.id - b.id);
}

type ViewMode = 'graph' | 'gantt' | 'table';

export default function RunDetail() {
  const { runID } = useParams<{ runID: string }>();
  const navigate = useNavigate();
  const [run, setRun] = useState<RunDetailType | null>(null);
  const [error, setError] = useState('');
  const [view, setView] = useState<ViewMode>('graph');
  const [selectedStep, setSelectedStep] = useState<Step | null>(null);
  const [showRerun, setShowRerun] = useState(false);
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const visibleRef = useRef(true);
  const runRef = useRef<RunDetailType | null>(null);
  const sinceRef = useRef<string | null>(null);
  const initialLoadDone = useRef(false);

  const loadRun = useCallback(async () => {
    if (!runID) return;
    try {
      const since = initialLoadDone.current ? sinceRef.current ?? undefined : undefined;
      const data = await api.getRun(runID, since);

      if (!initialLoadDone.current || !since) {
        initialLoadDone.current = true;
        sinceRef.current = maxUpdatedAt(data.steps);
        runRef.current = data;
        setRun(data);
      } else {
        if (data.steps.length === 0) {
          setRun(prev => {
            if (!prev) return data;
            if (prev.status !== data.status || prev.completed_at !== data.completed_at) {
              const updated = { ...data, steps: prev.steps };
              runRef.current = updated;
              return updated;
            }
            return prev;
          });
        } else {
          const deltaMax = maxUpdatedAt(data.steps);
          if (deltaMax) sinceRef.current = deltaMax;
          setRun(prev => {
            if (!prev) { runRef.current = data; return data; }
            const merged = mergeSteps(prev.steps, data.steps);
            const updated = { ...data, steps: merged };
            runRef.current = updated;
            return updated;
          });
        }
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load run');
    }
  }, [runID]);

  const scheduleNext = useCallback(() => {
    if (timerRef.current) clearTimeout(timerRef.current);
    const r = runRef.current;
    const interval = r ? pollInterval(r.steps.length, r.status) : 3000;
    timerRef.current = setTimeout(async () => {
      if (visibleRef.current) {
        await loadRun();
      }
      scheduleNext();
    }, interval);
  }, [loadRun]);

  useEffect(() => {
    initialLoadDone.current = false;
    sinceRef.current = null;
    runRef.current = null;
    loadRun().then(() => scheduleNext());
    return () => {
      if (timerRef.current) clearTimeout(timerRef.current);
    };
  }, [runID, loadRun, scheduleNext]);

  useEffect(() => {
    const onVisChange = () => {
      visibleRef.current = document.visibilityState === 'visible';
      if (visibleRef.current) loadRun();
    };
    document.addEventListener('visibilitychange', onVisChange);
    return () => document.removeEventListener('visibilitychange', onVisChange);
  }, [loadRun]);

  const handleCancel = async () => {
    if (!runID) return;
    try {
      await api.cancelRun(runID);
      await loadRun();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to cancel run');
    }
  };

  const handleDelete = async () => {
    if (!runID) return;
    if (!window.confirm(`Delete run ${runID}? This cannot be undone.`)) return;
    try {
      await api.deleteRun(runID);
      navigate('/runs');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete run');
    }
  };

  const handleRerunConfirm = async (
    vars: Record<string, string>,
    volumes: { name: string; pvc: string; mount_path: string }[],
    secretVolumes: { name: string; secret: string; mount_path: string }[],
  ) => {
    if (!run) return;
    try {
      const newRun = await api.createRun(run.workflow_name, vars, false, volumes, secretVolumes);
      setShowRerun(false);
      navigate(`/runs/${newRun.run_id}`);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to re-run');
    }
  };

  const handleStepClick = (step: Step) => setSelectedStep(step);

  if (error) return <div className="msg-error">{error}</div>;
  if (!run) return <div className="loading-state">Loading run...</div>;

  const interval = pollInterval(run.steps.length, run.status);

  return (
    <div>
      <div className="breadcrumb">
        <Link to="/runs">Runs</Link>
        <span className="breadcrumb-sep">/</span>
        <span>{run.run_id}</span>
      </div>

      <div className="page-header">
        <h1 className="page-title">Run {run.run_id}</h1>
        <div>
          {(run.status === 'running' || run.status === 'pending') && (
            <button className="btn btn-danger btn-sm" onClick={handleCancel}>Stop</button>
          )}
          {' '}
          <button className="btn btn-ghost btn-sm" onClick={handleDelete}>Delete</button>
          {' '}
          <button className="btn btn-primary btn-sm" onClick={() => setShowRerun(true)}>Re-run</button>
        </div>
      </div>

      <div className="meta-grid">
        <div className="meta-card">
          <div className="meta-label">Workflow</div>
          <div className="meta-value">{run.workflow_name}</div>
        </div>
        <div className="meta-card">
          <div className="meta-label">Status</div>
          <div className="meta-value">
            <span className={badgeClass(run.status)}>{run.status}</span>
          </div>
        </div>
        <div className="meta-card">
          <div className="meta-label">Started</div>
          <div className="meta-value mono">
            {run.started_at ? new Date(run.started_at).toLocaleString() : '-'}
          </div>
        </div>
        <div className="meta-card">
          <div className="meta-label">Completed</div>
          <div className="meta-value mono">
            {run.completed_at ? new Date(run.completed_at).toLocaleString() : '-'}
          </div>
        </div>
        <div className="meta-card">
          <div className="meta-label">Steps</div>
          <div className="meta-value mono">{run.steps.length.toLocaleString()}</div>
        </div>
        <div className="meta-card">
          <div className="meta-label">Refresh</div>
          <div className="meta-value mono">{interval / 1000}s</div>
        </div>
      </div>

      {(() => {
        try {
          const vars = JSON.parse(run.vars_json || '{}');
          const entries = Object.entries(vars);
          if (entries.length === 0) return null;
          return (
            <div className="run-vars-card">
              <div className="meta-label">Variables</div>
              <div className="run-vars-list">
                {entries.map(([k, v]) => (
                  <div key={k} className="run-vars-row">
                    <span className="run-vars-key">{k}</span>
                    <span className="run-vars-val">{String(v)}</span>
                  </div>
                ))}
              </div>
            </div>
          );
        } catch { return null; }
      })()}

      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 16 }}>
        <div className="section-heading" style={{ margin: 0, flex: 1 }}>Steps</div>
        <div className="view-toggle">
          <button
            className={`view-toggle-btn${view === 'graph' ? ' active' : ''}`}
            onClick={() => setView('graph')}
          >
            Graph
          </button>
          <button
            className={`view-toggle-btn${view === 'gantt' ? ' active' : ''}`}
            onClick={() => setView('gantt')}
          >
            Gantt
          </button>
          <button
            className={`view-toggle-btn${view === 'table' ? ' active' : ''}`}
            onClick={() => setView('table')}
          >
            Table
          </button>
        </div>
      </div>

      {view === 'graph' && <WorkflowGraph steps={run.steps} onStepClick={handleStepClick} />}
      {view === 'gantt' && <GanttChart steps={run.steps} onStepClick={handleStepClick} />}
      {view === 'table' && <StepTable steps={run.steps} onStepClick={handleStepClick} />}

      <StepDetailModal step={selectedStep} onClose={() => setSelectedStep(null)} />
      <RerunModal
        run={showRerun ? run : null}
        onClose={() => setShowRerun(false)}
        onConfirm={handleRerunConfirm}
      />
    </div>
  );
}
