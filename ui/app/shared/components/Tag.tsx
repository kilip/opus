import type { ReactNode } from "react";

export type Tone =
  | "neutral"
  | "primary"
  | "accent"
  | "success"
  | "info"
  | "danger";

const toneClass: Record<Tone, string> = {
  neutral: "bg-surface-2 text-muted",
  primary: "bg-primary-soft text-primary-ink",
  accent: "bg-accent-soft text-accent-ink",
  success: "bg-success-soft text-success-ink",
  info: "bg-info-soft text-info-ink",
  danger: "bg-danger-soft text-danger-ink",
};

const TAG_TONES: Tone[] = ["primary", "accent", "success", "info", "danger"];

/** Stable color for a free-text tag. */
export function toneFor(text: string): Tone {
  let h = 0;
  for (const ch of text) h = (h * 31 + ch.charCodeAt(0)) >>> 0;
  return TAG_TONES[h % TAG_TONES.length];
}

export function Tag({
  tone = "neutral",
  children,
}: {
  tone?: Tone;
  children: ReactNode;
}) {
  return (
    <span
      className={`inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-xs font-medium ${toneClass[tone]}`}
    >
      {children}
    </span>
  );
}
