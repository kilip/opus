import assert from "node:assert";
import { describe, it } from "node:test";

import {
	buildStateTimeline,
	isActiveRunState,
	parseRunReport,
	parseToolCallPayload,
} from "./model";

describe("runs model", () => {
	it("identifies active run states", () => {
		assert.strictEqual(isActiveRunState("queued"), true);
		assert.strictEqual(isActiveRunState("preparing"), true);
		assert.strictEqual(isActiveRunState("running"), true);
		assert.strictEqual(isActiveRunState("paused"), true);
		assert.strictEqual(isActiveRunState("merged"), false);
		assert.strictEqual(isActiveRunState("failed"), false);
		assert.strictEqual(isActiveRunState("needs-info"), false);
		assert.strictEqual(isActiveRunState("cancelled"), false);
	});

	it("builds happy-path timeline for running", () => {
		const steps = buildStateTimeline({ state: "running" });
		assert.deepStrictEqual(
			steps.map((s) => `${s.state}:${s.status}`),
			["queued:done", "preparing:done", "running:current", "merged:pending"],
		);
	});

	it("builds terminal outcome timeline for failed and paused", () => {
		const failed = buildStateTimeline({ state: "failed" });
		assert.deepStrictEqual(failed[3], {
			state: "failed",
			label: "failed",
			status: "failed",
		});

		const paused = buildStateTimeline({ state: "paused" });
		assert.deepStrictEqual(paused[3], {
			state: "paused",
			label: "paused",
			status: "current",
		});
	});

	it("parses tool_call payloads and rejects invalid payloads", () => {
		const ok = parseToolCallPayload({
			tool: "get_task",
			status: "ok",
			duration_ms: 14,
		});
		assert.deepStrictEqual(ok, {
			tool: "get_task",
			args: undefined,
			status: "ok",
			error: undefined,
			duration_ms: 14,
		});

		const err = parseToolCallPayload(
			JSON.stringify({
				tool: "set_status",
				status: "error",
				error: "invalid_status",
				duration_ms: 3,
			}),
		);
		assert.strictEqual(err?.tool, "set_status");
		assert.strictEqual(err?.status, "error");
		assert.strictEqual(err?.error, "invalid_status");
		assert.strictEqual(err?.duration_ms, 3);

		assert.strictEqual(parseToolCallPayload(null), null);
		assert.strictEqual(parseToolCallPayload({ status: "ok" }), null);
	});

	it("parses Run.report objects and returns null for empty reports", () => {
		assert.strictEqual(parseRunReport(null), null);
		assert.strictEqual(parseRunReport({}), null);

		const parsed = parseRunReport({
			summary: "Implemented MCP tool service",
			pr_url: "https://github.com/kilip/opus/pull/137",
			commit_sha: "abc1234",
			assumptions: ["Clean tree"],
			acceptance_criteria: [
				{
					id: "AC1",
					criterion: "Tool calls logged",
					met: true,
					evidence: "ok",
				},
			],
		});
		assert.strictEqual(parsed?.summary, "Implemented MCP tool service");
		assert.strictEqual(
			parsed?.pr_url,
			"https://github.com/kilip/opus/pull/137",
		);
		assert.strictEqual(parsed?.commit_sha, "abc1234");
		assert.deepStrictEqual(parsed?.assumptions, ["Clean tree"]);
		assert.strictEqual(parsed?.acceptance_criteria?.[0].met, true);
	});
});
