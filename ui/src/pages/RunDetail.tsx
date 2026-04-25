import { useState, useEffect } from 'react';
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

type ViewMode = 'graph' | 'gantt' | 'table';

export default function RunDetail() {
  const { runID } = useParams<{ runID: string }>();
  const navigate = useNavigate();
  const [run, setRun] = useState<RunDetailType | null>(null);
  const [error, setError] = useState('');
  const [view, setView] = useState<ViewMode>('graph');
  const [selectedStep, setSelectedStep] = useState<Step | null>(null);
  const [showRerun, setShowRerun] = useState(false);

  const loadRun = async () => {
    if (!runID) return;
    try {
      setRun(await api.getRun(runID));
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load run');
    }
  };

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

  useEffect(() => {
    loadRun();
    const interval = setInterval(loadRun, 3000);
    return () => clearInterval(interval);
  }, [runID]);

  if (error) return <div className="msg-error">{error}</div>;
  if (!run) return <div className="loading-state">Loading run...</div>;

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
