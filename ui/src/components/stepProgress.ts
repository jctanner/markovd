import type { StepProgressEvent } from '../api';

export interface ProgressSummary {
  turns: number;
  tokens: number;
  costUSD: number | null;
  limitExceeded: string | null;
  finished: boolean;
}

// Events that mean the step will emit nothing further.
const terminalKinds = new Set(['result', 'limit_exceeded']);

export function isTerminalKind(kind: string): boolean {
  return terminalKinds.has(kind);
}

function num(v: unknown): number | null {
  return typeof v === 'number' && Number.isFinite(v) ? v : null;
}

// summarizeProgress folds a step's progress events into the running totals
// shown above the transcript. `usage` events carry running totals, so the
// latest one wins; `result` carries the authoritative final numbers.
export function summarizeProgress(events: StepProgressEvent[]): ProgressSummary {
  const summary: ProgressSummary = { turns: 0, tokens: 0, costUSD: null, limitExceeded: null, finished: false };
  for (const e of events) {
    switch (e.kind) {
      case 'usage':
        summary.turns = num(e.data.turns) ?? summary.turns;
        summary.tokens = num(e.data.tokens) ?? summary.tokens;
        break;
      case 'result':
        summary.turns = num(e.data.num_turns) ?? summary.turns;
        summary.costUSD = num(e.data.total_cost_usd);
        summary.finished = true;
        break;
      case 'limit_exceeded':
        summary.limitExceeded = typeof e.data.limit === 'string' ? e.data.limit : 'unknown';
        summary.finished = true;
        break;
    }
  }
  return summary;
}

// mergeProgress appends newly fetched events, ignoring any id already seen
// (a poll can overlap a previous one).
export function mergeProgress(existing: StepProgressEvent[], incoming: StepProgressEvent[]): StepProgressEvent[] {
  if (incoming.length === 0) return existing;
  const seen = new Set(existing.map((e) => e.id));
  const fresh = incoming.filter((e) => !seen.has(e.id));
  return fresh.length === 0 ? existing : [...existing, ...fresh];
}

// describeToolInput renders a tool call's input as one short line.
export function describeToolInput(input: unknown, max = 160): string {
  if (input === null || input === undefined) return '';
  let text: string;
  if (typeof input === 'string') {
    text = input;
  } else if (typeof input === 'object') {
    const obj = input as Record<string, unknown>;
    const primary = ['command', 'file_path', 'path', 'pattern', 'url', 'query', 'prompt']
      .map((k) => obj[k])
      .find((v) => typeof v === 'string');
    text = typeof primary === 'string' ? primary : JSON.stringify(input);
  } else {
    text = String(input);
  }
  text = text.replace(/\s+/g, ' ').trim();
  return text.length > max ? `${text.slice(0, max - 1)}…` : text;
}
