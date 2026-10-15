import assert from 'node:assert/strict';
import { toLocalInput, toScheduledInstant } from './scheduleTime.ts';

process.env.TZ = 'America/New_York';
const originalNow = Date.now;
Date.now = () => new Date('2026-10-15T03:00:00Z').getTime();
try {
  assert.equal(toScheduledInstant('2026-10-15T00:30'), '2026-10-15T04:30:00.000Z');
  assert.equal(toLocalInput('2026-10-15T04:30:00Z'), '2026-10-15T00:30');
  assert.equal(toScheduledInstant('2026-11-02T00:30'), '2026-11-02T05:30:00.000Z');
  assert.throws(() => toScheduledInstant('2026-10-14T22:30'), /future/);
  assert.throws(() => toScheduledInstant('2027-03-14T02:30'), /daylight saving/);
  assert.throws(() => toScheduledInstant(''), /valid/);
} finally { Date.now = originalNow; }
