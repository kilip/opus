/** Local date as YYYY-MM-DD. */
export function todayISO() {
  return new Date().toLocaleDateString("en-CA");
}

/** Format YYYY-MM-DD deadline, e.g. "Nov 20". */
export function formatDeadline(date: string) {
  return new Date(`${date}T00:00:00`).toLocaleDateString(undefined, {
    month: "short",
    day: "numeric",
  });
}

/** Format an ISO timestamp as a concise relative string, e.g. "5m ago". */
export function formatRelativeTime(iso: string, now = Date.now()): string {
  const ms = new Date(iso).getTime();
  if (Number.isNaN(ms)) return iso;
  const diffSec = Math.max(0, Math.floor((now - ms) / 1000));
  if (diffSec < 60) return "just now";
  const diffMin = Math.floor(diffSec / 60);
  if (diffMin < 60) return `${diffMin}m ago`;
  const diffHr = Math.floor(diffMin / 60);
  if (diffHr < 24) return `${diffHr}h ago`;
  const diffDay = Math.floor(diffHr / 24);
  return `${diffDay}d ago`;
}

/** Format an ISO timestamp as a readable local date/time string. */
export function formatAbsoluteTime(iso: string): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  return d.toLocaleString();
}
