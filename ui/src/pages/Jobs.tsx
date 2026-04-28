import { useState, useEffect, useRef } from 'react';
import { Link } from 'react-router-dom';
import { api } from '../api';
import type { ActiveJob, ConcurrencyBucket } from '../api';

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

function duration(start: string | null): string {
  if (!start) return '-';
  const ms = Date.now() - new Date(start).getTime();
  if (ms < 1000) return `${ms}ms`;
  if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
  return `${(ms / 60000).toFixed(1)}m`;
}

function jobKey(job: ActiveJob): string {
  return `${job.kind}-${job.run_id}-${job.fork_id}-${job.workflow_name}-${job.step_name}`;
}

function parseSseLines(buffer: string): { lines: string[]; remainder: string } {
  const lines: string[] = [];
  let remainder = buffer;
  let idx: number;
  while ((idx = remainder.indexOf('\n\n')) !== -1) {
    const frame = remainder.slice(0, idx);
    remainder = remainder.slice(idx + 2);
    for (const line of frame.split('\n')) {
      if (line.startsWith('data: ')) {
        lines.push(line.slice(6));
      }
    }
  }
  return { lines, remainder };
}

function formatTime(iso: string): string {
  const d = new Date(iso);
  return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
}

function ConcurrencyChart({ buckets }: { buckets: ConcurrencyBucket[] }) {
  if (buckets.length === 0) return null;

  const max = Math.max(...buckets.map((b) => b.count), 1);
  const tickCount = Math.min(max, 5);
  const ticks = Array.from({ length: tickCount + 1 }, (_, i) => Math.round((max / tickCount) * i));

  const labelInterval = Math.max(1, Math.floor(buckets.length / 8));

  return (
    <div className="concurrency-chart">
      <div className="concurrency-chart-header">
        <span className="concurrency-chart-title">Concurrency (24h)</span>
        <span className="concurrency-chart-subtitle">15-min buckets</span>
      </div>
      <div className="concurrency-chart-body">
        <div className="concurrency-chart-yaxis">
          {[...ticks].reverse().map((t) => (
            <span key={t} className="concurrency-chart-ylabel">{t}</span>
          ))}
        </div>
        <div className="concurrency-chart-area">
          <div className="concurrency-chart-grid">
            {ticks.map((t) => (
              <div key={t} className="concurrency-chart-gridline" style={{ bottom: `${(t / max) * 100}%` }} />
            ))}
          </div>
          <div className="concurrency-chart-bars">
            {buckets.map((b, i) => (
              <div key={b.t} className="concurrency-chart-col" title={`${formatTime(b.t)}: ${b.count} concurrent`}>
                <div
                  className={`concurrency-chart-bar${b.count > 0 ? ' active' : ''}`}
                  style={{ height: `${(b.count / max) * 100}%` }}
                />
                {i % labelInterval === 0 && (
                  <span className="concurrency-chart-xlabel">{formatTime(b.t)}</span>
                )}
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}

function JobLogModal({ job, onClose }: { job: ActiveJob; onClose: () => void }) {
  const [logs, setLogs] = useState('');
  const [loading, setLoading] = useState(true);
  const [streaming, setStreaming] = useState(false);
  const logsRef = useRef<HTMLPreElement>(null);

  useEffect(() => {
    setLogs('');
    setLoading(true);
    setStreaming(false);

    if (!job.job_name) {
      setLogs('No job name available');
      setLoading(false);
      return;
    }

    if (job.status === 'running' || job.status === 'pending') {
      const controller = new AbortController();
      const token = localStorage.getItem('token');

      const connect = () => {
        fetch(`/api/v1/jobs/${encodeURIComponent(job.job_name)}/logs/stream`, {
          headers: { 'Authorization': `Bearer ${token}` },
          signal: controller.signal,
        }).then(async (res) => {
          if (!res.body) {
            setLogs('Streaming not supported');
            setLoading(false);
            return;
          }
          setLoading(false);
          setStreaming(true);

          const reader = res.body.getReader();
          const decoder = new TextDecoder();
          let buffer = '';
          let accumulated = '';

          while (true) {
            const { done, value } = await reader.read();
            if (done) break;
            buffer += decoder.decode(value, { stream: true });
            const { lines, remainder } = parseSseLines(buffer);
            buffer = remainder;
            if (lines.length > 0) {
              accumulated += lines.join('\n') + '\n';
              setLogs(accumulated);
            }
          }
          setStreaming(false);
          if (!controller.signal.aborted) {
            setTimeout(connect, 3000);
          }
        }).catch((err) => {
          if (err.name === 'AbortError') return;
          setLoading(false);
          setStreaming(false);
          if (!controller.signal.aborted) {
            setTimeout(connect, 3000);
          }
        });
      };

      connect();
      return () => controller.abort();
    }

    api.getJobLogs(job.job_name).then((res) => {
      setLogs(res.logs || res.error || 'No logs available');
    }).catch(() => {
      setLogs('Failed to fetch logs');
    }).finally(() => setLoading(false));
  }, [job]);

  useEffect(() => {
    if (streaming && logsRef.current) {
      logsRef.current.scrollTop = logsRef.current.scrollHeight;
    }
  }, [logs, streaming]);

  return (
    <div className="modal-backdrop" onClick={onClose}>
      <div className="modal-card" onClick={(e) => e.stopPropagation()}>
        <div className="modal-header">
          <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
            <strong>{job.kind === 'run' ? job.run_id : job.step_name}</strong>
            <span className={badgeClass(job.status)}>{job.status}</span>
            {streaming && <span className="modal-live-badge">Live</span>}
          </div>
          <button className="modal-close" onClick={onClose}>&times;</button>
        </div>
        <div className="modal-body">
          <div className="step-detail-meta">
            <div className="step-detail-field">
              <div className="step-detail-label">Type</div>
              <div className="step-detail-value">{job.kind === 'run' ? 'Run' : 'Step'}</div>
            </div>
            <div className="step-detail-field">
              <div className="step-detail-label">Workflow</div>
              <div className="step-detail-value">{job.workflow_name}</div>
            </div>
            {job.kind === 'step' && (
              <div className="step-detail-field">
                <div className="step-detail-label">Run</div>
                <div className="step-detail-value mono">{job.run_id}</div>
              </div>
            )}
            <div className="step-detail-field">
              <div className="step-detail-label">K8s Job</div>
              <div className="step-detail-value mono">{job.job_name || '-'}</div>
            </div>
            <div className="step-detail-field">
              <div className="step-detail-label">Duration</div>
              <div className="step-detail-value mono">{duration(job.started_at)}</div>
            </div>
          </div>
          {loading ? (
            <div className="modal-loading">Loading logs...</div>
          ) : (
            <pre className="modal-logs" ref={logsRef}>{logs}</pre>
          )}
        </div>
      </div>
    </div>
  );
}

export default function Jobs() {
  const [jobs, setJobs] = useState<ActiveJob[]>([]);
  const [buckets, setBuckets] = useState<ConcurrencyBucket[]>([]);
  const [error, setError] = useState('');
  const [selectedJob, setSelectedJob] = useState<ActiveJob | null>(null);

  const loadJobs = () => {
    api.listActiveJobs().then(setJobs).catch(() => setError('Failed to load jobs'));
  };

  useEffect(() => {
    loadJobs();
    api.getConcurrencyHistory().then(setBuckets).catch(() => {});
    const interval = setInterval(loadJobs, 5000);
    const chartInterval = setInterval(() => {
      api.getConcurrencyHistory().then(setBuckets).catch(() => {});
    }, 30000);
    return () => { clearInterval(interval); clearInterval(chartInterval); };
  }, []);

  const handleCancel = (e: React.MouseEvent, job: ActiveJob) => {
    e.stopPropagation();
    const label = job.kind === 'run' ? job.run_id : `${job.step_name} (${job.run_id})`;
    if (!window.confirm(`Cancel ${label}?`)) return;
    api.cancelJob({
      kind: job.kind,
      run_id: job.run_id,
      fork_id: job.fork_id,
      workflow_name: job.workflow_name,
      step_name: job.step_name,
    }).then(loadJobs).catch(() => setError('Failed to cancel job'));
  };

  const handleDelete = (e: React.MouseEvent, job: ActiveJob) => {
    e.stopPropagation();
    if (!window.confirm(`Delete run ${job.run_id} and all its data?`)) return;
    api.deleteRun(job.run_id).then(loadJobs).catch(() => setError('Failed to delete run'));
  };

  return (
    <>
      <div className="page-header">
        <h1 className="page-title">Active Jobs</h1>
      </div>
      {error && <div className="msg-error">{error}</div>}
      <ConcurrencyChart buckets={buckets} />
      <div className="table-wrap">
        <table>
          <thead>
            <tr>
              <th>Type</th>
              <th>Identifier</th>
              <th>Workflow</th>
              <th>Status</th>
              <th>Duration</th>
              <th style={{ width: 160 }}>Actions</th>
            </tr>
          </thead>
          <tbody>
            {jobs.map((job) => (
              <tr key={jobKey(job)} className="step-row-clickable" onClick={() => setSelectedJob(job)}>
                <td>{job.kind === 'run' ? 'Run' : 'Step'}</td>
                <td className="cell-mono">
                  {job.kind === 'run' ? (
                    <Link to={`/runs/${job.run_id}`} className="cell-link" onClick={(e) => e.stopPropagation()}>
                      {job.run_id}
                    </Link>
                  ) : (
                    <div>
                      <div>{job.step_name}</div>
                      <div style={{ fontSize: 11, color: 'var(--text-muted)' }}>{job.run_id}</div>
                    </div>
                  )}
                </td>
                <td>{job.workflow_name}</td>
                <td><span className={badgeClass(job.status)}>{job.status}</span></td>
                <td className="cell-mono">{duration(job.started_at)}</td>
                <td>
                  <button className="btn btn-sm btn-danger" onClick={(e) => handleCancel(e, job)}>Stop</button>
                  {job.kind === 'run' && (
                    <button className="btn btn-sm btn-ghost" style={{ marginLeft: 4 }} onClick={(e) => handleDelete(e, job)}>Delete</button>
                  )}
                </td>
              </tr>
            ))}
            {jobs.length === 0 && (
              <tr><td colSpan={6} className="table-empty">No active jobs.</td></tr>
            )}
          </tbody>
        </table>
      </div>
      {selectedJob && <JobLogModal job={selectedJob} onClose={() => setSelectedJob(null)} />}
    </>
  );
}
