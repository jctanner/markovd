import { useState, useEffect, useRef } from 'react';
import { api } from '../api';
import type { Step, StepProgressEvent } from '../api';
import { describeToolInput, isTerminalKind, mergeProgress, summarizeProgress } from './stepProgress.ts';

interface Props {
  step: Step | null;
  // The step's description from the workflow definition, when it has one.
  description?: string;
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

const PROGRESS_POLL_MS = 2000;
const PROGRESS_PAGE = 500;
const settledStatuses = new Set(['completed', 'failed', 'skipped']);

function TranscriptItem({ event }: { event: StepProgressEvent }) {
  const d = event.data;
  switch (event.kind) {
    case 'init':
      return (
        <div className="transcript-item transcript-meta">
          Session started{d.model ? ` · ${String(d.model)}` : ''}{d.session_id ? ` · ${String(d.session_id)}` : ''}
        </div>
      );
    case 'text':
      return <div className="transcript-item transcript-text">{String(d.text ?? '')}</div>;
    case 'tool_use':
      return (
        <div className="transcript-item transcript-tool">
          <span className="transcript-tool-name">{String(d.name ?? 'tool')}</span>
          <span className="transcript-tool-input">{describeToolInput(d.input)}</span>
        </div>
      );
    case 'tool_result':
      return (
        <pre className={`transcript-item transcript-result${d.is_error ? ' transcript-result-error' : ''}`}>
          {String(d.content ?? '') || '(no output)'}
        </pre>
      );
    case 'limit_exceeded':
      return (
        <div className="transcript-item transcript-limit">
          Limit exceeded: {String(d.limit)} (max {String(d.max)}{d.observed !== undefined ? `, reached ${String(d.observed)}` : ''})
        </div>
      );
    case 'result':
      return (
        <div className="transcript-item transcript-meta">
          Finished{d.is_error ? ' with an error' : ''}{d.stop_reason ? ` · ${String(d.stop_reason)}` : ''}
        </div>
      );
    default:
      return null;
  }
}

// Live in-step progress (for example a claude transcript). Renders nothing
// when the step emitted no progress events. Callers key it by step identity so
// switching steps remounts it with fresh state.
function TranscriptSection({ step }: { step: Step }) {
  const [events, setEvents] = useState<StepProgressEvent[]>([]);
  const [error, setError] = useState(false);
  const bodyRef = useRef<HTMLDivElement>(null);
  const stickRef = useRef(true);

  useEffect(() => {
    let cancelled = false;
    let timer: ReturnType<typeof setTimeout> | undefined;
    let lastId = 0;
    let seen: StepProgressEvent[] = [];

    const tick = async () => {
      let more = false;
      let failed = false;
      try {
        const res = await api.getStepProgress(step, lastId);
        if (cancelled) return;
        const incoming = res.events ?? [];
        if (incoming.length > 0) {
          lastId = incoming[incoming.length - 1].id;
          seen = mergeProgress(seen, incoming);
          setEvents(seen);
        }
        more = incoming.length >= PROGRESS_PAGE;
        setError(false);
      } catch {
        if (cancelled) return;
        failed = true;
        setError(true);
      }
      // The step prop is a snapshot, so a step that was running when opened is
      // followed until its stream reports a terminal event.
      const terminal = seen.some((e) => isTerminalKind(e.kind));
      const settled = settledStatuses.has(step.status);
      if (more) {
        timer = setTimeout(tick, 0);
      } else if (!terminal && !(settled && !failed)) {
        timer = setTimeout(tick, PROGRESS_POLL_MS);
      }
    };
    tick();

    return () => {
      cancelled = true;
      if (timer) clearTimeout(timer);
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [step.run_id, step.fork_id, step.workflow_name, step.step_name]);

  useEffect(() => {
    const el = bodyRef.current;
    if (el && stickRef.current) el.scrollTop = el.scrollHeight;
  }, [events]);

  if (events.length === 0) return error ? <div className="modal-loading">Failed to load progress</div> : null;

  const summary = summarizeProgress(events);
  return (
    <div className="step-detail-section">
      <div className="step-detail-section-header">
        <span className="step-detail-section-label">Transcript</span>
        {!summary.finished && step.status === 'running' && <span className="modal-live-badge">Live</span>}
        <span className="transcript-summary">
          {summary.turns} turn{summary.turns === 1 ? '' : 's'} · {summary.tokens.toLocaleString()} tokens
          {summary.costUSD !== null ? ` · $${summary.costUSD.toFixed(2)}` : ''}
        </span>
        {summary.limitExceeded && <span className="badge badge-failed">{summary.limitExceeded}</span>}
      </div>
      <div
        className="transcript"
        ref={bodyRef}
        onScroll={(e) => {
          const el = e.currentTarget;
          stickRef.current = el.scrollHeight - el.scrollTop - el.clientHeight < 24;
        }}
      >
        {events.map((e) => <TranscriptItem key={e.id} event={e} />)}
      </div>
    </div>
  );
}

export default function StepDetailModal({ step, description, onClose }: Props) {
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
          {description && <p className="step-detail-description">{description}</p>}
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

          <TranscriptSection
            key={`${step.run_id}/${step.fork_id}/${step.workflow_name}/${step.step_name}`}
            step={step}
          />

          {outputEntries.length > 0 && (
            <div className="step-detail-section">
              <div className="step-detail-section-label">Output</div>
              <div className="step-detail-output">
                {outputEntries.map(([k, v]) => (
                  <div key={k} className="step-detail-output-row">
                    <span className="step-detail-output-key">{k}</span>
                    <span className="step-detail-output-val">
                      {typeof v === 'string'
                        ? v
                        : k === 'events' && Array.isArray(v)
                          ? `${v.length} events (see Transcript)`
                          : JSON.stringify(v)}
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
