import { useState, useEffect } from 'react';
import { useParams, Link } from 'react-router-dom';
import { api } from '../api';
import type { RunDetail as RunDetailType } from '../api';
import StepTable from '../components/StepTable';
import WorkflowGraph from '../components/WorkflowGraph';
import GanttChart from '../components/GanttChart';

function badgeClass(status: string): string {
  const map: Record<string, string> = {
    pending: 'badge-pending',
    running: 'badge-running',
    completed: 'badge-completed',
    failed: 'badge-failed',
  };
  return `badge ${map[status] || 'badge-pending'}`;
}

type ViewMode = 'graph' | 'gantt' | 'table';

export default function RunDetail() {
  const { runID } = useParams<{ runID: string }>();
  const [run, setRun] = useState<RunDetailType | null>(null);
  const [error, setError] = useState('');
  const [view, setView] = useState<ViewMode>('graph');

  const loadRun = async () => {
    if (!runID) return;
    try {
      setRun(await api.getRun(runID));
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load run');
    }
  };

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

      {view === 'graph' && <WorkflowGraph steps={run.steps} />}
      {view === 'gantt' && <GanttChart steps={run.steps} />}
      {view === 'table' && <StepTable steps={run.steps} />}
    </div>
  );
}
