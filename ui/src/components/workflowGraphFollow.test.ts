import assert from 'node:assert/strict';
import test from 'node:test';
import type { Step } from '../api';
import { latestRunningStep, resolveFollowTarget } from './workflowGraphFollow.ts';

function step(id: number, status: string, updatedAt: string | null): Step {
  return {
    id,
    run_id: 'run-1',
    fork_id: '',
    workflow_name: 'main',
    step_name: `step-${id}`,
    step_type: 'shell_exec',
    status,
    output_json: '',
    error: '',
    started_at: null,
    completed_at: null,
    updated_at: updatedAt,
  };
}

test('selects the most recently updated running step', () => {
  assert.equal(latestRunningStep([
    step(1, 'completed', '2026-07-12T12:00:02Z'),
    step(2, 'running', '2026-07-12T12:00:01Z'),
    step(3, 'running', '2026-07-12T12:00:03Z'),
  ])?.id, 3);
});

test('uses the greater stable step id when timestamps tie', () => {
  assert.equal(latestRunningStep([
    step(4, 'running', '2026-07-12T12:00:01Z'),
    step(7, 'running', '2026-07-12T12:00:01Z'),
  ])?.id, 7);
});

test('returns no target when no step is running', () => {
  assert.equal(resolveFollowTarget(
    [step(1, 'completed', '2026-07-12T12:00:01Z')],
    new Map([[1, 'main::step-1']]),
  ), null);
});

test('resolves a hidden running step to its collapsed fork summary', () => {
  assert.equal(resolveFollowTarget(
    [step(8, 'running', '2026-07-12T12:00:01Z')],
    new Map([[8, 'summary::main-fan-out']]),
  ), 'summary::main-fan-out');
});
