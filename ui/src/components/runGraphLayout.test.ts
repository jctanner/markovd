import assert from 'node:assert/strict';
import test from 'node:test';
import { childPath, findForEachForks, measureExtents, placeForks, type ChainStep, type Extent } from './runGraphLayout.ts';

const m = { nodeW: 294, childGapX: 340, groupPadX: 44, siblingGap: 40, collapseThreshold: 5 };

function steps(...names: string[]): ChainStep[] {
  return names.map(step_name => ({ step_name }));
}

// The strat-workflow benchmark's shape: rounds -> tests (S1, S2) -> variants, two rounds.
function benchmarkGroups(): Map<string, ChainStep[]> {
  const g = new Map<string, ChainStep[]>();
  g.set('', steps('wipe', 'rounds', 'summarize'));
  g.set('wipe', steps('a', 'b'));
  for (const r of ['rounds-0', 'rounds-1']) {
    g.set(r, steps('tests'));
    for (const t of ['S1', 'S2']) {
      g.set(`${r}-tests-${t}`, steps('variants'));
      g.set(`${r}-tests-${t}-variants-0`, steps('select_arm', 'submit', 'wait_for_run'));
      g.set(`${r}-tests-${t}-variants-0-submit`, steps('submit_job'));
    }
  }
  return g;
}

// Lay out every chain the way WorkflowGraph does and return each node's [left, right] by path.
function layout(g: Map<string, ChainStep[]>): Map<string, Array<[number, number]>> {
  const ext = measureExtents(g, m);
  const boxes = new Map<string, Array<[number, number]>>();
  function chain(path: string, x: number) {
    boxes.set(path, [[x, x + m.nodeW]]);
    for (const s of g.get(path) || []) {
      const prefix = childPath(path, s.step_name);
      if (g.has(prefix)) {
        chain(prefix, x + m.childGapX);
        continue;
      }
      const forks = findForEachForks(g, prefix);
      if (forks.length === 0) continue;
      const xs = placeForks(x, forks.map(f => ext.get(f) as Extent), m);
      forks.forEach((f, i) => chain(f, xs[i]));
    }
  }
  chain('', 0);
  return boxes;
}

test('sibling branches of nested fan-outs never share columns', () => {
  const boxes = layout(benchmarkGroups());
  const branches = ['rounds-0-tests-S1', 'rounds-0-tests-S2', 'rounds-1-tests-S1', 'rounds-1-tests-S2']
    .map(p => boxes.get(p)![0]);
  for (let i = 0; i < branches.length; i++) {
    for (let j = i + 1; j < branches.length; j++) {
      const [a, b] = [branches[i], branches[j]];
      assert.ok(a[1] <= b[0] || b[1] <= a[0], `columns ${i} and ${j} overlap: ${a} vs ${b}`);
    }
  }
});

test('each branch, with its sub-workflow box, fits between its neighbours', () => {
  const g = benchmarkGroups();
  const ext = measureExtents(g, m);
  const xs = placeForks(0, ['rounds-0', 'rounds-1'].map(p => ext.get(p)!), m);
  const r0 = ext.get('rounds-0')!;
  const r1 = ext.get('rounds-1')!;
  assert.ok(xs[0] + r0.right + m.siblingGap <= xs[1] + r1.left + 1e-9, 'round 0 reaches into round 1');
  // The variants chain's submit sub-workflow (and its box) is inside its branch's extent.
  const v = ext.get('rounds-0-tests-S1-variants-0')!;
  assert.equal(v.right, m.childGapX + m.nodeW + m.groupPadX);
});

test('branches are centred under their parent', () => {
  const e: Extent = { left: 0, right: 294 };
  const xs = placeForks(100, [e, e], m);
  const mid = (xs[0] + xs[1] + m.nodeW) / 2;
  assert.equal(mid, 100 + m.nodeW / 2);
  assert.equal(xs[1] - xs[0], m.nodeW + m.siblingGap, 'neighbouring plain branches no longer overlap');
});

test('a collapsed fan-out adds no width', () => {
  const g = new Map<string, ChainStep[]>([['', steps('each')]]);
  for (let i = 0; i < 6; i++) g.set(`each-${i}`, steps('x'));
  assert.deepEqual(measureExtents(g, m).get(''), { left: 0, right: m.nodeW });
});
