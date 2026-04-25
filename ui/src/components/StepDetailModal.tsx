import { useState, useEffect, useRef } from 'react';
import { api } from '../api';
import type { Step } from '../api';

interface Props {
  step: Step | null;
  onClose: () => void;
}

function parseOutputJson(s: string): Record<string, unknown> | null {
  if (!s) return null;
  try {
    const parsed = JSON.parse(s);
    if (typeof parsed === 'object' && parsed !== null) return parsed as Record<string, unknown>;
    return null;
  } catch {
    return null;
  }
}

function duration(start: string | null, end: string | null): string {
  if (!start) return '-';
  const s = new Date(start).getTime();
  const e = end ? new Date(end).getTime() : Date.now();
  const ms = e - s;
  if (ms < 1000) return `${ms}ms`;
  if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
  return `${(ms / 60000).toFixed(1)}m`;
}

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

function LogsSection({ step, jobName }: { step: Step; jobName: string }) {
  const [logs, setLogs] = useState('');
  const [loading, setLoading] = useState(true);
  const [streaming, setStreaming] = useState(false);
  const [cached, setCached] = useState(false);
  const logsRef = useRef<HTMLPreElement>(null);

  useEffect(() => {
    setLogs('');
    setLoading(true);
    setCached(false);
    setStreaming(false);

    const parsed = parseOutputJson(step.output_json);
    const cachedLogs = parsed?.logs as string | undefined;

    if (step.status === 'running') {
      const controller = new AbortController();
      const token = localStorage.getItem('token');

      fetch(`/api/v1/jobs/${encodeURIComponent(jobName)}/logs/stream`, {
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
      }).catch((err) => {
        if (err.name === 'AbortError') return;
        if (cachedLogs) {
          setLogs(cachedLogs);
          setCached(true);
        } else {
          setLogs('Failed to stream logs');
        }
        setLoading(false);
        setStreaming(false);
      });

      return () => controller.abort();
    }

    api.getJobLogs(jobName).then((res) => {
      if (res.logs) {
        setLogs(res.logs);
        setCached(res.cached === 'true');
      } else if (cachedLogs) {
        setLogs(cachedLogs);
        setCached(true);
      } else {
        setLogs(res.error || 'No logs available');
      }
    }).catch(() => {
      if (cachedLogs) {
        setLogs(cachedLogs);
        setCached(true);
      } else {
        setLogs('Failed to fetch logs');
      }
    }).finally(() => setLoading(false));
  }, [step, jobName]);

  useEffect(() => {
    if (streaming && logsRef.current) {
      logsRef.current.scrollTop = logsRef.current.scrollHeight;
    }
  }, [logs, streaming]);

  return (
    <div className="step-detail-section">
      <div className="step-detail-section-header">
        <span className="step-detail-section-label">Logs</span>
        {streaming && <span className="modal-live-badge">Live</span>}
        {cached && <span className="badge badge-pending">cached</span>}
      </div>
      {loading ? (
        <div className="modal-loading">Loading logs...</div>
      ) : (
        <pre className="modal-logs" ref={logsRef}>{logs}</pre>
      )}
    </div>
  );
}

export default function StepDetailModal({ step, onClose }: Props) {
  if (!step) return null;

  const output = parseOutputJson(step.output_json);
  const jobName = output?.job_name as string | undefined;
  const hasError = !!step.error;
  const errorLabel = step.status === 'skipped' ? 'Skip Reason' : 'Error';
  const errorClass = step.status === 'skipped' ? 'step-detail-error-skipped' : 'step-detail-error-failed';

  const outputEntries = output
    ? Object.entries(output).filter(([k]) => k !== 'logs')
    : [];

  return (
    <div className="modal-backdrop" onClick={onClose}>
      <div className="modal-card" onClick={(e) => e.stopPropagation()}>
        <div className="modal-header">
          <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
            <strong>{step.step_name}</strong>
            <span className={badgeClass(step.status)}>{step.status}</span>
          </div>
          <button className="modal-close" onClick={onClose}>&times;</button>
        </div>
        <div className="modal-body">
          <div className="step-detail-meta">
            <div className="step-detail-field">
              <div className="step-detail-label">Workflow</div>
              <div className="step-detail-value">{step.workflow_name}</div>
            </div>
            <div className="step-detail-field">
              <div className="step-detail-label">Type</div>
              <div className="step-detail-value mono">{step.step_type || '-'}</div>
            </div>
            {step.fork_id && (
              <div className="step-detail-field">
                <div className="step-detail-label">Fork</div>
                <div className="step-detail-value mono">{step.fork_id}</div>
              </div>
            )}
            <div className="step-detail-field">
              <div className="step-detail-label">Duration</div>
              <div className="step-detail-value mono">{duration(step.started_at, step.completed_at)}</div>
            </div>
            <div className="step-detail-field">
              <div className="step-detail-label">Started</div>
              <div className="step-detail-value mono">
                {step.started_at ? new Date(step.started_at).toLocaleString() : '-'}
              </div>
            </div>
            <div className="step-detail-field">
              <div className="step-detail-label">Completed</div>
              <div className="step-detail-value mono">
                {step.completed_at ? new Date(step.completed_at).toLocaleString() : '-'}
              </div>
            </div>
          </div>

          {hasError && (
            <div className="step-detail-section">
              <div className="step-detail-section-label">{errorLabel}</div>
              <pre className={`step-detail-error ${errorClass}`}>{step.error}</pre>
            </div>
          )}

          {outputEntries.length > 0 && (
            <div className="step-detail-section">
              <div className="step-detail-section-label">Output</div>
              <div className="step-detail-output">
                {outputEntries.map(([k, v]) => (
                  <div key={k} className="step-detail-output-row">
                    <span className="step-detail-output-key">{k}</span>
                    <span className="step-detail-output-val">
                      {typeof v === 'string' ? v : JSON.stringify(v)}
                    </span>
                  </div>
                ))}
              </div>
            </div>
          )}

          {jobName && <LogsSection step={step} jobName={jobName} />}
        </div>
      </div>
    </div>
  );
}
