import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { api } from '../api';
import type { Schedule, ScheduleOccurrence } from '../api';

function displayTime(value: string | null, zone: string): string {
  if (!value) return '—';
  return new Date(value).toLocaleString(undefined, { timeZone: zone, timeZoneName: 'short' });
}

export default function Schedules() {
  const [items, setItems] = useState<Schedule[]>([]);
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState<number | null>(null);
  const [expanded, setExpanded] = useState<number | null>(null);
  const [history, setHistory] = useState<ScheduleOccurrence[]>([]);

  useEffect(() => {
    let active = true;
    const poll = async () => {
      try {
        const rows = await api.listSchedules();
        if (active) { setItems(rows); setError(''); }
        if (expanded !== null) {
          const occurrences = await api.scheduleOccurrences(expanded);
          if (active) setHistory(occurrences);
        }
      } catch (err) { if (active) setError(String(err instanceof Error ? err.message : err)); }
      finally { if (active) setLoading(false); }
    };
    void poll();
    const timer = setInterval(poll, 5000);
    return () => { active = false; clearInterval(timer); };
  }, [expanded]);

  const change = async (item: Schedule, action: 'pause' | 'resume' | 'delete') => {
    if (action === 'delete' && !window.confirm(`Delete schedule "${item.name}"? This removes its history, but does not stop a run already launched.`)) return;
    setBusy(item.id);
    try {
      if (action === 'pause') await api.pauseSchedule(item.id);
      if (action === 'resume') await api.resumeSchedule(item.id);
      if (action === 'delete') { await api.deleteSchedule(item.id); if (expanded === item.id) setExpanded(null); }
      setItems(await api.listSchedules()); setError('');
    } catch (err) { setError(err instanceof Error ? err.message : 'Schedule update failed'); }
    finally { setBusy(null); }
  };

  return <div>
    <div className="page-header">
      <h1 className="page-title">Schedules</h1>
      <Link to="/trigger?timing=once" className="btn btn-primary">New Schedule</Link>
    </div>
    <p className="schedule-help">One-time and recurring launches use saved workflow definitions and inputs. Recurring runs wait for their previous run to finish. Pausing or deleting a schedule does not stop a run already launched.</p>
    {error && <div className="msg-error" role="alert">{error}</div>}
    {loading ? <p>Loading schedules...</p> : <div className="table-wrap"><table>
      <thead><tr><th>Name / Workflow</th><th>Timing</th><th>Next Launch</th><th>State / Last Run</th><th>Actions</th></tr></thead>
      <tbody>{items.map(item => <tr key={item.id}>
        <td><button className="btn btn-ghost btn-sm" onClick={() => { setHistory([]); setExpanded(expanded === item.id ? null : item.id); }}>{item.name}</button><div className="cell-mono">{item.request.workflow_name}</div></td>
        <td>{item.kind === 'cron' ? <code>{item.cron}</code> : displayTime(item.run_at, item.timezone)}<div className="schedule-help">{item.timezone}</div></td>
        <td>{item.state === 'active' ? displayTime(item.next_run_at, item.timezone) : '—'}</td>
        <td><span className={`badge badge-${item.state === 'active' || item.state === 'dispatching' ? 'running' : item.state}`}>{item.state}</span>{item.last_run_id && <div><Link to={`/runs/${item.last_run_id}`}>{item.last_run_id}</Link></div>}{item.last_error && <div className="msg-error">{item.last_error}</div>}</td>
        <td><div className="schedule-actions">
          {['active', 'paused'].includes(item.state) && <Link className="btn btn-ghost btn-sm" to={`/trigger?schedule=${item.id}`}>Edit</Link>}
          {item.state === 'active' && <button className="btn btn-ghost btn-sm" disabled={busy === item.id} onClick={() => void change(item, 'pause')}>Pause</button>}
          {item.state === 'paused' && <button className="btn btn-ghost btn-sm" disabled={busy === item.id} onClick={() => void change(item, 'resume')}>Resume</button>}
          <button className="btn btn-danger btn-sm" disabled={busy === item.id || item.state === 'dispatching'} onClick={() => void change(item, 'delete')}>Delete</button>
        </div></td>
      </tr>)}</tbody>
    </table>{items.length === 0 && <p>No schedules yet.</p>}</div>}
    {expanded !== null && items.filter(item => item.id === expanded).map(item => <section key={item.id} className="schedule-details">
      <h2>{item.name}</h2>
      <p>Saved source commit: <code>{item.source_commit || 'manual definition'}</code>. Editing timing or inputs retains this definition. Selecting a different workflow saves that workflow's current definition.</p>
      <details><summary>Saved launch inputs</summary><pre>{JSON.stringify(item.request, null, 2)}</pre></details>
      <h3>Recent launches</h3>
      {history.length === 0 ? <p>No launches recorded.</p> : <table><thead><tr><th>Scheduled For</th><th>Run</th><th>Dispatch</th></tr></thead><tbody>{history.map(o => <tr key={o.id}><td>{displayTime(o.scheduled_for, item.timezone)}</td><td><Link to={`/runs/${o.run_id}`}>{o.run_id}</Link></td><td>{o.state}{o.error && <div className="msg-error">{o.error}</div>}</td></tr>)}</tbody></table>}
    </section>)}
  </div>;
}
