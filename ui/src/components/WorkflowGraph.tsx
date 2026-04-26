import { useMemo, useCallback, useState, useEffect } from 'react';
import {
  ReactFlow,
  Background,
  BackgroundVariant,
  Controls,
  Handle,
  Position,
  MarkerType,
} from '@xyflow/react';
import type { Node, Edge, NodeProps } from '@xyflow/react';
import '@xyflow/react/dist/style.css';
import type { Step } from '../api';

const NODE_H = 72;
const NODE_GAP_Y = 60;
const FORK_GAP_X = 280;
const START_Y = 40;
const START_X = 0;

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
    default:
      return <svg {...props}><circle cx="12" cy="12" r="4" /></svg>;
  }
}

const nodeTypes = { step: StepNode };

function buildGraph(steps: Step[]): { nodes: Node<StepNodeData>[]; edges: Edge[]; stepMap: Map<string, Step> } {
  if (steps.length === 0) return { nodes: [], edges: [], stepMap: new Map() };

  const stepMap = new Map<string, Step>();
  const groups = new Map<string, Step[]>();
  for (const step of steps) {
    const fid = step.fork_id || '';
    if (!groups.has(fid)) groups.set(fid, []);
    groups.get(fid)!.push(step);
  }

  const nodes: Node<StepNodeData>[] = [];
  const edges: Edge[] = [];

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

  return { nodes, edges, stepMap };
}

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

interface Props {
  steps: Step[];
  onStepClick?: (step: Step) => void;
}

export default function WorkflowGraph({ steps, onStepClick }: Props) {
  const { nodes, edges, stepMap } = useMemo(() => buildGraph(steps), [steps]);
  const colorMode = useColorMode();

  const maxY = nodes.reduce((m, n) => Math.max(m, n.position.y), 0);
  const graphHeight = Math.max(300, maxY + NODE_H + 80);

  const onInit = useCallback((instance: { fitView: () => void }) => {
    setTimeout(() => instance.fitView(), 50);
  }, []);

  const handleNodeClick = useCallback((_: React.MouseEvent, node: Node) => {
    if (!onStepClick) return;
    const step = stepMap.get(node.id);
    if (step) onStepClick(step);
  }, [onStepClick, stepMap]);

  if (steps.length === 0) {
    return <div className="graph-empty">Waiting for steps...</div>;
  }

  return (
    <div className="graph-container" style={{ height: Math.min(graphHeight, 800) }}>
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
        minZoom={0.2}
        maxZoom={1.5}
        colorMode={colorMode}
      >
        <Background variant={BackgroundVariant.Dots} gap={24} size={1} />
        <Controls showInteractive={false} />
      </ReactFlow>
    </div>
  );
}
