export function toLocalInput(instant: string): string {
  const date = new Date(instant);
  const pad = (value: number) => String(value).padStart(2, '0');
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}`;
}

export function toScheduledInstant(local: string): string {
  const date = new Date(local);
  if (!local || !Number.isFinite(date.getTime()) || toLocalInput(date.toISOString()) !== local) {
    throw new Error('Choose a valid local date and time. Times skipped by daylight saving are invalid.');
  }
  if (date.getTime() <= Date.now()) throw new Error('The scheduled time must be in the future.');
  return date.toISOString();
}
