// Horizontal placement for the run graph (WorkflowGraph.tsx). Steps are grouped by fork ID into
// chains; a step's sub-workflow is drawn to its right, and a for_each step's branches are drawn
// side by side below it. Branches used to sit a fixed distance apart, so nested fan-outs
// (rounds -> tests -> variants) drew one branch's column on top of another's. Here each chain's
// extent is measured first, including everything it contains, and sibling branches are placed
// by their real widths.

export interface ChainStep {
  step_name: string;
}

export interface RunGraphMetrics {
  nodeW: number;
  childGapX: number; // offset of a sub-workflow chain from its caller
  groupPadX: number; // padding of the box drawn around a sub-workflow chain
  siblingGap: number; // space between neighbouring fork branches
  collapseThreshold: number; // more branches than this collapse into one summary node
}

// Extent of a chain relative to the x (left edge) of its own nodes.
export interface Extent {
  left: number; // <= 0
  right: number; // >= nodeW
}

export function childPath(path: string, stepName: string): string {
  return path ? `${path}-${stepName}` : stepName;
}

// The for_each branches under forkPrefix: fork IDs that start with it and aren't nested inside
// another such fork ID.
export function findForEachForks(groups: Map<string, ChainStep[]>, forkPrefix: string): string[] {
  const prefix = forkPrefix + '-';
  const candidates = Array.from(groups.keys()).filter(fid => fid.startsWith(prefix));
  return candidates.filter(fid => !candidates.some(other => other !== fid && fid.startsWith(other + '-')));
}

export function measureExtents(groups: Map<string, ChainStep[]>, m: RunGraphMetrics): Map<string, Extent> {
  const memo = new Map<string, Extent>();

  function measure(path: string): Extent {
    const known = memo.get(path);
    if (known) return known;
    let left = 0;
    let right = m.nodeW;
    for (const step of groups.get(path) || []) {
      const forkPrefix = childPath(path, step.step_name);
      if (groups.has(forkPrefix)) {
        // Sub-workflow chain, boxed, to the right.
        const c = measure(forkPrefix);
        left = Math.min(left, m.childGapX + c.left - m.groupPadX);
        right = Math.max(right, m.childGapX + c.right + m.groupPadX);
        continue;
      }
      const forks = findForEachForks(groups, forkPrefix);
      if (forks.length === 0 || forks.length > m.collapseThreshold) continue;
      const span = forkSpan(forks.map(measure), m);
      left = Math.min(left, span.start);
      right = Math.max(right, span.start + span.width);
    }
    const extent = { left, right };
    memo.set(path, extent);
    return extent;
  }

  for (const path of groups.keys()) measure(path);
  return memo;
}

// Where a fork's branches start, relative to the parent's x, and how wide they are together:
// centred under the parent node.
function forkSpan(extents: Extent[], m: RunGraphMetrics): { start: number; width: number } {
  const widths = extents.map(e => e.right - e.left);
  const width = widths.reduce((a, b) => a + b, 0) + m.siblingGap * Math.max(0, extents.length - 1);
  return { start: m.nodeW / 2 - width / 2, width };
}

// The x (left edge of its nodes) of each branch, side by side by their extents, centred under a
// parent at parentX.
export function placeForks(parentX: number, branchExtents: Extent[], m: RunGraphMetrics): number[] {
  const span = forkSpan(branchExtents, m);
  let cursor = parentX + span.start;
  return branchExtents.map(e => {
    const x = cursor - e.left;
    cursor += e.right - e.left + m.siblingGap;
    return x;
  });
}
