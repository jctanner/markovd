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

function StepNode({ data }: NodeProps<Node<StepNodeData>>) {
  const border = statusBorder[data.status] || statusBorder.pending;
  const icon = typeIcons[data.stepType] || 'circle';
  const isSub = data.forkId !== '';

  return (
    <div className={`graph-node graph-node-${data.status}${isSub ? ' graph-node-sub' : ''}`} style={{ borderLeftColor: border }}>
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

interface ForkGroup {
  forkId: string;
  steps: Step[];
}

function buildGraph(steps: Step[]): { nodes: Node<StepNodeData>[]; edges: Edge[] } {
  if (steps.length === 0) return { nodes: [], edges: [] };

  const groups = new Map<string, Step[]>();
  for (const step of steps) {
    const fid = step.fork_id || '';
    if (!groups.has(fid)) groups.set(fid, []);
    groups.get(fid)!.push(step);
  }

  const mainSteps = groups.get('') || [];
  groups.delete('');

  // Collect unique fork prefixes that represent direct sub-workflows of main
  // e.g. "deploy_all-0", "deploy_all-1" are direct forks; "deploy_all-0-health_check" is nested
  const forkGroups: ForkGroup[] = [];
  const sortedForks = Array.from(groups.keys()).sort();
  for (const fid of sortedForks) {
    forkGroups.push({ forkId: fid, steps: groups.get(fid)! });
  }

  const nodes: Node<StepNodeData>[] = [];
  const edges: Edge[] = [];

  // Track Y positions for layout
  let globalY = START_Y;

  // Build main chain
  const mainNodeIds: string[] = [];
  // Track which main steps have forks branching off
  const mainStepForks = new Map<string, ForkGroup[]>();

  for (const fg of forkGroups) {
    // Direct fork: fork_id like "deploy_all-0" has parent in main
    const fid = fg.forkId;
    const dashIdx = fid.indexOf('-');
    if (dashIdx === -1) continue;
    // Check if this is a top-level fork (no parent fork in our groups)
    const parentFork = fid.substring(0, fid.lastIndexOf('-'));
    const lastSeg = fid.substring(fid.lastIndexOf('-') + 1);
    if (/^\d+$/.test(lastSeg) && !groups.has(parentFork) && parentFork.indexOf('-') === -1) {
      // Top-level for_each fork: parent step name is the part before the index
      const stepName = parentFork;
      if (!mainStepForks.has(stepName)) mainStepForks.set(stepName, []);
      mainStepForks.get(stepName)!.push(fg);
    }
  }

  // Place main steps
  for (let i = 0; i < mainSteps.length; i++) {
    const step = mainSteps[i];
    const nodeId = `main::${step.step_name}`;
    nodes.push({
      id: nodeId,
      type: 'step',
      position: { x: START_X, y: globalY },
      data: {
        label: step.step_name,
        stepType: step.step_type || '',
        status: step.status,
        duration: duration(step.started_at, step.completed_at),
        error: step.error || '',
        forkId: '',
        workflowName: step.workflow_name,
      },
      draggable: false,
    });
    mainNodeIds.push(nodeId);
    globalY += NODE_H + NODE_GAP_Y;

    // If this step has forks, place them
    const forks = mainStepForks.get(step.step_name);
    if (forks && forks.length > 0) {
      const forkStartY = globalY;
      const totalWidth = (forks.length - 1) * FORK_GAP_X;
      const startX = START_X - totalWidth / 2;

      let maxForkEndY = globalY;

      for (let fi = 0; fi < forks.length; fi++) {
        const fg = forks[fi];
        const forkX = startX + fi * FORK_GAP_X;
        let forkY = forkStartY;

        // Collect nested steps for this fork: any fork_id that starts with this fork's id
        const nestedSteps: Step[] = [...fg.steps];
        const nestedForks: ForkGroup[] = [];
        for (const ofg of forkGroups) {
          if (ofg.forkId !== fg.forkId && ofg.forkId.startsWith(fg.forkId + '-')) {
            nestedForks.push(ofg);
          }
        }

        // Build nodes for this fork's direct steps
        const forkNodeIds: string[] = [];
        // Track which steps in this fork have nested sub-workflows
        const subForkMap = new Map<string, ForkGroup[]>();
        for (const nfg of nestedForks) {
          const suffix = nfg.forkId.substring(fg.forkId.length + 1);
          // Direct child: suffix has no more dashes, or suffix is "stepname" without index
          const subDash = suffix.indexOf('-');
          if (subDash === -1) {
            // Direct sub-workflow (no index), e.g. health_check
            if (!subForkMap.has(suffix)) subForkMap.set(suffix, []);
            subForkMap.get(suffix)!.push(nfg);
          }
        }

        for (let si = 0; si < nestedSteps.length; si++) {
          const step = nestedSteps[si];
          const nodeId = `${fg.forkId}::${step.step_name}`;
          nodes.push({
            id: nodeId,
            type: 'step',
            position: { x: forkX, y: forkY },
            data: {
              label: step.step_name,
              stepType: step.step_type || '',
              status: step.status,
              duration: duration(step.started_at, step.completed_at),
              error: step.error || '',
              forkId: fg.forkId,
              workflowName: step.workflow_name,
            },
            draggable: false,
          });
          forkNodeIds.push(nodeId);
          forkY += NODE_H + NODE_GAP_Y;
        }

        // Connect fork steps sequentially
        for (let si = 0; si < forkNodeIds.length - 1; si++) {
          const isRunning = nestedSteps[si + 1]?.status === 'running';
          edges.push({
            id: `${forkNodeIds[si]}->${forkNodeIds[si + 1]}`,
            source: forkNodeIds[si],
            target: forkNodeIds[si + 1],
            type: 'smoothstep',
            animated: isRunning,
            markerEnd: { type: MarkerType.ArrowClosed, width: 14, height: 14 },
            style: { stroke: isRunning ? 'var(--status-running)' : 'var(--border-hover)', strokeWidth: 1.5 },
          });
        }

        // Edge from parent main step to first fork node
        if (forkNodeIds.length > 0) {
          edges.push({
            id: `${nodeId}-fork->${forkNodeIds[0]}`,
            source: nodeId,
            target: forkNodeIds[0],
            type: 'smoothstep',
            animated: nestedSteps[0]?.status === 'running',
            markerEnd: { type: MarkerType.ArrowClosed, width: 14, height: 14 },
            style: { stroke: 'var(--accent)', strokeWidth: 1.5, strokeDasharray: '6 3' },
          });
        }

        // Edge from last fork node back to next main step
        if (forkNodeIds.length > 0 && i + 1 < mainSteps.length) {
          const nextMainId = `main::${mainSteps[i + 1].step_name}`;
          edges.push({
            id: `${forkNodeIds[forkNodeIds.length - 1]}-join->${nextMainId}`,
            source: forkNodeIds[forkNodeIds.length - 1],
            target: nextMainId,
            type: 'smoothstep',
            markerEnd: { type: MarkerType.ArrowClosed, width: 14, height: 14 },
            style: { stroke: 'var(--accent)', strokeWidth: 1.5, strokeDasharray: '6 3' },
          });
        }

        maxForkEndY = Math.max(maxForkEndY, forkY);
      }

      globalY = maxForkEndY;
    }
  }

  // Connect main steps sequentially (skip connections where forks already join)
  for (let i = 0; i < mainNodeIds.length - 1; i++) {
    const stepName = mainSteps[i].step_name;
    if (mainStepForks.has(stepName)) continue; // fork edges handle this connection

    const isRunning = mainSteps[i + 1].status === 'running';
    edges.push({
      id: `${mainNodeIds[i]}->${mainNodeIds[i + 1]}`,
      source: mainNodeIds[i],
      target: mainNodeIds[i + 1],
      type: 'smoothstep',
      animated: isRunning,
      markerEnd: { type: MarkerType.ArrowClosed, width: 16, height: 16 },
      style: { stroke: isRunning ? 'var(--status-running)' : 'var(--border-hover)', strokeWidth: 2 },
    });
  }

  // Add remaining fork groups that aren't direct children of main (nested sub-workflows)
  // These aren't laid out yet; place them as additional columns
  // For now, nested sub-workflow steps are included inline in their parent fork

  return { nodes, edges };
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

export default function WorkflowGraph({ steps }: { steps: Step[] }) {
  const { nodes, edges } = useMemo(() => buildGraph(steps), [steps]);
  const colorMode = useColorMode();

  const maxY = nodes.reduce((m, n) => Math.max(m, n.position.y), 0);
  const graphHeight = Math.max(300, maxY + NODE_H + 80);

  const onInit = useCallback((instance: { fitView: () => void }) => {
    setTimeout(() => instance.fitView(), 50);
  }, []);

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
