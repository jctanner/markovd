import { useState, useEffect } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { api } from '../api';
import type { Run } from '../api';
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

function duration(start: string | null, end: string | null): string {
  if (!start) return '-';
  const s = new Date(start).getTime();
  const e = end ? new Date(end).getTime() : Date.now();
  const ms = e - s;
  if (ms < 1000) return `${ms}ms`;
  return `${(ms / 1000).toFixed(1)}s`;
}

export default function Runs() {
  const navigate = useNavigate();
  const [runs, setRuns] = useState<Run[]>([]);
  const [error, setError] = useState('');
  const [rerunTarget, setRerunTarget] = useState<Run | null>(null);

  const loadRuns = async () => {
    try {
      setRuns(await api.listRuns());
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load runs');
    }
  };

  const handleCancel = async (runID: string) => {
    try {
      await api.cancelRun(runID);
      await loadRuns();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to cancel run');
    }
  };

  const handleDelete = async (runID: string) => {
    if (!window.confirm(`Delete run ${runID}? This cannot be undone.`)) return;
    try {
      await api.deleteRun(runID);
      await loadRuns();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete run');
    }
  };

  const handleRerunConfirm = async (
    vars: Record<string, string>,
    volumes: { name: string; pvc: string; mount_path: string }[],
    secretVolumes: { name: string; secret: string; mount_path: string }[],
  ) => {
    if (!rerunTarget) return;
    try {
      const newRun = await api.createRun(rerunTarget.workflow_name, vars, false, volumes, secretVolumes);
      setRerunTarget(null);
      navigate(`/runs/${newRun.run_id}`);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to re-run');
    }
  };

  useEffect(() => {
    loadRuns();
    const interval = setInterval(loadRuns, 5000);
    return () => clearInterval(interval);
  }, []);

  return (
    <div>
      <div className="page-header">
        <h1 className="page-title">Runs</h1>
        <Link to="/trigger" className="btn btn-primary btn-sm">New Run</Link>
      </div>

      {error && <div className="msg-error">{error}</div>}

      <div className="table-wrap">
        <table>
          <thead>
            <tr>
              <th>Run ID</th>
              <th>Workflow</th>
              <th>Status</th>
              <th>Started</th>
              <th>Duration</th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            {runs.map((run) => (
              <tr key={run.run_id}>
                <td className="cell-link">
                  <Link to={`/runs/${run.run_id}`}>{run.run_id}</Link>
                </td>
                <td>{run.workflow_name}</td>
                <td>
                  <span className={badgeClass(run.status)}>{run.status}</span>
                </td>
                <td className="cell-mono">
                  {run.started_at ? new Date(run.started_at).toLocaleString() : '-'}
                </td>
                <td className="cell-mono">
                  {duration(run.started_at, run.completed_at)}
                </td>
                <td>
                  {(run.status === 'running' || run.status === 'pending') && (
                    <button className="btn btn-sm btn-danger" onClick={() => handleCancel(run.run_id)}>
                      Stop
                    </button>
                  )}
                  {' '}
                  <button className="btn btn-sm btn-ghost" onClick={() => handleDelete(run.run_id)}>
                    Delete
                  </button>
                  {' '}
                  <button className="btn btn-sm btn-ghost" onClick={() => setRerunTarget(run)}>
                    Re-run
                  </button>
                </td>
              </tr>
            ))}
            {runs.length === 0 && (
              <tr>
                <td colSpan={6} className="table-empty">
                  No runs yet. <Link to="/trigger">Trigger one</Link>.
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>

      <RerunModal
        run={rerunTarget}
        onClose={() => setRerunTarget(null)}
        onConfirm={handleRerunConfirm}
      />
    </div>
  );
}
