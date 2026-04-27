import { useMemo, useRef, useState, useEffect, useCallback } from 'react';
import type { Step } from '../api';

const ROW_H = 26;
const ROW_GAP = 2;
const LABEL_W = 260;
const MIN_BAR_W = 3;
const PADDING_TOP = 32;
const PADDING_BOTTOM = 20;
const TICK_HEIGHT = 20;
const COLLAPSE_THRESHOLD = 5;

const statusColor: Record<string, string> = {
  completed: 'var(--status-completed)',
  running: 'var(--status-running)',
  failed: 'var(--status-failed)',
  skipped: 'var(--status-skipped)',
  pending: 'var(--status-pending)',
};

interface GanttRow {
  type: 'step';
  label: string;
  forkId: string;
  workflowName: string;
  status: string;
  stepType: string;
  startMs: number;
  endMs: number;
  step: Step;
  forkGroup?: string;
}

interface GanttSummaryRow {
  type: 'summary';
  forkPrefix: string;
  stepName: string;
  branchCount: number;
  completed: number;
  running: number;
  failed: number;
  skipped: number;
  pending: number;
  startMs: number;
  endMs: number;
}

type DisplayRow = GanttRow | GanttSummaryRow;

function detectForkGroups(steps: Step[]): Map<string, string[]> {
  const forkIds = new Set<string>();
  for (const s of steps) {
    const fid = s.fork_id || '';
    if (fid) forkIds.add(fid);
  }

  const mainStepNames = new Set<string>();
  for (const s of steps) {
    if (!s.fork_id || s.fork_id === '') mainStepNames.add(s.step_name);
  }

  const groups = new Map<string, Set<string>>();
  for (const fid of forkIds) {
    for (const name of mainStepNames) {
      const prefix = name + '-';
      if (fid.startsWith(prefix)) {
        const remainder = fid.substring(prefix.length);
        const nextDash = remainder.indexOf('-');
        if (!groups.has(name)) groups.set(name, new Set());

        const rootFid = nextDash === -1 ? fid : findBranchRoot(fid, name);
        groups.get(name)!.add(rootFid);
        break;
      }
    }
  }

  const result = new Map<string, string[]>();
  for (const [name, branches] of groups) {
    if (branches.size > COLLAPSE_THRESHOLD) {
      result.set(name, [...branches]);
    }
  }
  return result;
}

function findBranchRoot(forkId: string, stepName: string): string {
  const allFids = new Set<string>();
  let current = forkId;
  while (current.length > stepName.length) {
    allFids.add(current);
    const lastDash = current.lastIndexOf('-');
    if (lastDash <= stepName.length) break;
    current = current.substring(0, lastDash);
  }
  if (current.length > stepName.length) allFids.add(current);

  let shortest = forkId;
  for (const fid of allFids) {
    if (fid.length < shortest.length) shortest = fid;
  }
  return shortest;
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
      type: 'step' as const,
      label,
      forkId: fid,
      workflowName: s.workflow_name,
      status: s.status,
      stepType: s.step_type || '',
      startMs: start - t0,
      endMs: Math.max(end - t0, start - t0 + 1),
      step: s,
    };
  });
}

function buildDisplayRows(
  allRows: GanttRow[],
  forkGroups: Map<string, string[]>,
  expanded: Set<string>,
  _t0Offset: number,
): DisplayRow[] {
  if (forkGroups.size === 0) return allRows;

  const forkMembership = new Map<string, string>();
  for (const [stepName, branchRoots] of forkGroups) {
    for (const root of branchRoots) {
      forkMembership.set(root, stepName);
    }
  }

  function belongsToGroup(forkId: string): string | null {
    if (forkMembership.has(forkId)) return forkMembership.get(forkId)!;
    for (const [root, stepName] of forkMembership) {
      if (forkId.startsWith(root + '-')) return stepName;
    }
    return null;
  }

  const display: DisplayRow[] = [];
  const insertedSummaries = new Set<string>();

  for (const row of allRows) {
    const group = belongsToGroup(row.forkId);
    if (!group) {
      display.push(row);
      continue;
    }

    if (expanded.has(group)) {
      display.push({ ...row, forkGroup: group });
      continue;
    }

    if (!insertedSummaries.has(group)) {
      insertedSummaries.add(group);
      const branchRoots = forkGroups.get(group)!;
      const groupRows = allRows.filter(r => belongsToGroup(r.forkId) === group);
      const counts = { completed: 0, running: 0, failed: 0, skipped: 0, pending: 0 };
      const branchStatuses = new Map<string, string[]>();
      for (const root of branchRoots) branchStatuses.set(root, []);
      for (const r of groupRows) {
        const root = branchRoots.find(br => r.forkId === br || r.forkId.startsWith(br + '-'));
        if (root) branchStatuses.get(root)!.push(r.status);
      }
      for (const [, statuses] of branchStatuses) {
        if (statuses.some(s => s === 'running')) counts.running++;
        else if (statuses.some(s => s === 'failed')) counts.failed++;
        else if (statuses.length > 0 && statuses.every(s => s === 'completed' || s === 'skipped')) counts.completed++;
        else if (statuses.some(s => s === 'skipped')) counts.skipped++;
        else counts.pending++;
      }
      const starts = groupRows.map(r => r.startMs).filter(v => v > 0);
      const ends = groupRows.map(r => r.endMs);
      display.push({
        type: 'summary',
        forkPrefix: group,
        stepName: group,
        branchCount: branchRoots.length,
        ...counts,
        startMs: starts.length > 0 ? Math.min(...starts) : 0,
        endMs: ends.length > 0 ? Math.max(...ends) : 0,
      });
    }
  }

  return display;
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

function parseJobName(outputJson: string): string | null {
  if (!outputJson) return null;
  try {
    const parsed = JSON.parse(outputJson);
    return parsed.job_name || null;
  } catch {
    return null;
  }
}

interface Props {
  steps: Step[];
  onStepClick?: (step: Step) => void;
}

export default function GanttChart({ steps, onStepClick }: Props) {
  const containerRef = useRef<HTMLDivElement>(null);
  const scrollRef = useRef<HTMLDivElement>(null);
  const [containerW, setContainerW] = useState(800);
  const [tooltip, setTooltip] = useState<{ x: number; y: number; row: GanttRow } | null>(null);
  const [expanded, setExpanded] = useState<Set<string>>(new Set());

  useEffect(() => {
    if (!containerRef.current) return;
    const obs = new ResizeObserver(entries => {
      for (const e of entries) setContainerW(e.contentRect.width);
    });
    obs.observe(containerRef.current);
    return () => obs.disconnect();
  }, []);

  const { allRows, durationMs, forkGroups } = useMemo(() => {
    const withTime = steps.filter(s => s.started_at);
    if (withTime.length === 0) return { allRows: [], durationMs: 0, forkGroups: new Map<string, string[]>() };

    const t0 = Math.min(...withTime.map(s => new Date(s.started_at!).getTime()));
    const hasRunning = steps.some(s => s.status === 'running');
    const stepEnds = steps.map(s => {
      if (s.completed_at) return new Date(s.completed_at).getTime();
      if (s.started_at) return new Date(s.started_at).getTime();
      return t0;
    });
    const tEnd = Math.max(...stepEnds, ...(hasRunning ? [Date.now()] : []));
    const durationMs = Math.max(tEnd - t0, 1);
    const allRows = buildRows(steps, t0, tEnd);
    const forkGroups = detectForkGroups(steps);
    return { allRows, durationMs, forkGroups };
  }, [steps]);

  const displayRows = useMemo(
    () => buildDisplayRows(allRows, forkGroups, expanded, 0),
    [allRows, forkGroups, expanded],
  );

  const toggleGroup = useCallback((group: string) => {
    setExpanded(prev => {
      const next = new Set(prev);
      if (next.has(group)) next.delete(group);
      else next.add(group);
      return next;
    });
  }, []);

  const jumpToBottom = useCallback(() => {
    if (!scrollRef.current) return;
    scrollRef.current.scrollTo({ top: scrollRef.current.scrollHeight, behavior: 'smooth' });
  }, []);

  const jumpToRunning = useCallback(() => {
    if (!scrollRef.current) return;
    const idx = displayRows.findIndex(r =>
      (r.type === 'step' && r.status === 'running') ||
      (r.type === 'summary' && r.running > 0)
    );
    if (idx === -1) return;
    const offset = PADDING_TOP + idx * (ROW_H + ROW_GAP);
    const viewH = scrollRef.current.clientHeight;
    scrollRef.current.scrollTo({ top: Math.max(0, offset - viewH / 2), behavior: 'smooth' });
  }, [displayRows]);

  if (displayRows.length === 0) {
    return <div className="graph-empty">No timing data yet.</div>;
  }

  const chartW = Math.max(containerW - LABEL_W - 24, 200);
  const chartH = displayRows.length * (ROW_H + ROW_GAP) + PADDING_TOP + PADDING_BOTTOM;
  const ticks = getTicks(durationMs, chartW);
  const hasRunning = displayRows.some(r =>
    (r.type === 'step' && r.status === 'running') ||
    (r.type === 'summary' && r.running > 0)
  );

  let prevForkId = '';

  return (
    <div className="gantt-wrap" ref={containerRef} style={{ position: 'relative' }}>
      <div className="gantt-nav-buttons">
        {hasRunning && (
          <button className="gantt-jump-btn" onClick={jumpToRunning} title="Jump to running">
            <svg width={14} height={14} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2} strokeLinecap="round" strokeLinejoin="round">
              <circle cx="12" cy="12" r="10" /><polygon points="10 8 16 12 10 16 10 8" />
            </svg>
          </button>
        )}
        <button className="gantt-jump-btn" onClick={jumpToBottom} title="Jump to bottom">
          <svg width={14} height={14} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2} strokeLinecap="round" strokeLinejoin="round">
            <line x1="12" y1="5" x2="12" y2="19" /><polyline points="19 12 12 19 5 12" />
          </svg>
        </button>
      </div>

      <div className="gantt-scroll" ref={scrollRef} style={{ overflowX: 'auto', overflowY: 'auto', maxHeight: 700 }}>
        <div className="gantt-inner" style={{ display: 'flex', minWidth: LABEL_W + chartW + 24 }}>
          {/* Labels column */}
          <div className="gantt-labels" style={{ width: LABEL_W, flexShrink: 0, paddingTop: PADDING_TOP }}>
            {displayRows.map((row, i) => {
              if (row.type === 'summary') {
                const isExpanded = expanded.has(row.forkPrefix);
                const overallStatus = row.running > 0 ? 'running' : row.failed > 0 ? 'failed' : row.completed === row.branchCount ? 'completed' : 'pending';
                return (
                  <div
                    key={`summary-${row.forkPrefix}`}
                    className="gantt-label gantt-label-boundary gantt-label-summary"
                    style={{ height: ROW_H, marginBottom: ROW_GAP, cursor: 'pointer' }}
                    onClick={() => toggleGroup(row.forkPrefix)}
                    title={`${row.branchCount} branches — click to ${isExpanded ? 'collapse' : 'expand'}`}
                  >
                    <span className="gantt-summary-chevron">{isExpanded ? '▾' : '▸'}</span>
                    <span className={`gantt-label-text gantt-label-text-summary gantt-label-status-${overallStatus}`}>
                      {row.stepName}
                    </span>
                    <span className="gantt-summary-badge">{row.branchCount}</span>
                  </div>
                );
              }
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
            {displayRows.map((row, i) => {
              const y = PADDING_TOP + i * (ROW_H + ROW_GAP);

              if (row.type === 'summary') {
                const total = row.branchCount;
                const barX = (row.startMs / durationMs) * chartW;
                const barW = Math.max(((row.endMs - row.startMs) / durationMs) * chartW, MIN_BAR_W);
                const segments: { color: string; pct: number; status: string }[] = [];
                if (row.completed > 0) segments.push({ color: statusColor.completed, pct: row.completed / total, status: 'completed' });
                if (row.running > 0) segments.push({ color: statusColor.running, pct: row.running / total, status: 'running' });
                if (row.failed > 0) segments.push({ color: statusColor.failed, pct: row.failed / total, status: 'failed' });
                if (row.skipped > 0) segments.push({ color: statusColor.skipped, pct: row.skipped / total, status: 'skipped' });
                if (row.pending > 0) segments.push({ color: statusColor.pending, pct: row.pending / total, status: 'pending' });

                let segX = barX;
                return (
                  <g
                    key={`summary-bar-${row.forkPrefix}`}
                    style={{ cursor: 'pointer' }}
                    onClick={() => toggleGroup(row.forkPrefix)}
                  >
                    <rect x={barX} y={y + 1} width={barW} height={ROW_H - 2} rx={3} fill="var(--bg-root)" opacity={0.5} />
                    {segments.map((seg, si) => {
                      const segW = seg.pct * barW;
                      const el = (
                        <rect
                          key={si}
                          x={segX}
                          y={y + 1}
                          width={segW}
                          height={ROW_H - 2}
                          rx={si === 0 ? 3 : 0}
                          fill={seg.color}
                          opacity={0.85}
                        />
                      );
                      segX += segW;
                      return el;
                    })}
                    {row.running > 0 && (
                      <rect x={barX} y={y + 1} width={barW} height={ROW_H - 2} rx={3} fill={statusColor.running} className="gantt-bar-pulse" />
                    )}
                  </g>
                );
              }

              const x = (row.startMs / durationMs) * chartW;
              const w = Math.max(((row.endMs - row.startMs) / durationMs) * chartW, MIN_BAR_W);
              const color = statusColor[row.status] || statusColor.pending;
              return (
                <g
                  key={i}
                  style={{ cursor: 'pointer' }}
                  onClick={() => onStepClick?.(row.step)}
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
          {parseJobName(tooltip.row.step.output_json) && (
            <div className="gantt-tooltip-detail">
              job: {parseJobName(tooltip.row.step.output_json)}
            </div>
          )}
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
