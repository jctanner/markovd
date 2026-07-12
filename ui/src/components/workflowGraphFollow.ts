import type { Step } from '../api';

export function latestRunningStep(steps: Step[]): Step | null {
  let latest: Step | null = null;

  for (const step of steps) {
    if (step.status !== 'running') continue;
    if (!latest) {
      latest = step;
      continue;
    }

    const updatedAt = step.updated_at || '';
    const latestUpdatedAt = latest.updated_at || '';
    if (updatedAt > latestUpdatedAt || (updatedAt === latestUpdatedAt && step.id > latest.id)) {
      latest = step;
    }
  }

  return latest;
}

export function resolveFollowTarget(
  steps: Step[],
  followNodeByStepId: ReadonlyMap<number, string>,
): string | null {
  const step = latestRunningStep(steps);
  return step ? followNodeByStepId.get(step.id) ?? null : null;
}
