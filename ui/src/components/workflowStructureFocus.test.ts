import assert from 'node:assert/strict';
import test from 'node:test';
import { resolveStructureFocus } from './workflowStructureFocus.ts';

const nodes = [
  { id: 'root-call', data: { invocationPath: 'main' } },
  { id: 'root-next', data: { invocationPath: 'main' } },
  { id: 'root-other', data: { invocationPath: 'main' } },
  { id: 'child-group', data: { invocationPath: 'main/call@child' } },
  { id: 'child-first', data: { invocationPath: 'main/call@child' } },
  { id: 'child-last', data: { invocationPath: 'main/call@child' } },
  { id: 'nested', data: { invocationPath: 'main/call@child/nested@leaf' } },
  { id: 'second-child', data: { invocationPath: 'main/other@child' } },
];

const edges = [
  { id: 'call', source: 'root-call', target: 'child-first', relation: 'call' as const },
  { id: 'inside', source: 'child-first', target: 'child-last', relation: 'sequence' as const },
  { id: 'nested-call', source: 'child-last', target: 'nested', relation: 'call' as const },
  { id: 'return', source: 'nested', target: 'root-next', relation: 'return' as const },
  { id: 'unrelated', source: 'root-next', target: 'root-other', relation: 'sequence' as const },
  { id: 'other-call', source: 'root-other', target: 'second-child', relation: 'call' as const },
];

test('focuses one invocation subtree with its caller and return boundary', () => {
  const focus = resolveStructureFocus(nodes, edges, { kind: 'node', id: 'child-first' });
  assert.ok(focus);
  assert.equal(focus.invocationPath, 'main/call@child');
  assert.deepEqual([...focus.nodeIDs].sort(), [
    'child-first',
    'child-group',
    'child-last',
    'nested',
    'root-call',
    'root-next',
  ]);
  assert.deepEqual([...focus.edgeIDs].sort(), ['call', 'inside', 'nested-call', 'return']);
});
test('uses the called invocation when a call edge is selected', () => {
  const focus = resolveStructureFocus(nodes, edges, { kind: 'edge', id: 'call' });
  assert.equal(focus?.invocationPath, 'main/call@child');
  assert.equal(focus?.edgeIDs.has('other-call'), false);
});

test('uses the returning invocation when a return edge is selected', () => {
  const focus = resolveStructureFocus(nodes, edges, { kind: 'edge', id: 'return' });
  assert.equal(focus?.invocationPath, 'main/call@child/nested@leaf');
  assert.deepEqual([...focus!.edgeIDs].sort(), ['nested-call', 'return']);
});

test('returns no focus when selection cannot be resolved', () => {
  assert.equal(resolveStructureFocus(nodes, edges, { kind: 'node', id: 'missing' }), null);
  assert.equal(resolveStructureFocus(nodes, edges, null), null);
});
