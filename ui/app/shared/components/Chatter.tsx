import { useId, useState } from "react";

import { Avatar } from "./Avatar";
import { Button } from "./Button";
import { Icon } from "./Icon";

export type ChatterKind = "message" | "note" | "tracking";

export type ChatterMessage = {
  id: string;
  kind: ChatterKind;
  author: string;
  body: string;
  /** ISO timestamp */
  at: string;
};

const stamp = new Intl.DateTimeFormat(undefined, {
  dateStyle: "medium",
  timeStyle: "short",
});

type ChatterProps = {
  messages: ChatterMessage[];
  onPost: (kind: "message" | "note", body: string) => void;
  /** Disable posting (e.g. record not saved yet). */
  disabledReason?: string;
};

/** Chatter: send message, log note, and a tracking timeline. */
export function Chatter({ messages, onPost, disabledReason }: ChatterProps) {
  const [mode, setMode] = useState<"message" | "note" | null>(null);
  const [body, setBody] = useState("");
  const textareaId = useId();
  const disabled = Boolean(disabledReason);

  function submit() {
    const text = body.trim();
    if (!text || !mode) return;
    onPost(mode, text);
    setBody("");
    setMode(null);
  }

  const timeline = [...messages].reverse();

  return (
    <section aria-label="Chatter" className="flex flex-col gap-4">
      <div className="flex items-center gap-2">
        <Button
          variant={mode === "message" ? "primary" : "secondary"}
          disabled={disabled}
          aria-pressed={mode === "message"}
          onClick={() => setMode(mode === "message" ? null : "message")}
        >
          <Icon name="send" size={15} />
          Send message
        </Button>
        <Button
          variant={mode === "note" ? "primary" : "secondary"}
          disabled={disabled}
          aria-pressed={mode === "note"}
          onClick={() => setMode(mode === "note" ? null : "note")}
        >
          <Icon name="note" size={15} />
          Log note
        </Button>
      </div>

      {disabledReason && <p className="text-muted">{disabledReason}</p>}

      {mode && (
        <div className="flex flex-col gap-2">
          <label htmlFor={textareaId} className="sr-only">
            {mode === "message" ? "Message" : "Internal note"}
          </label>
          <textarea
            id={textareaId}
            className="field min-h-24 resize-y"
            placeholder={
              mode === "message"
                ? "Send a message to followers…"
                : "Log an internal note…"
            }
            value={body}
            onChange={(e) => setBody(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter" && (e.metaKey || e.ctrlKey)) submit();
            }}
          />
          <div className="flex items-center gap-2">
            <Button variant="primary" disabled={!body.trim()} onClick={submit}>
              {mode === "message" ? "Send" : "Log"}
            </Button>
            <Button variant="ghost" onClick={() => setMode(null)}>
              Cancel
            </Button>
            <span className="ml-auto text-xs text-muted">
              Ctrl+Enter to post
            </span>
          </div>
        </div>
      )}

      <ol className="flex flex-col gap-3">
        {timeline.length === 0 && (
          <li className="text-muted">No activity yet.</li>
        )}
        {timeline.map((m) =>
          m.kind === "tracking" ? (
            <li
              key={m.id}
              className="flex items-start gap-2 text-xs text-muted"
            >
              <span className="mt-0.5 inline-flex size-6 shrink-0 items-center justify-center">
                <span className="size-1.5 rounded-full bg-line-strong" />
              </span>
              <span>
                {m.body} ·{" "}
                <time dateTime={m.at}>{stamp.format(new Date(m.at))}</time>
              </span>
            </li>
          ) : (
            <li key={m.id} className="flex items-start gap-2">
              <Avatar name={m.author} size={28} />
              <div
                className={`min-w-0 flex-1 rounded-lg px-3 py-2 ${
                  m.kind === "note"
                    ? "bg-accent-soft"
                    : "border border-line bg-surface"
                }`}
              >
                <p className="flex flex-wrap items-baseline gap-x-2">
                  <span className="font-semibold">{m.author}</span>
                  <time dateTime={m.at} className="text-xs text-muted">
                    {stamp.format(new Date(m.at))}
                  </time>
                  {m.kind === "note" && (
                    <span className="text-xs font-medium text-accent-ink">
                      Internal note
                    </span>
                  )}
                </p>
                <p className="mt-1 whitespace-pre-wrap">{m.body}</p>
              </div>
            </li>
          ),
        )}
      </ol>
    </section>
  );
}
