import { useEffect, useRef, useState } from "react";

import { Tag } from "~/shared/components/Tag";
import { parseToolCallPayload } from "../model";

export const DEFAULT_MAX_LOG_ENTRIES = 2000;
const BASE_RECONNECT_DELAY_MS = 1000;
const MAX_RECONNECT_DELAY_MS = 30_000;

export type SessionLogEntry = {
  seq?: number;
  kind?: string;
  at?: string;
  payload?: unknown;
};

export function nextReconnectDelayMs(attempt: number): number {
  return Math.min(
    BASE_RECONNECT_DELAY_MS * 2 ** Math.max(0, attempt),
    MAX_RECONNECT_DELAY_MS,
  );
}

export function buildSessionEventsUrl(
  sessionId: string,
  afterSeq?: number,
): string {
  const base = `/api/sessions/${sessionId}/events`;
  if (afterSeq !== undefined) {
    return `${base}?after_seq=${encodeURIComponent(String(afterSeq))}`;
  }
  return base;
}

export function appendLogEntries(
  prev: SessionLogEntry[],
  next: SessionLogEntry[],
  maxEntries: number,
): { entries: SessionLogEntry[]; dropped: number } {
  const combined = [...prev, ...next];
  if (maxEntries > 0 && combined.length > maxEntries) {
    const dropped = combined.length - maxEntries;
    return { entries: combined.slice(dropped), dropped };
  }
  return { entries: combined, dropped: 0 };
}

export function SessionLogStream({
  sessionId,
  initialEntries,
  maxEntries = DEFAULT_MAX_LOG_ENTRIES,
}: {
  sessionId?: string;
  initialEntries?: SessionLogEntry[];
  maxEntries?: number;
}) {
  const initialRef = useRef(initialEntries);
  initialRef.current = initialEntries;

  const [state, setState] = useState<{
    entries: SessionLogEntry[];
    truncatedCount: number;
  }>(() => {
    const seeded = initialEntries ?? [];
    const { entries, dropped } = appendLogEntries([], seeded, maxEntries);
    return { entries, truncatedCount: dropped };
  });
  const [status, setStatus] = useState<
    "idle" | "connected" | "reconnecting" | "closed"
  >(sessionId ? "connected" : "idle");

  useEffect(() => {
    if (!sessionId) return;

    const seeded = initialRef.current ?? [];
    const initial = appendLogEntries([], seeded, maxEntries);
    setState({ entries: initial.entries, truncatedCount: initial.dropped });
    setStatus("connected");

    let lastSeq: number | undefined;
    for (const entry of initial.entries) {
      if (typeof entry.seq === "number") {
        lastSeq = entry.seq;
      }
    }

    let cancelled = false;
    let attempt = 0;
    let es: EventSource | null = null;
    let timer: ReturnType<typeof setTimeout> | null = null;

    const connect = () => {
      if (cancelled) return;
      const url = buildSessionEventsUrl(sessionId, lastSeq);
      es = new EventSource(url, { withCredentials: true });
      setStatus("connected");

      es.onopen = () => {
        attempt = 0;
        setStatus("connected");
      };

      es.onmessage = (evt) => {
        attempt = 0;
        let parsed: SessionLogEntry;
        try {
          parsed = JSON.parse(evt.data) as SessionLogEntry;
        } catch {
          parsed = { payload: evt.data };
        }
        if (typeof parsed.seq === "number") {
          lastSeq = parsed.seq;
        }
        setState((prev) => {
          const next = appendLogEntries(prev.entries, [parsed], maxEntries);
          return {
            entries: next.entries,
            truncatedCount: prev.truncatedCount + next.dropped,
          };
        });
      };

      es.onerror = () => {
        es?.close();
        es = null;
        if (cancelled) return;
        setStatus("reconnecting");
        const delay = nextReconnectDelayMs(attempt);
        attempt += 1;
        timer = setTimeout(connect, delay);
      };
    };

    connect();

    return () => {
      cancelled = true;
      if (timer !== null) clearTimeout(timer);
      es?.close();
      setStatus("closed");
    };
  }, [sessionId, maxEntries]);

  if (!sessionId) {
    return (
      <div className="rounded-lg border border-line bg-surface p-4 text-sm text-muted">
        No session attached to this run yet.
      </div>
    );
  }

  return (
    <div className="rounded-lg border border-line bg-surface">
      <div className="flex items-center justify-between border-b border-line px-4 py-2 text-xs text-muted">
        <span className="font-mono">Session {sessionId}</span>
        <span>
          {status === "connected"
            ? "Live SSE"
            : status === "reconnecting"
              ? "Reconnecting…"
              : "Stream closed"}
        </span>
      </div>
      <div
        role="log"
        aria-live="polite"
        className="max-h-96 overflow-y-auto p-4 font-mono text-xs"
      >
        {state.truncatedCount > 0 && (
          <p className="mb-2 rounded bg-surface-2 px-2.5 py-1 text-muted">
            {state.truncatedCount} older entries hidden (showing latest{" "}
            {state.entries.length})
          </p>
        )}
        {state.entries.length === 0 ? (
          <p className="text-muted">Waiting for session events…</p>
        ) : (
          <ul className="space-y-1.5">
            {state.entries.map((entry, idx) => {
              const toolCall =
                entry.kind === "tool_call"
                  ? parseToolCallPayload(entry.payload)
                  : null;
              if (toolCall) {
                const isOk = toolCall.status === "ok";
                return (
                  <li
                    key={entry.seq ?? idx}
                    className="flex flex-wrap items-center gap-2 rounded bg-surface-2 px-2.5 py-1.5"
                  >
                    <span className="text-muted">
                      #{entry.seq ?? idx + 1} [tool_call]
                    </span>
                    <span className="font-semibold text-fg">
                      {toolCall.tool}
                    </span>
                    <Tag tone={isOk ? "success" : "danger"}>
                      {toolCall.status}
                    </Tag>
                    {toolCall.duration_ms !== undefined && (
                      <span className="text-muted">
                        {`${toolCall.duration_ms}ms`}
                      </span>
                    )}
                    {toolCall.error && (
                      <span className="text-danger-ink">{toolCall.error}</span>
                    )}
                  </li>
                );
              }
              return (
                <li
                  key={entry.seq ?? idx}
                  className="rounded bg-surface-2 px-2.5 py-1.5"
                >
                  <span className="text-muted">
                    #{entry.seq ?? idx + 1} [{entry.kind ?? "event"}]{" "}
                  </span>
                  <span className="break-all text-fg">
                    {typeof entry.payload === "string"
                      ? entry.payload
                      : JSON.stringify(entry.payload ?? {})}
                  </span>
                </li>
              );
            })}
          </ul>
        )}
      </div>
    </div>
  );
}
