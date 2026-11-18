import assert from "node:assert";
import { describe, it } from "node:test";

import { summarizeDashboard } from "./model";

describe("dashboard model", () => {
  it("summarizes online nodes, task counts, active runs, and recent outcomes", () => {
    const summary = summarizeDashboard(
      [
        { id: "n1", name: "alpha", role: "worker", status: "online" },
        { id: "n2", name: "beta", role: "worker", status: "offline" },
      ],
      [
        {
          id: "t1",
          key: "OP-1",
          projectId: "p1",
          source: "github",
          title: "Task 1",
          description: "desc",
          status: "todo",
          priority: "1",
          type: "feat",
          labels: [],
          createdAt: "2026-11-18T00:00:00Z",
          updatedAt: "2026-11-18T00:00:00Z",
        },
        {
          id: "t2",
          key: "OP-2",
          projectId: "p1",
          source: "github",
          title: "Task 2",
          description: "desc",
          status: "done",
          priority: "1",
          type: "feat",
          labels: [],
          createdAt: "2026-11-18T00:00:00Z",
          updatedAt: "2026-11-18T00:00:00Z",
        },
      ],
      [
        { id: "r1", state: "running", mode: "vibe" },
        {
          id: "r2",
          state: "merged",
          mode: "vibe",
          endedAt: "2026-11-18T01:00:00Z",
        },
        {
          id: "r3",
          state: "failed",
          mode: "vibe",
          endedAt: "2026-11-18T02:00:00Z",
        },
      ],
    );

    assert.strictEqual(summary.onlineNodes.length, 1);
    assert.strictEqual(summary.totalNodes, 2);
    assert.strictEqual(summary.taskCounts.todo, 1);
    assert.strictEqual(summary.taskCounts.done, 1);
    assert.strictEqual(summary.activeRuns.length, 1);
    assert.strictEqual(summary.recentOutcomes[0]?.id, "r3");
    assert.strictEqual(summary.recentOutcomes[1]?.id, "r2");
  });
});
