import { useEffect, useState } from "react";

import { Tag } from "~/shared/components/Tag";
import { parseToolCallPayload } from "../model";

export type SessionLogEntry = {
  seq?: number;
  kind?: string;
  at?: string;
  payload?: unknown;
};

export function SessionLogStream({
  sessionId,
  initialEntries,
}: {
  sessionId?: string;
  initialEntries?: SessionLogEntry[];
}) {
  const [entries, setEntries] = useState<SessionLogEntry[]>(
    initialEntries ?? [],
  );
  const [status, setStatus] = useState<"idle" | "connected" | "closed">(
    sessionId ? "connected" : "idle",
  );

  useEffect(() => {
    if (!sessionId) return;
    setEntries(initialEntries ?? []);
    setStatus("connected");

    const es = new EventSource(`/api/sessions/${sessionId}/events`, {
      withCredentials: true,
    });

    es.onmessage = (evt) => {
      try {
        const parsed = JSON.parse(evt.data) as SessionLogEntry;
        setEntries((prev) => [...prev, parsed]);
      } catch {
        setEntries((prev) => [...prev, { payload: evt.data }]);
      }
    };

    es.onerror = () => {
      setStatus("closed");
      es.close();
    };

    return () => {
      es.close();
    };
  }, [sessionId, initialEntries]);

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
        <span>{status === "connected" ? "Live SSE" : "Stream closed"}</span>
      </div>
      <div className="max-h-96 overflow-y-auto p-4 font-mono text-xs">
        {entries.length === 0 ? (
          <p className="text-muted">Waiting for session events…</p>
        ) : (
          <ul className="space-y-1.5">
            {entries.map((entry, idx) => {
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
