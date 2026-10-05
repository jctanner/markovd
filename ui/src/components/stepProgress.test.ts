import assert from 'node:assert/strict';
import test from 'node:test';
import type { StepProgressEvent } from '../api';
import { describeToolInput, isTerminalKind, mergeProgress, summarizeProgress } from './stepProgress.ts';

function ev(id: number, kind: string, data: Record<string, unknown> = {}): StepProgressEvent {
  return { id, run_id: 'r', kind, data, received_at: '2026-10-05T00:00:00Z' };
}

test('summarizeProgress tracks running usage and the final result', () => {
  const live = summarizeProgress([ev(1, 'init'), ev(2, 'usage', { turns: 1, tokens: 100 }), ev(3, 'usage', { turns: 2, tokens: 250 })]);
  assert.deepEqual(live, { turns: 2, tokens: 250, costUSD: null, limitExceeded: null, finished: false });

  const done = summarizeProgress([ev(1, 'usage', { turns: 2, tokens: 250 }), ev(2, 'result', { num_turns: 3, total_cost_usd: 0.5 })]);
  assert.equal(done.turns, 3);
  assert.equal(done.tokens, 250);
  assert.equal(done.costUSD, 0.5);
  assert.equal(done.finished, true);
});

test('summarizeProgress reports an exceeded limit as finished', () => {
  const s = summarizeProgress([ev(1, 'limit_exceeded', { limit: 'max_turns' })]);
  assert.equal(s.limitExceeded, 'max_turns');
  assert.equal(s.finished, true);
});

test('summarizeProgress ignores malformed numbers', () => {
  const s = summarizeProgress([ev(1, 'usage', { turns: 'x', tokens: null })]);
  assert.equal(s.turns, 0);
  assert.equal(s.tokens, 0);
});

test('mergeProgress appends only unseen events and keeps identity when nothing is new', () => {
  const a = [ev(1, 'init'), ev(2, 'text')];
  assert.equal(mergeProgress(a, []), a);
  assert.equal(mergeProgress(a, [ev(2, 'text')]), a);
  const merged = mergeProgress(a, [ev(2, 'text'), ev(3, 'result')]);
  assert.deepEqual(merged.map((e) => e.id), [1, 2, 3]);
});

test('isTerminalKind', () => {
  assert.equal(isTerminalKind('result'), true);
  assert.equal(isTerminalKind('limit_exceeded'), true);
  assert.equal(isTerminalKind('text'), false);
});

test('describeToolInput prefers the primary field and truncates', () => {
  assert.equal(describeToolInput({ command: 'echo   hi', description: 'x' }), 'echo hi');
  assert.equal(describeToolInput({ file_path: '/a/b.go' }), '/a/b.go');
  assert.equal(describeToolInput({ a: 1 }), '{"a":1}');
  assert.equal(describeToolInput(null), '');
  const long = describeToolInput({ command: 'x'.repeat(500) }, 20);
  assert.equal(long.length, 20);
  assert.ok(long.endsWith('…'));
});
