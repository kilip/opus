import assert from "node:assert";
import { describe, it } from "node:test";

import { buildStateTimeline, isActiveRunState } from "./model";

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
});
