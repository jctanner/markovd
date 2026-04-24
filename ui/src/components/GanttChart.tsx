import { useMemo, useRef, useState, useEffect } from 'react';
import type { Step } from '../api';

const ROW_H = 26;
const ROW_GAP = 2;
const LABEL_W = 260;
const MIN_BAR_W = 3;
const PADDING_TOP = 32;
const PADDING_BOTTOM = 20;
const TICK_HEIGHT = 20;

const statusColor: Record<string, string> = {
  completed: 'var(--status-completed)',
  running: 'var(--status-running)',
  failed: 'var(--status-failed)',
  skipped: 'var(--status-skipped)',
  pending: 'var(--status-pending)',
};

interface GanttRow {
  label: string;
  forkId: string;
  workflowName: string;
  status: string;
  stepType: string;
  startMs: number;
  endMs: number;
}

function buildRows(steps: Step[], t0: number, tEnd: number): GanttRow[] {
  const forkOrder = new Map<string, number>();
  let forkIdx = 0;
  for (const s of steps) {
    const fid = s.fork_id || '';
    if (!forkOrder.has(fid)) forkOrder.set(fid, forkIdx++);
  }

  const sorted = [...steps].sort((a, b) => {
    const fa = forkOrder.get(a.fork_id || '') || 0;
    const fb = forkOrder.get(b.fork_id || '') || 0;
    if (fa !== fb) return fa - fb;
    const ta = a.started_at ? new Date(a.started_at).getTime() : tEnd;
    const tb = b.started_at ? new Date(b.started_at).getTime() : tEnd;
    return ta - tb;
  });

  return sorted.map(s => {
    const start = s.started_at ? new Date(s.started_at).getTime() : tEnd;
    const end = s.completed_at ? new Date(s.completed_at).getTime() : (s.status === 'running' ? Date.now() : start);
    const fid = s.fork_id || '';
    const prefix = fid ? fid.split('-').slice(-2).join('-') : '';
    const label = prefix ? `${prefix} / ${s.step_name}` : s.step_name;
    return {
      label,
      forkId: fid,
      workflowName: s.workflow_name,
      status: s.status,
      stepType: s.step_type || '',
      startMs: start - t0,
      endMs: Math.max(end - t0, start - t0 + 1),
    };
  });
}

function formatMs(ms: number): string {
  if (ms < 1000) return `${Math.round(ms)}ms`;
  if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
  return `${(ms / 60000).toFixed(1)}m`;
}

function getTicks(durationMs: number, chartW: number): number[] {
  const minTickSpacing = 60;
  const maxTicks = Math.floor(chartW / minTickSpacing);
  const intervals = [10, 20, 50, 100, 200, 500, 1000, 2000, 5000, 10000, 30000, 60000];
  let interval = intervals[intervals.length - 1];
  for (const iv of intervals) {
    if (durationMs / iv <= maxTicks) { interval = iv; break; }
  }
  const ticks: number[] = [];
  for (let t = 0; t <= durationMs; t += interval) {
    ticks.push(t);
  }
  return ticks;
}

export default function GanttChart({ steps }: { steps: Step[] }) {
  const containerRef = useRef<HTMLDivElement>(null);
  const [containerW, setContainerW] = useState(800);
  const [tooltip, setTooltip] = useState<{ x: number; y: number; row: GanttRow } | null>(null);

  useEffect(() => {
    if (!containerRef.current) return;
    const obs = new ResizeObserver(entries => {
      for (const e of entries) setContainerW(e.contentRect.width);
    });
    obs.observe(containerRef.current);
    return () => obs.disconnect();
  }, []);

  const { rows, durationMs } = useMemo(() => {
    const withTime = steps.filter(s => s.started_at);
    if (withTime.length === 0) return { rows: [], durationMs: 0, t0: 0 };

    const t0 = Math.min(...withTime.map(s => new Date(s.started_at!).getTime()));
    const hasRunning = steps.some(s => s.status === 'running');
    const stepEnds = steps.map(s => {
      if (s.completed_at) return new Date(s.completed_at).getTime();
      if (s.started_at) return new Date(s.started_at).getTime();
      return t0;
    });
    const tEnd = Math.max(...stepEnds, ...(hasRunning ? [Date.now()] : []));
    const durationMs = Math.max(tEnd - t0, 1);
    return { rows: buildRows(steps, t0, tEnd), durationMs, t0 };
  }, [steps]);

  if (rows.length === 0) {
    return <div className="graph-empty">No timing data yet.</div>;
  }

  const chartW = Math.max(containerW - LABEL_W - 24, 200);
  const chartH = rows.length * (ROW_H + ROW_GAP) + PADDING_TOP + PADDING_BOTTOM;
  const ticks = getTicks(durationMs, chartW);

  let prevForkId = '';

  return (
    <div className="gantt-wrap" ref={containerRef}>
      <div className="gantt-scroll" style={{ overflowX: 'auto', overflowY: 'auto', maxHeight: 700 }}>
        <div className="gantt-inner" style={{ display: 'flex', minWidth: LABEL_W + chartW + 24 }}>
          {/* Labels column */}
          <div className="gantt-labels" style={{ width: LABEL_W, flexShrink: 0, paddingTop: PADDING_TOP }}>
            {rows.map((row, i) => {
              const isForkBoundary = row.forkId !== prevForkId;
              prevForkId = row.forkId;
              return (
                <div
                  key={i}
                  className={`gantt-label${isForkBoundary ? ' gantt-label-boundary' : ''}`}
                  style={{ height: ROW_H, marginBottom: ROW_GAP }}
                  title={row.forkId ? `${row.forkId} / ${row.workflowName}` : row.workflowName}
                >
                  <span className="gantt-label-text">{row.label}</span>
                </div>
              );
            })}
          </div>

          {/* Chart area */}
          <svg
            className="gantt-svg"
            width={chartW}
            height={chartH}
            style={{ flexShrink: 0 }}
            onMouseLeave={() => setTooltip(null)}
          >
            {/* Tick lines + labels */}
            {ticks.map(t => {
              const x = (t / durationMs) * chartW;
              return (
                <g key={t}>
                  <line x1={x} y1={PADDING_TOP - TICK_HEIGHT} x2={x} y2={chartH} className="gantt-tick-line" />
                  <text x={x} y={PADDING_TOP - TICK_HEIGHT - 4} className="gantt-tick-text">{formatMs(t)}</text>
                </g>
              );
            })}

            {/* Bars */}
            {rows.map((row, i) => {
              const x = (row.startMs / durationMs) * chartW;
              const w = Math.max(((row.endMs - row.startMs) / durationMs) * chartW, MIN_BAR_W);
              const y = PADDING_TOP + i * (ROW_H + ROW_GAP);
              const color = statusColor[row.status] || statusColor.pending;
              return (
                <g
                  key={i}
                  onMouseEnter={e => setTooltip({ x: e.clientX, y: e.clientY, row })}
                  onMouseMove={e => setTooltip({ x: e.clientX, y: e.clientY, row })}
                  onMouseLeave={() => setTooltip(null)}
                >
                  <rect
                    x={x}
                    y={y + 2}
                    width={w}
                    height={ROW_H - 4}
                    rx={3}
                    fill={color}
                    opacity={row.status === 'skipped' ? 0.35 : 0.85}
                    className="gantt-bar"
                  />
                  {row.status === 'running' && (
                    <rect
                      x={x}
                      y={y + 2}
                      width={w}
                      height={ROW_H - 4}
                      rx={3}
                      fill={color}
                      className="gantt-bar-pulse"
                    />
                  )}
                </g>
              );
            })}
          </svg>
        </div>
      </div>

      {tooltip && (
        <div
          className="gantt-tooltip"
          style={{ left: tooltip.x + 12, top: tooltip.y - 10, position: 'fixed' }}
        >
          <div className="gantt-tooltip-name">{tooltip.row.label}</div>
          <div className="gantt-tooltip-detail">
            <span>{tooltip.row.workflowName}</span>
            {tooltip.row.stepType && <span> &middot; {tooltip.row.stepType}</span>}
          </div>
          <div className="gantt-tooltip-detail">
            <span className={`gantt-tooltip-status gantt-tooltip-status-${tooltip.row.status}`}>{tooltip.row.status}</span>
            <span> &middot; {formatMs(tooltip.row.endMs - tooltip.row.startMs)}</span>
          </div>
          <div className="gantt-tooltip-detail">
            offset {formatMs(tooltip.row.startMs)}
          </div>
        </div>
      )}
    </div>
  );
}
