import type { Step } from '../api';

function badgeClass(status: string): string {
  const map: Record<string, string> = {
    pending: 'badge-pending',
    running: 'badge-running',
    completed: 'badge-completed',
    failed: 'badge-failed',
    skipped: 'badge-skipped',
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

export default function StepTable({ steps }: { steps: Step[] }) {
  if (steps.length === 0) {
    return <div className="table-empty">No steps recorded yet.</div>;
  }

  const hasForks = steps.some(s => s.fork_id);

  return (
    <div className="table-wrap">
      <table>
        <thead>
          <tr>
            <th>Step</th>
            {hasForks && <th>Fork</th>}
            <th>Workflow</th>
            <th>Type</th>
            <th>Status</th>
            <th>Duration</th>
            <th>Error</th>
          </tr>
        </thead>
        <tbody>
          {steps.map((step) => (
            <tr key={`${step.fork_id || ''}-${step.workflow_name}-${step.step_name}`}>
              <td className="cell-mono">{step.step_name}</td>
              {hasForks && <td className="cell-mono cell-fork">{step.fork_id || '-'}</td>}
              <td>{step.workflow_name}</td>
              <td>{step.step_type || '-'}</td>
              <td>
                <span className={badgeClass(step.status)}>{step.status}</span>
              </td>
              <td className="cell-mono">{duration(step.started_at, step.completed_at)}</td>
              <td className="cell-error">{step.error || ''}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
