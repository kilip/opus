// NOTE: "@opus/shared" is an assumed workspace package name; adjust to yours.
import type { RunEvent, TokenUsage } from "@opus/shared";

/**
 * Parser for `agy -p <prompt> --output-format stream-json` (NDJSON on stdout).
 *
 * Built from real output. Observed envelope types:
 *   - init         conversation_id, init{cwd, tools, permission_mode}
 *   - step_update  step_update{step_index, state ACTIVE|DONE, step_type
 *                  user_input|agent_response|tool, text_delta, tool_name,
 *                  tool_info{parameters, output}, duration_seconds, usage}
 *   - result       result{status SUCCESS|ERROR, response, error, usage,
 *                  num_turns, finished_reason}
 *
 * Usage: feed stdout chunks to push() as they arrive (chunks may split lines),
 * call end() when the process exits.
 */

interface RawUsage {
  input_tokens?: number;
  output_tokens?: number;
  thinking_tokens?: number;
  cache_read_tokens?: number;
  total_tokens?: number;
}

interface RawStep {
  conversation_id?: string;
  step_index?: number;
  state?: string;
  step_type?: string;
  tool_name?: string;
  tool_info?: {
    name?: string;
    parameters?: Record<string, unknown>;
    output?: string;
  };
  text_delta?: string;
  duration_seconds?: number;
  usage?: RawUsage;
}

interface RawResult {
  conversation_id?: string;
  status?: string;
  response?: string;
  error?: string;
  duration_seconds?: number;
  num_turns?: number;
  usage?: RawUsage;
  finished_reason?: string;
}

interface RawEnvelope {
  event?: string;
  conversation_id?: string;
  init?: { cwd?: string; tools?: string[]; permission_mode?: string };
  step_update?: RawStep;
  result?: RawResult;
}

const emptyUsage = (): TokenUsage => ({
  inputTokens: 0,
  outputTokens: 0,
  thinkingTokens: 0,
  cacheReadTokens: 0,
  totalTokens: 0,
});

const toUsage = (raw?: RawUsage): TokenUsage => ({
  inputTokens: raw?.input_tokens ?? 0,
  outputTokens: raw?.output_tokens ?? 0,
  thinkingTokens: raw?.thinking_tokens ?? 0,
  cacheReadTokens: raw?.cache_read_tokens ?? 0,
  totalTokens: raw?.total_tokens ?? 0,
});

const toMs = (seconds?: number) => Math.round((seconds ?? 0) * 1000);

export class AntigravityStreamParser {
  private buffer = "";
  private text = "";
  private startedAt = Date.now();
  private finished = false;
  private usageSum = emptyUsage();
  private pendingTools = new Map<number, { tool: string; input: Record<string, unknown> }>();

  /** Feed a raw stdout chunk. Returns the events completed by this chunk. */
  push(chunk: string): RunEvent[] {
    this.buffer += chunk;
    const events: RunEvent[] = [];
    let nl = this.buffer.indexOf("\n");
    while (nl !== -1) {
      const line = this.buffer.slice(0, nl);
      this.buffer = this.buffer.slice(nl + 1);
      events.push(...this.parseLine(line));
      nl = this.buffer.indexOf("\n");
    }
    return events;
  }

  /**
   * Call when the process has exited. Flushes a trailing partial line and, if
   * the CLI never emitted a `result` event, synthesizes a run.finished so
   * consumers always get a terminal event.
   */
  end(opts: { cancelled?: boolean; exitCode?: number | null } = {}): RunEvent[] {
    const events: RunEvent[] = [];
    if (this.buffer.trim()) events.push(...this.parseLine(this.buffer));
    this.buffer = "";

    if (!this.finished) {
      this.finished = true;
      events.push({
        type: "run.finished",
        status: opts.cancelled ? "cancelled" : "error",
        response: this.text,
        error: opts.cancelled
          ? undefined
          : `agy exited without a result event (exit code ${opts.exitCode ?? "unknown"})`,
        usage: this.usageSum,
        durationMs: Date.now() - this.startedAt,
        turns: 0,
      });
    }
    return events;
  }

  private parseLine(line: string): RunEvent[] {
    const trimmed = line.trim();
    if (!trimmed) return [];

    let parsed: unknown;
    try {
      parsed = JSON.parse(trimmed);
    } catch {
      return [warn("non-JSON line on stdout", trimmed)];
    }
    if (typeof parsed !== "object" || parsed === null) {
      return [warn("unexpected JSON value on stdout", trimmed)];
    }

    const raw = parsed as RawEnvelope;
    switch (raw.event) {
      case "init":
        return this.onInit(raw);
      case "step_update":
        return raw.step_update ? this.onStep(raw.step_update) : [];
      case "result":
        return raw.result ? this.onResult(raw.result) : [];
      default:
        return [warn(`unknown event type: ${String(raw.event)}`, trimmed)];
    }
  }

  private onInit(raw: RawEnvelope): RunEvent[] {
    this.startedAt = Date.now();
    return [
      {
        type: "run.started",
        providerSessionId: raw.conversation_id ?? "",
        cwd: raw.init?.cwd ?? "",
        permissionMode: raw.init?.permission_mode ?? "unknown",
        availableTools: raw.init?.tools ?? [],
      },
    ];
  }

  private onStep(step: RawStep): RunEvent[] {
    const stepId = step.step_index ?? -1;
    const out: RunEvent[] = [];

    switch (step.step_type) {
      case "agent_response": {
        // Text arrives as deltas while ACTIVE, and the DONE event may carry a
        // last delta too. Intermediate agent_response steps (the model deciding
        // to call a tool) have no text at all, only usage.
        if (step.text_delta) {
          this.text += step.text_delta;
          out.push({ type: "message.delta", stepId, text: step.text_delta });
        }
        if (step.state === "DONE" && step.usage) {
          const usage = toUsage(step.usage);
          this.addUsage(usage);
          out.push({
            type: "step.usage",
            stepId,
            durationMs: toMs(step.duration_seconds),
            usage,
          });
        }
        break;
      }

      case "tool": {
        // tool_name here can differ from init.tools (e.g. "read_file" vs
        // "view_file"), so never validate against the init tool list.
        const tool = step.tool_name ?? step.tool_info?.name ?? "unknown";
        const input = step.tool_info?.parameters ?? {};
        if (step.state === "ACTIVE") {
          this.pendingTools.set(stepId, { tool, input });
          out.push({ type: "tool.started", stepId, tool, input });
        } else if (step.state === "DONE") {
          const started = this.pendingTools.get(stepId);
          this.pendingTools.delete(stepId);
          out.push({
            type: "tool.finished",
            stepId,
            tool,
            input: step.tool_info?.parameters ?? started?.input ?? {},
            output: step.tool_info?.output ?? "",
            durationMs: toMs(step.duration_seconds),
          });
        }
        break;
      }

      // user_input (the echoed prompt) and unknown step types: nothing to emit.
    }
    return out;
  }

  private onResult(res: RawResult): RunEvent[] {
    this.finished = true;

    const interrupted = res.finished_reason === "interrupted" || /cancel/i.test(res.error ?? "");
    const status = res.status === "SUCCESS" ? "success" : interrupted ? "cancelled" : "error";

    return [
      {
        type: "run.finished",
        status,
        // Prefer the CLI's own final text over our accumulated deltas.
        response: res.response || this.text,
        error: res.error || undefined,
        finishReason: res.finished_reason,
        usage: res.usage ? toUsage(res.usage) : this.usageSum,
        durationMs: toMs(res.duration_seconds),
        turns: res.num_turns ?? 0,
      },
    ];
  }

  private addUsage(u: TokenUsage) {
    this.usageSum = {
      inputTokens: this.usageSum.inputTokens + u.inputTokens,
      outputTokens: this.usageSum.outputTokens + u.outputTokens,
      thinkingTokens: this.usageSum.thinkingTokens + u.thinkingTokens,
      cacheReadTokens: this.usageSum.cacheReadTokens + u.cacheReadTokens,
      totalTokens: this.usageSum.totalTokens + u.totalTokens,
    };
  }
}

function warn(message: string, raw: string): RunEvent {
  return { type: "run.warning", message, raw: raw.slice(0, 500) };
}
