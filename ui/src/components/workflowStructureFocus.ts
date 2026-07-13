export type StructureFocusNode = {
  id: string;
  data: {
    invocationPath?: string;
  };
};

export type StructureFocusEdge = {
  id: string;
  source: string;
  target: string;
  relation?: 'sequence' | 'call' | 'return';
};

export type StructureSelection = {
  kind: 'node' | 'edge';
  id: string;
};

export type StructureFocus = {
  nodeIDs: Set<string>;
  edgeIDs: Set<string>;
  invocationPath: string;
};

function isInvocationInSubtree(candidate: string | undefined, root: string): boolean {
  return candidate === root || candidate?.startsWith(`${root}/`) === true;
}
export function resolveStructureFocus(
  nodes: StructureFocusNode[],
  edges: StructureFocusEdge[],
  selection: StructureSelection | null,
): StructureFocus | null {
  if (!selection) return null;

  const nodeByID = new Map(nodes.map((node) => [node.id, node]));
  const selectedEdge = selection.kind === 'edge'
    ? edges.find((edge) => edge.id === selection.id)
    : undefined;
  const anchorNode = selection.kind === 'node'
    ? nodeByID.get(selection.id)
    : selectedEdge
      ? nodeByID.get(selectedEdge.relation === 'call' ? selectedEdge.target : selectedEdge.source)
      : undefined;
  const invocationPath = anchorNode?.data.invocationPath;
  if (!invocationPath) return null;

  const nodeIDs = new Set(
    nodes
      .filter((node) => isInvocationInSubtree(node.data.invocationPath, invocationPath))
      .map((node) => node.id),
  );
  const edgeIDs = new Set<string>();

  for (const edge of edges) {
    const sourceInFocus = nodeIDs.has(edge.source);
    const targetInFocus = nodeIDs.has(edge.target);
    if (sourceInFocus && targetInFocus) {
      edgeIDs.add(edge.id);
      continue;
    }
    if (edge.relation === 'call' && targetInFocus) {
      nodeIDs.add(edge.source);
      edgeIDs.add(edge.id);
    }
    if (edge.relation === 'return' && sourceInFocus) {
      nodeIDs.add(edge.target);
      edgeIDs.add(edge.id);
    }
  }

  if (selectedEdge) edgeIDs.add(selectedEdge.id);
  return { nodeIDs, edgeIDs, invocationPath };
}
