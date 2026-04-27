import { useCallback, useState, useEffect } from 'react';
import {
  ReactFlow,
  Background,
  BackgroundVariant,
  Controls,
  MiniMap,
  Handle,
  Position,
  useReactFlow,
} from '@xyflow/react';
import type { Node, Edge, NodeProps } from '@xyflow/react';
import '@xyflow/react/dist/style.css';
import type { DiagramNode, DiagramEdge } from '../api';

const categoryBorder: Record<string, string> = {
  gate: '#d97706',
  foreach: '#3b82f6',
  subworkflow: '#7c3aed',
  conditional: '#db2777',
  normal: 'var(--border)',
};

const categoryLabel: Record<string, string> = {
  gate: 'GATE',
  foreach: 'FOR EACH',
  subworkflow: 'SUB-WORKFLOW',
  conditional: 'CONDITIONAL',
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
    case 'loop':
      return <svg {...props}><polyline points="17 1 21 5 17 9" /><path d="M3 11V9a4 4 0 0 1 4-4h14" /><polyline points="7 23 3 19 7 15" /><path d="M21 13v2a4 4 0 0 1-4 4H3" /></svg>;
    case 'git-branch':
      return <svg {...props}><line x1="6" y1="3" x2="6" y2="15" /><circle cx="18" cy="6" r="3" /><circle cx="6" cy="18" r="3" /><path d="M18 9a9 9 0 0 1-9 9" /></svg>;
    case 'help':
      return <svg {...props}><circle cx="12" cy="12" r="10" /><path d="M9.09 9a3 3 0 0 1 5.83 1c0 2-3 3-3 3" /><line x1="12" y1="17" x2="12.01" y2="17" /></svg>;
    default:
      return <svg {...props}><circle cx="12" cy="12" r="4" /></svg>;
  }
}

type StructureNodeData = {
  label: string;
  stepType: string;
  category: string;
  forEach?: string;
  subWorkflow?: string;
  when?: string;
  workflowGroup: string;
};

function StructureStepNode({ data }: NodeProps<Node<StructureNodeData>>) {
  const border = categoryBorder[data.category] || categoryBorder.normal;
  const icon = data.category === 'foreach' ? 'loop'
    : data.category === 'subworkflow' ? 'git-branch'
    : data.category === 'conditional' ? 'help'
    : typeIcons[data.stepType] || 'circle';

  const tag = categoryLabel[data.category];

  return (
    <div
      className={`graph-node struct-node struct-node-${data.category}`}
      style={{ borderLeftColor: border }}
    >
      <Handle type="target" position={Position.Top} className="graph-handle" />
      <div className="graph-node-top">
        <span className="graph-node-icon" title={data.stepType || 'step'}>
          {iconSvg(icon)}
        </span>
        <span className="graph-node-name">{data.label}</span>
      </div>
      <div className="graph-node-bottom">
        {data.stepType && (
          <span className="struct-node-type">{data.stepType}</span>
        )}
        {tag && (
          <span className="struct-node-tag" style={{ color: border }}>
            {tag}
          </span>
        )}
      </div>
      {data.subWorkflow && (
        <div className="struct-node-sub">{data.subWorkflow}</div>
      )}
      <Handle type="source" position={Position.Bottom} className="graph-handle" />
    </div>
  );
}

type GroupNodeData = {
  label: string;
  workflowGroup: string;
  category: string;
};

function WorkflowGroupNode({ data }: NodeProps<Node<GroupNodeData>>) {
  return (
    <div className="struct-group-node">
      <div className="struct-group-label">{data.label}</div>
    </div>
  );
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

function miniMapColor(node: Node): string {
  const d = node.data as Record<string, unknown>;
  const cat = (d.category as string) || '';
  return categoryBorder[cat] || 'var(--border-hover)';
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

const nodeTypes = {
  workflowStep: StructureStepNode,
  group: WorkflowGroupNode,
};

interface Props {
  nodes: DiagramNode[];
  edges: DiagramEdge[];
}

export default function WorkflowStructureGraph({ nodes: rawNodes, edges: rawEdges }: Props) {
  const colorMode = useColorMode();
  const [fullscreen, setFullscreen] = useState(false);

  const nodes: Node[] = rawNodes.map((n) => ({
    id: n.id,
    type: n.type,
    position: n.position,
    data: n.data,
    parentId: n.parentId,
    extent: n.extent as 'parent' | undefined,
    style: n.style,
  }));

  const edges: Edge[] = rawEdges.map((e) => ({
    id: e.id,
    source: e.source,
    target: e.target,
    type: e.type || 'smoothstep',
    animated: e.animated || false,
    style: e.style ? {
      stroke: 'var(--border-hover)',
      strokeWidth: 2,
      ...e.style,
    } : {
      stroke: 'var(--border-hover)',
      strokeWidth: 2,
    },
  }));

  const maxY = nodes.reduce((m, n) => Math.max(m, n.position.y), 0);
  const graphHeight = Math.max(400, maxY + 150);

  const onInit = useCallback((instance: { fitView: () => void }) => {
    setTimeout(() => instance.fitView(), 50);
  }, []);

  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && fullscreen) setFullscreen(false);
    };
    window.addEventListener('keydown', handler);
    return () => window.removeEventListener('keydown', handler);
  }, [fullscreen]);

  if (nodes.length === 0) {
    return <div className="graph-empty">No workflow structure to display.</div>;
  }

  return (
    <div
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
        fitView
        fitViewOptions={{ padding: 0.3 }}
        nodesDraggable={false}
        nodesConnectable={false}
        elementsSelectable={false}
        panOnDrag
        zoomOnScroll
        minZoom={0.05}
        maxZoom={2}
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
  );
}
