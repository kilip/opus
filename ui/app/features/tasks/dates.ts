/** Local date as YYYY-MM-DD. */
export function todayISO() {
  return new Date().toLocaleDateString("en-CA");
}

/** Format a YYYY-MM-DD deadline, e.g. "Nov 20". */
export function formatDeadline(date: string) {
  return new Date(`${date}T00:00:00`).toLocaleDateString(undefined, {
    month: "short",
    day: "numeric",
  });
}
