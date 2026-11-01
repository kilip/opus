/**
 * Provider-agnostic run events.
 *
 * Every provider adapter (antigravity, claude-code, ...) translates its raw CLI
 * output into these events. The UI, the Telegram channel and storage only ever
 * see RunEvent, never provider-specific JSON.
 */

export interface TokenUsage {
  inputTokens: number;
  outputTokens: number;
  thinkingTokens: number;
  cacheReadTokens: number;
  totalTokens: number;
}

export type RunStatus = "success" | "error" | "cancelled";

export type RunEvent =
  | {
      type: "run.started";
      /** Provider-side conversation id. Needed to resume a conversation later. */
      providerSessionId: string;
      cwd: string;
      /** e.g. "always-proceed": the CLI will run tools without asking. */
      permissionMode: string;
      availableTools: string[];
    }
  /** Append-only chunk of the assistant's visible answer. */
  | { type: "message.delta"; stepId: number; text: string }
  /** Token usage for one model call. Several of these happen per run. */
  | { type: "step.usage"; stepId: number; durationMs: number; usage: TokenUsage }
  | {
      type: "tool.started";
      stepId: number;
      tool: string;
      input: Record<string, unknown>;
    }
  | {
      type: "tool.finished";
      stepId: number;
      tool: string;
      input: Record<string, unknown>;
      /** Raw tool output text. Not structured by the provider. */
      output: string;
      durationMs: number;
    }
  | {
      type: "run.finished";
      status: RunStatus;
      /** Full final answer; source of truth over the accumulated deltas. */
      response: string;
      error?: string;
      finishReason?: string;
      usage: TokenUsage;
      durationMs: number;
      turns: number;
    }
  /** Something the parser could not understand. Never fatal. */
  | { type: "run.warning"; message: string; raw?: string };
