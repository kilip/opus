import { describe, expect, test } from "bun:test";
import { readFileSync } from "node:fs";
import { join } from "node:path";
import type { RunEvent } from "@opus/shared";
import { AntigravityStreamParser } from "./stream-parser";

const fixture = (name: string) => readFileSync(join(import.meta.dir, "__fixtures__", name), "utf8");

function parseAll(raw: string, chunkSize = raw.length): RunEvent[] {
  const parser = new AntigravityStreamParser();
  const events: RunEvent[] = [];
  for (let i = 0; i < raw.length; i += chunkSize) {
    events.push(...parser.push(raw.slice(i, i + chunkSize)));
  }
  events.push(...parser.end());
  return events;
}

const ofType = <T extends RunEvent["type"]>(events: RunEvent[], type: T) =>
  events.filter((e): e is Extract<RunEvent, { type: T }> => e.type === type);

describe("simple-run fixture", () => {
  const events = parseAll(fixture("simple-run.jsonl"));

  test("starts with run.started carrying session, cwd and permission mode", () => {
    const first = events[0];
    expect(first?.type).toBe("run.started");
    if (first?.type !== "run.started") return;
    expect(first.providerSessionId).toBe("cc499092-94ad-4832-a770-20312be68108");
    expect(first.cwd).toBe("/home/toni/code/opus");
    expect(first.permissionMode).toBe("always-proceed");
    expect(first.availableTools).toContain("run_command");
  });

  test("pairs every tool.started with a tool.finished of the same step", () => {
    const started = ofType(events, "tool.started");
    const finished = ofType(events, "tool.finished");
    expect(started.map((e) => e.tool)).toEqual([
      "read_file",
      "run_command",
      "run_command",
      "run_command",
    ]);
    expect(finished.map((e) => e.stepId)).toEqual(started.map((e) => e.stepId));
    expect(finished[1]?.output).toContain("LISTEN");
    expect(finished[1]?.input).toEqual({
      CommandLine: "ss -tulpn | grep 9221 || lsof -i :9221 || true",
    });
  });

  test("streamed deltas add up to the CLI's final response", () => {
    const joined = ofType(events, "message.delta")
      .map((e) => e.text)
      .join("");
    const last = events.at(-1);
    expect(last?.type).toBe("run.finished");
    if (last?.type !== "run.finished") return;
    expect(last.status).toBe("success");
    expect(joined).toBe(last.response);
    expect(last.turns).toBe(1);
  });

  test("per-step usage sums to the aggregate usage in result", () => {
    const sum = ofType(events, "step.usage").reduce((acc, e) => acc + e.usage.totalTokens, 0);
    const last = events.at(-1);
    if (last?.type !== "run.finished") throw new Error("no run.finished");
    expect(sum).toBe(last.usage.totalTokens);
    expect(sum).toBe(64854);
  });

  test("produces identical events regardless of chunk boundaries", () => {
    const raw = fixture("simple-run.jsonl");
    expect(parseAll(raw, 7)).toEqual(events);
    expect(parseAll(raw, 1)).toEqual(events);
  });
});

describe("cancelled-run fixture", () => {
  test("an interrupted result maps to status cancelled, not error", () => {
    const events = parseAll(fixture("cancelled-run.jsonl"));
    expect(events).toHaveLength(1);
    const e = events[0];
    if (e?.type !== "run.finished") throw new Error("no run.finished");
    expect(e.status).toBe("cancelled");
    expect(e.finishReason).toBe("interrupted");
  });
});

describe("robustness", () => {
  test("non-JSON lines become warnings instead of throwing", () => {
    const parser = new AntigravityStreamParser();
    const events = parser.push('❯ agy -p "hi"\nnull\n');
    expect(events.map((e) => e.type)).toEqual(["run.warning", "run.warning"]);
  });

  test("unknown event types become warnings", () => {
    const parser = new AntigravityStreamParser();
    const events = parser.push('{"event":"something_new"}\n');
    expect(events[0]?.type).toBe("run.warning");
  });

  test("end() synthesizes an error when the CLI never sent a result", () => {
    const parser = new AntigravityStreamParser();
    const events = parser.end({ exitCode: 1 });
    const e = events[0];
    if (e?.type !== "run.finished") throw new Error("no run.finished");
    expect(e.status).toBe("error");
    expect(e.error).toContain("exit code 1");
  });

  test("end({ cancelled: true }) synthesizes a cancelled run", () => {
    const parser = new AntigravityStreamParser();
    const e = parser.end({ cancelled: true })[0];
    if (e?.type !== "run.finished") throw new Error("no run.finished");
    expect(e.status).toBe("cancelled");
  });

  test("a final line without trailing newline is still parsed", () => {
    const raw = fixture("cancelled-run.jsonl").trimEnd();
    const events = parseAll(raw);
    expect(events[0]?.type).toBe("run.finished");
  });
});
