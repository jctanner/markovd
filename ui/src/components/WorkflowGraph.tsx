import { useMemo, useCallback, useState, useEffect, useRef } from 'react';
import {
  ReactFlow,
  Background,
  BackgroundVariant,
  Controls,
  MiniMap,
  Handle,
  Position,
  MarkerType,
  useReactFlow,
} from '@xyflow/react';
import type { Node, Edge, NodeProps } from '@xyflow/react';
import '@xyflow/react/dist/style.css';
import type { Step } from '../api';

const NODE_H = 72;
const SUMMARY_NODE_H = 96;
const NODE_GAP_Y = 60;
const FORK_GAP_X = 280;
const START_Y = 40;
const START_X = 0;
const COLLAPSE_THRESHOLD = 5;

type StepNodeData = {
  label: string;
  stepType: string;
  status: string;
  duration: string;
  error: string;
  forkId: string;
  workflowName: string;
  outputJson: string;
};

type ForkSummaryData = {
  stepName: string;
  forkPrefix: string;
  totalBranches: number;
  completed: number;
  running: number;
  failed: number;
  skipped: number;
  pending: number;
};

const statusBorder: Record<string, string> = {
  pending: 'var(--status-pending)',
  running: 'var(--status-running)',
  completed: 'var(--status-completed)',
  failed: 'var(--status-failed)',
  skipped: 'var(--status-skipped)',
};

const typeIcons: Record<string, string> = {
  shell_exec: 'terminal',
  llm_invoke: 'brain',
  http_request: 'globe',
  gate: 'lock',
  human_gate: 'lock',
  agent_skill: 'cpu',
  transform: 'shuffle',
  validate: 'check',
  notify: 'bell',
  jira_api: 'globe',
  checkpoint: 'save',
};

function duration(start: string | null, end: string | null): string {
  if (!start) return '';
  const s = new Date(start).getTime();
  const e = end ? new Date(end).getTime() : Date.now();
  const ms = e - s;
  if (ms < 1000) return `${ms}ms`;
  return `${(ms / 1000).toFixed(1)}s`;
}

function durationLong(start: string | null, end: string | null): string {
  if (!start) return '-';
  const s = new Date(start).getTime();
  const e = end ? new Date(end).getTime() : Date.now();
  const ms = e - s;
  if (ms < 1000) return `${ms}ms`;
  if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
  return `${(ms / 60000).toFixed(1)}m`;
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

function StepNode({ data }: NodeProps<Node<StepNodeData>>) {
  const border = statusBorder[data.status] || statusBorder.pending;
  const icon = typeIcons[data.stepType] || 'circle';
  const isSub = data.forkId !== '';
  const jobName = parseJobName(data.outputJson);

  return (
    <div
      className={`graph-node graph-node-${data.status}${isSub ? ' graph-node-sub' : ''}`}
      style={{ borderLeftColor: border, cursor: 'pointer' }}
    >
      <Handle type="target" position={Position.Top} className="graph-handle" />
      <div className="graph-node-top">
        <span className="graph-node-icon" title={data.stepType || 'step'}>
          {iconSvg(icon)}
        </span>
        <span className="graph-node-name">{data.label}</span>
      </div>
      {isSub && (
        <div className="graph-node-fork">{data.forkId}</div>
      )}
      <div className="graph-node-bottom">
        <span className={`graph-node-status graph-node-status-${data.status}`}>
          <span className="graph-node-dot" />
          {data.status}
        </span>
        {data.duration && <span className="graph-node-dur">{data.duration}</span>}
      </div>
      {jobName && <div className="graph-node-job">{jobName}</div>}
      {data.error && <div className="graph-node-error" title={data.error}>{data.error}</div>}
      <Handle type="source" position={Position.Bottom} className="graph-handle" />
    </div>
  );
}

function ForkSummaryNode({ data }: NodeProps<Node<ForkSummaryData>>) {
  const total = data.totalBranches;
  const pctComplete = total > 0 ? (data.completed / total) * 100 : 0;
  const pctRunning = total > 0 ? (data.running / total) * 100 : 0;
  const pctFailed = total > 0 ? (data.failed / total) * 100 : 0;
  const pctSkipped = total > 0 ? (data.skipped / total) * 100 : 0;

  const overallStatus = data.running > 0 ? 'running' : data.failed > 0 ? 'failed' : data.completed === total ? 'completed' : 'pending';
  const border = statusBorder[overallStatus] || statusBorder.pending;

  return (
    <div
      className={`graph-node graph-summary-node graph-node-${overallStatus}`}
      style={{ borderLeftColor: border, cursor: 'pointer' }}
    >
      <Handle type="target" position={Position.Top} className="graph-handle" />
      <div className="graph-summary-title">{data.stepName}</div>
      <div className="graph-summary-count">{total} branches</div>
      <div className="graph-summary-bar">
        {pctComplete > 0 && <div className="graph-summary-seg graph-summary-seg-completed" style={{ width: `${pctComplete}%` }} />}
        {pctRunning > 0 && <div className="graph-summary-seg graph-summary-seg-running" style={{ width: `${pctRunning}%` }} />}
        {pctFailed > 0 && <div className="graph-summary-seg graph-summary-seg-failed" style={{ width: `${pctFailed}%` }} />}
        {pctSkipped > 0 && <div className="graph-summary-seg graph-summary-seg-skipped" style={{ width: `${pctSkipped}%` }} />}
      </div>
      <div className="graph-summary-legend">
        {data.completed > 0 && <span className="graph-summary-legend-item graph-node-status-completed">{data.completed}</span>}
        {data.running > 0 && <span className="graph-summary-legend-item graph-node-status-running">{data.running}</span>}
        {data.failed > 0 && <span className="graph-summary-legend-item graph-node-status-failed">{data.failed}</span>}
        {data.skipped > 0 && <span className="graph-summary-legend-item graph-node-status-skipped">{data.skipped}</span>}
      </div>
      <Handle type="source" position={Position.Bottom} className="graph-handle" />
    </div>
  );
}

function iconSvg(name: string) {
  const props = { width: 14, height: 14, viewBox: '0 0 24 24', fill: 'none', stroke: 'currentColor', strokeWidth: 2, strokeLinecap: 'round' as const, strokeLinejoin: 'round' as const };
  switch (name) {
    case 'terminal':
      return <svg {...props}><polyline points="4 17 10 11 4 5" /><line x1="12" y1="19" x2="20" y2="19" /></svg>;
    case 'brain':
      return <svg {...props}><circle cx="12" cy="12" r="3" /><path d="M12 2a7 7 0 0 1 7 7c0 2.4-1.2 4.5-3 5.7V17a2 2 0 0 1-2 2h-4a2 2 0 0 1-2-2v-2.3C6.2 13.5 5 11.4 5 9a7 7 0 0 1 7-7z" /><line x1="10" y1="22" x2="14" y2="22" /></svg>;
    case 'globe':
      return <svg {...props}><circle cx="12" cy="12" r="10" /><line x1="2" y1="12" x2="22" y2="12" /><path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z" /></svg>;
    case 'lock':
      return <svg {...props}><rect x="3" y="11" width="18" height="11" rx="2" ry="2" /><path d="M7 11V7a5 5 0 0 1 10 0v4" /></svg>;
    case 'cpu':
      return <svg {...props}><rect x="4" y="4" width="16" height="16" rx="2" /><rect x="9" y="9" width="6" height="6" /><line x1="9" y1="1" x2="9" y2="4" /><line x1="15" y1="1" x2="15" y2="4" /><line x1="9" y1="20" x2="9" y2="23" /><line x1="15" y1="20" x2="15" y2="23" /><line x1="20" y1="9" x2="23" y2="9" /><line x1="20" y1="14" x2="23" y2="14" /><line x1="1" y1="9" x2="4" y2="9" /><line x1="1" y1="14" x2="4" y2="14" /></svg>;
    case 'shuffle':
      return <svg {...props}><polyline points="16 3 21 3 21 8" /><line x1="4" y1="20" x2="21" y2="3" /><polyline points="21 16 21 21 16 21" /><line x1="15" y1="15" x2="21" y2="21" /><line x1="4" y1="4" x2="9" y2="9" /></svg>;
    case 'check':
      return <svg {...props}><path d="M22 11.08V12a10 10 0 1 1-5.93-9.14" /><polyline points="22 4 12 14.01 9 11.01" /></svg>;
    case 'bell':
      return <svg {...props}><path d="M18 8A6 6 0 0 0 6 8c0 7-3 9-3 9h18s-3-2-3-9" /><path d="M13.73 21a2 2 0 0 1-3.46 0" /></svg>;
    case 'save':
      return <svg {...props}><path d="M19 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11l5 5v11a2 2 0 0 1-2 2z" /><polyline points="17 21 17 13 7 13 7 21" /><polyline points="7 3 7 8 15 8" /></svg>;
    case 'expand':
      return <svg {...props}><polyline points="15 3 21 3 21 9" /><polyline points="9 21 3 21 3 15" /><line x1="21" y1="3" x2="14" y2="10" /><line x1="3" y1="21" x2="10" y2="14" /></svg>;
    case 'shrink':
      return <svg {...props}><polyline points="4 14 10 14 10 20" /><polyline points="20 10 14 10 14 4" /><line x1="14" y1="10" x2="21" y2="3" /><line x1="3" y1="21" x2="10" y2="14" /></svg>;
    case 'arrow-down':
      return <svg {...props}><line x1="12" y1="5" x2="12" y2="19" /><polyline points="19 12 12 19 5 12" /></svg>;
    default:
      return <svg {...props}><circle cx="12" cy="12" r="4" /></svg>;
  }
}

function miniMapColor(node: Node): string {
  const d = node.data as Record<string, unknown>;
  const status = (d.status as string) || '';
  const map: Record<string, string> = {
    running: 'var(--status-running)',
    completed: 'var(--status-completed)',
    failed: 'var(--status-failed)',
    skipped: 'var(--status-skipped)',
  };
  return map[status] || 'var(--border-hover)';
}

function JumpToBottomButton({ nodes }: { nodes: Node[] }) {
  const { setCenter } = useReactFlow();
  const jump = useCallback(() => {
    if (nodes.length === 0) return;
    let bottom = nodes[0];
    for (const n of nodes) {
      if (n.position.y > bottom.position.y) bottom = n;
    }
    setCenter(bottom.position.x + 130, bottom.position.y + 36, { zoom: 1, duration: 400 });
  }, [nodes, setCenter]);

  return (
    <button className="graph-jump-btn" onClick={jump} title="Jump to bottom">
      {iconSvg('arrow-down')}
    </button>
  );
}

const nodeTypes = { step: StepNode, forkSummary: ForkSummaryNode };

// ─── Branch status helpers ─────────────────────────────────────────

interface BranchInfo {
  key: string;
  forkId: string;
  status: string;
  currentStep: string;
  stepsCompleted: number;
  stepsTotal: number;
  startedAt: string | null;
  completedAt: string | null;
}

function deriveBranchStatus(branchSteps: Step[]): string {
  if (branchSteps.some(s => s.status === 'running')) return 'running';
  if (branchSteps.some(s => s.status === 'failed')) return 'failed';
  if (branchSteps.length > 0 && branchSteps.every(s => s.status === 'completed' || s.status === 'skipped')) return 'completed';
  return 'pending';
}

function buildBranchInfos(steps: Step[], forkPrefix: string, forkIds: string[]): BranchInfo[] {
  const allSteps = steps.filter(s => {
    const fid = s.fork_id || '';
    return forkIds.some(fi => fid === fi || fid.startsWith(fi + '-'));
  });

  const byBranch = new Map<string, Step[]>();
  for (const fid of forkIds) {
    byBranch.set(fid, []);
  }
  for (const step of allSteps) {
    const fid = step.fork_id || '';
    const branch = forkIds.find(fi => fid === fi || fid.startsWith(fi + '-'));
    if (branch) byBranch.get(branch)!.push(step);
  }

  return forkIds.map(fid => {
    const bSteps = byBranch.get(fid) || [];
    const status = deriveBranchStatus(bSteps);
    const running = bSteps.find(s => s.status === 'running');
    const failed = bSteps.find(s => s.status === 'failed');
    const currentStep = running?.step_name || failed?.step_name || bSteps[bSteps.length - 1]?.step_name || '';
    const completed = bSteps.filter(s => s.status === 'completed' || s.status === 'skipped').length;
    const starts = bSteps.map(s => s.started_at).filter(Boolean) as string[];
    const ends = bSteps.map(s => s.completed_at).filter(Boolean) as string[];
    return {
      key: fid.substring(forkPrefix.length + 1),
      forkId: fid,
      status,
      currentStep,
      stepsCompleted: completed,
      stepsTotal: bSteps.length,
      startedAt: starts.length > 0 ? starts.sort()[0] : null,
      completedAt: status === 'completed' && ends.length > 0 ? ends.sort().reverse()[0] : null,
    };
  });
}

// ─── Fork detail modal ─────────────────────────────────────────────

interface ForkModalProps {
  info: { stepName: string; forkPrefix: string; forkIds: string[] } | null;
  steps: Step[];
  onClose: () => void;
  onStepClick?: (step: Step) => void;
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

function ForkDetailModal({ info, steps, onClose, onStepClick }: ForkModalProps) {
  const [expanded, setExpanded] = useState<string | null>(null);
  const [sortKey, setSortKey] = useState<'key' | 'status'>('key');

  if (!info) return null;

  const branches = buildBranchInfos(steps, info.forkPrefix, info.forkIds);
  const sorted = [...branches].sort((a, b) => {
    if (sortKey === 'status') {
      const order: Record<string, number> = { running: 0, failed: 1, pending: 2, completed: 3 };
      return (order[a.status] ?? 4) - (order[b.status] ?? 4);
    }
    return a.key.localeCompare(b.key);
  });

  const statusCounts = { completed: 0, running: 0, failed: 0, skipped: 0, pending: 0 };
  for (const b of branches) {
    if (b.status in statusCounts) statusCounts[b.status as keyof typeof statusCounts]++;
  }

  const expandedSteps = expanded
    ? steps.filter(s => {
        const fid = s.fork_id || '';
        return fid === expanded || fid.startsWith(expanded + '-');
      })
    : [];

  return (
    <div className="modal-backdrop" onClick={onClose}>
      <div className="modal-card fork-modal-card" onClick={(e) => e.stopPropagation()}>
        <div className="modal-header">
          <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
            <strong>{info.stepName}</strong>
            <span className="fork-modal-branch-count">{branches.length} branches</span>
            {statusCounts.completed > 0 && <span className="badge badge-completed">{statusCounts.completed}</span>}
            {statusCounts.running > 0 && <span className="badge badge-running">{statusCounts.running}</span>}
            {statusCounts.failed > 0 && <span className="badge badge-failed">{statusCounts.failed}</span>}
          </div>
          <button className="modal-close" onClick={onClose}>&times;</button>
        </div>
        <div className="modal-body">
          <table className="fork-modal-table">
            <thead>
              <tr>
                <th className="fork-modal-th" onClick={() => setSortKey('key')} style={{ cursor: 'pointer' }}>
                  Branch {sortKey === 'key' ? '▴' : ''}
                </th>
                <th className="fork-modal-th" onClick={() => setSortKey('status')} style={{ cursor: 'pointer' }}>
                  Status {sortKey === 'status' ? '▴' : ''}
                </th>
                <th className="fork-modal-th">Progress</th>
                <th className="fork-modal-th">Current Step</th>
                <th className="fork-modal-th">Duration</th>
              </tr>
            </thead>
            <tbody>
              {sorted.map(b => (
                <>
                  <tr
                    key={b.forkId}
                    className={`fork-modal-row fork-modal-row-${b.status}${expanded === b.forkId ? ' fork-modal-row-expanded' : ''}`}
                    onClick={() => setExpanded(expanded === b.forkId ? null : b.forkId)}
                  >
                    <td className="fork-modal-td mono">{b.key}</td>
                    <td className="fork-modal-td"><span className={badgeClass(b.status)}>{b.status}</span></td>
                    <td className="fork-modal-td mono">{b.stepsCompleted}/{b.stepsTotal}</td>
                    <td className="fork-modal-td mono">{b.currentStep}</td>
                    <td className="fork-modal-td mono">{durationLong(b.startedAt, b.completedAt)}</td>
                  </tr>
                  {expanded === b.forkId && expandedSteps.length > 0 && (
                    <tr key={b.forkId + '-detail'} className="fork-modal-detail-row">
                      <td colSpan={5} className="fork-modal-detail-cell">
                        <table className="fork-modal-detail-table">
                          <tbody>
                            {expandedSteps.map((s, idx) => (
                              <tr
                                key={idx}
                                className="fork-modal-step-row"
                                onClick={(e) => { e.stopPropagation(); onStepClick?.(s); }}
                              >
                                <td className="fork-modal-td mono">{s.step_name}</td>
                                <td className="fork-modal-td"><span className={badgeClass(s.status)}>{s.status}</span></td>
                                <td className="fork-modal-td mono">{s.step_type || '-'}</td>
                                <td className="fork-modal-td mono">{durationLong(s.started_at, s.completed_at)}</td>
                              </tr>
                            ))}
                          </tbody>
                        </table>
                      </td>
                    </tr>
                  )}
                </>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}

// ─── Graph builder ─────────────────────────────────────────────────

interface CollapsedForkMeta {
  stepName: string;
  forkPrefix: string;
  forkIds: string[];
}

function buildGraph(
  steps: Step[],
): { nodes: Node[]; edges: Edge[]; stepMap: Map<string, Step>; collapsedForks: Map<string, CollapsedForkMeta> } {
  if (steps.length === 0) return { nodes: [], edges: [], stepMap: new Map(), collapsedForks: new Map() };

  const stepMap = new Map<string, Step>();
  const groups = new Map<string, Step[]>();
  for (const step of steps) {
    const fid = step.fork_id || '';
    if (!groups.has(fid)) groups.set(fid, []);
    groups.get(fid)!.push(step);
  }

  const nodes: Node[] = [];
  const edges: Edge[] = [];
  const collapsedForks = new Map<string, CollapsedForkMeta>();

  interface ChainEntry {
    step: Step;
    path: string;
    forkPrefix: string;
  }

  function expandChain(path: string): ChainEntry[] {
    const stepsAtPath = groups.get(path) || [];
    const chain: ChainEntry[] = [];
    for (const step of stepsAtPath) {
      const forkPrefix = path ? `${path}-${step.step_name}` : step.step_name;
      chain.push({ step, path, forkPrefix });
      if (groups.has(forkPrefix)) {
        chain.push(...expandChain(forkPrefix));
      }
    }
    return chain;
  }

  function findForEachForks(forkPrefix: string, chainPrefixes: string[]): string[] {
    const prefix = forkPrefix + '-';
    const candidates: string[] = [];
    for (const fid of groups.keys()) {
      if (!fid.startsWith(prefix)) continue;
      const ownedByChild = chainPrefixes.some(cp => cp.length > forkPrefix.length && fid.startsWith(cp + '-'));
      if (!ownedByChild) candidates.push(fid);
    }
    return candidates.filter(fid =>
      !candidates.some(other => other !== fid && fid.startsWith(other + '-'))
    );
  }

  function forkDisplayKey(forkId: string, forkPrefix: string): string {
    return forkId.substring(forkPrefix.length + 1);
  }

  function aggregateBranchStatuses(forkIds: string[]): { completed: number; running: number; failed: number; skipped: number; pending: number } {
    const counts = { completed: 0, running: 0, failed: 0, skipped: 0, pending: 0 };
    for (const fid of forkIds) {
      const allSteps: Step[] = [];
      for (const [gid, gsteps] of groups) {
        if (gid === fid || gid.startsWith(fid + '-')) allSteps.push(...gsteps);
      }
      const status = deriveBranchStatus(allSteps);
      if (status in counts) counts[status as keyof typeof counts]++;
    }
    return counts;
  }

  function layoutChain(
    chain: ChainEntry[],
    x: number,
    startY: number,
    displayForkId: string,
  ): { nodeIds: string[]; endY: number } {
    const chainNodeIds: string[] = [];
    const chainSteps: Step[] = [];
    const chainForkIndices = new Map<number, string[]>();
    let y = startY;

    const chainPrefixes = chain.map(e => e.forkPrefix);
    for (let i = 0; i < chain.length; i++) {
      const forks = findForEachForks(chain[i].forkPrefix, chainPrefixes);
      if (forks.length > 0) chainForkIndices.set(i, forks);
    }

    for (let i = 0; i < chain.length; i++) {
      const { step, path, forkPrefix } = chain[i];
      const nodeId = `${displayForkId || 'main'}::${path}::${step.step_name}`;
      stepMap.set(nodeId, step);
      nodes.push({
        id: nodeId,
        type: 'step',
        position: { x, y },
        data: {
          label: step.step_name,
          stepType: step.step_type || '',
          status: step.status,
          duration: duration(step.started_at, step.completed_at),
          error: step.error || '',
          forkId: displayForkId,
          workflowName: step.workflow_name,
          outputJson: step.output_json || '',
        },
        draggable: false,
      });
      chainNodeIds.push(nodeId);
      chainSteps.push(step);
      y += NODE_H + NODE_GAP_Y;

      const forkIds = chainForkIndices.get(i);
      if (forkIds && forkIds.length > 0) {
        if (forkIds.length > COLLAPSE_THRESHOLD) {
          // Collapsed summary node
          const counts = aggregateBranchStatuses(forkIds);
          const summaryId = `summary::${forkPrefix}`;
          const overallStatus = counts.running > 0 ? 'running' : counts.failed > 0 ? 'failed' : counts.completed === forkIds.length ? 'completed' : 'pending';

          collapsedForks.set(summaryId, { stepName: step.step_name, forkPrefix, forkIds });

          nodes.push({
            id: summaryId,
            type: 'forkSummary',
            position: { x, y },
            data: {
              stepName: step.step_name,
              forkPrefix,
              totalBranches: forkIds.length,
              ...counts,
            },
            draggable: false,
          });

          edges.push({
            id: `${nodeId}-fork->${summaryId}`,
            source: nodeId,
            target: summaryId,
            type: 'smoothstep',
            animated: overallStatus === 'running',
            markerEnd: { type: MarkerType.ArrowClosed, width: 14, height: 14 },
            style: { stroke: 'var(--accent)', strokeWidth: 1.5, strokeDasharray: '6 3' },
          });

          y += SUMMARY_NODE_H + NODE_GAP_Y;

          if (i + 1 < chain.length) {
            const nextNodeId = `${displayForkId || 'main'}::${chain[i + 1].path}::${chain[i + 1].step.step_name}`;
            edges.push({
              id: `${summaryId}-join->${nextNodeId}`,
              source: summaryId,
              target: nextNodeId,
              type: 'smoothstep',
              markerEnd: { type: MarkerType.ArrowClosed, width: 14, height: 14 },
              style: { stroke: 'var(--accent)', strokeWidth: 1.5, strokeDasharray: '6 3' },
            });
          }
        } else {
          // Expanded fork columns
          const forkStartY = y;
          const totalWidth = (forkIds.length - 1) * FORK_GAP_X;
          const forkBaseX = x - totalWidth / 2;
          let maxForkEndY = y;

          for (let fi = 0; fi < forkIds.length; fi++) {
            const forkId = forkIds[fi];
            const forkX = forkBaseX + fi * FORK_GAP_X;
            const forkKey = forkDisplayKey(forkId, forkPrefix);
            const forkChain = expandChain(forkId);

            const { nodeIds: fNodeIds, endY: fEndY } = layoutChain(
              forkChain, forkX, forkStartY, forkKey,
            );

            if (fNodeIds.length > 0) {
              edges.push({
                id: `${nodeId}-fork->${fNodeIds[0]}`,
                source: nodeId,
                target: fNodeIds[0],
                type: 'smoothstep',
                animated: forkChain[0]?.step.status === 'running',
                markerEnd: { type: MarkerType.ArrowClosed, width: 14, height: 14 },
                style: { stroke: 'var(--accent)', strokeWidth: 1.5, strokeDasharray: '6 3' },
              });

              if (i + 1 < chain.length) {
                const nextNodeId = `${displayForkId || 'main'}::${chain[i + 1].path}::${chain[i + 1].step.step_name}`;
                edges.push({
                  id: `${fNodeIds[fNodeIds.length - 1]}-join->${nextNodeId}`,
                  source: fNodeIds[fNodeIds.length - 1],
                  target: nextNodeId,
                  type: 'smoothstep',
                  markerEnd: { type: MarkerType.ArrowClosed, width: 14, height: 14 },
                  style: { stroke: 'var(--accent)', strokeWidth: 1.5, strokeDasharray: '6 3' },
                });
              }
            }

            maxForkEndY = Math.max(maxForkEndY, fEndY);
          }

          y = maxForkEndY;
        }
      }
    }

    for (let i = 0; i < chainNodeIds.length - 1; i++) {
      if (chainForkIndices.has(i)) continue;
      const isRunning = chainSteps[i + 1].status === 'running';
      edges.push({
        id: `${chainNodeIds[i]}->${chainNodeIds[i + 1]}`,
        source: chainNodeIds[i],
        target: chainNodeIds[i + 1],
        type: 'smoothstep',
        animated: isRunning,
        markerEnd: { type: MarkerType.ArrowClosed, width: 16, height: 16 },
        style: { stroke: isRunning ? 'var(--status-running)' : 'var(--border-hover)', strokeWidth: 2 },
      });
    }

    return { nodeIds: chainNodeIds, endY: y };
  }

  const mainChain = expandChain('');
  layoutChain(mainChain, START_X, START_Y, '');

  return { nodes, edges, stepMap, collapsedForks };
}

// ─── Color mode hook ───────────────────────────────────────────────

function useColorMode(): 'dark' | 'light' {
  const [mode, setMode] = useState<'dark' | 'light'>(
    () => (document.documentElement.dataset.theme as 'dark' | 'light') || 'dark'
  );
  useEffect(() => {
    const observer = new MutationObserver(() => {
      setMode((document.documentElement.dataset.theme as 'dark' | 'light') || 'dark');
    });
    observer.observe(document.documentElement, { attributes: true, attributeFilter: ['data-theme'] });
    return () => observer.disconnect();
  }, []);
  return mode;
}

// ─── Main component ────────────────────────────────────────────────

interface Props {
  steps: Step[];
  onStepClick?: (step: Step) => void;
}

export default function WorkflowGraph({ steps, onStepClick }: Props) {
  const { nodes, edges, stepMap, collapsedForks } = useMemo(() => buildGraph(steps), [steps]);
  const colorMode = useColorMode();
  const [fullscreen, setFullscreen] = useState(false);
  const [activeFork, setActiveFork] = useState<CollapsedForkMeta | null>(null);
  const containerRef = useRef<HTMLDivElement>(null);

  const maxY = nodes.reduce((m, n) => Math.max(m, n.position.y), 0);
  const graphHeight = Math.max(300, maxY + NODE_H + 80);

  const onInit = useCallback((instance: { fitView: () => void }) => {
    setTimeout(() => instance.fitView(), 50);
  }, []);

  const handleNodeClick = useCallback((_: React.MouseEvent, node: Node) => {
    if (node.type === 'forkSummary') {
      const meta = collapsedForks.get(node.id);
      if (meta) setActiveFork(meta);
      return;
    }
    if (!onStepClick) return;
    const step = stepMap.get(node.id);
    if (step) onStepClick(step);
  }, [onStepClick, stepMap, collapsedForks]);

  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && fullscreen) setFullscreen(false);
    };
    window.addEventListener('keydown', handler);
    return () => window.removeEventListener('keydown', handler);
  }, [fullscreen]);

  if (steps.length === 0) {
    return <div className="graph-empty">Waiting for steps...</div>;
  }

  return (
    <>
      <div
        ref={containerRef}
        className={`graph-container${fullscreen ? ' graph-fullscreen' : ''}`}
        style={fullscreen ? undefined : { height: Math.min(graphHeight, 800) }}
      >
        <button
          className="graph-fullscreen-btn"
          onClick={() => setFullscreen(!fullscreen)}
          title={fullscreen ? 'Exit fullscreen' : 'Fullscreen'}
        >
          {iconSvg(fullscreen ? 'shrink' : 'expand')}
        </button>
        <ReactFlow
          nodes={nodes}
          edges={edges}
          nodeTypes={nodeTypes}
          onInit={onInit}
          onNodeClick={handleNodeClick}
          fitView
          fitViewOptions={{ padding: 0.3 }}
          nodesDraggable={false}
          nodesConnectable={false}
          elementsSelectable={false}
          panOnDrag
          zoomOnScroll
          minZoom={0.1}
          maxZoom={1.5}
          colorMode={colorMode}
        >
          <Background variant={BackgroundVariant.Dots} gap={24} size={1} />
          <Controls showInteractive={false} />
          <MiniMap
            nodeColor={miniMapColor}
            maskColor="rgba(0, 0, 0, 0.35)"
            pannable
            zoomable
          />
          <JumpToBottomButton nodes={nodes} />
        </ReactFlow>
      </div>
      <ForkDetailModal
        info={activeFork}
        steps={steps}
        onClose={() => setActiveFork(null)}
        onStepClick={onStepClick}
      />
    </>
  );
}
