import assert from "node:assert";
import { describe, it } from "node:test";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderToString } from "react-dom/server";
import { MemoryRouter } from "react-router";

import { runKeys } from "../api/keys";
import type { RunItem } from "../api/types";
import { SessionLogStream } from "../components/SessionLogStream";
import RunDetailPage from "./RunDetailPage";

function renderRunDetail(run: RunItem) {
	const runId = run.id ?? "run-1";
	const queryClient = new QueryClient({
		defaultOptions: {
			queries: {
				retry: false,
				retryOnMount: false,
				refetchOnMount: false,
			},
		},
	});
	queryClient.setQueryData(runKeys.detail(runId), run);

	return renderToString(
		<QueryClientProvider client={queryClient}>
			<MemoryRouter initialEntries={[`/runs/${runId}`]}>
				<RunDetailPage initialRunId={runId} />
			</MemoryRouter>
		</QueryClientProvider>,
	);
}

describe("RunDetailPage & SessionLogStream (AC1, AC2, AC3)", () => {
	it("renders tool_call session events with tool name, status, and duration (AC1)", () => {
		const html = renderToString(
			<SessionLogStream
				sessionId="sess-123"
				initialEntries={[
					{
						seq: 1,
						kind: "tool_call",
						payload: {
							tool: "get_task",
							status: "ok",
							duration_ms: 12,
						},
					},
					{
						seq: 2,
						kind: "tool_call",
						payload: {
							tool: "set_status",
							status: "error",
							error: "invalid_status",
							duration_ms: 4,
						},
					},
				]}
			/>,
		);

		assert.ok(html.includes("[tool_call]"));
		assert.ok(html.includes("get_task"));
		assert.ok(html.includes("ok"));
		assert.ok(html.includes("12ms"));
		assert.ok(html.includes("set_status"));
		assert.ok(html.includes("error"));
		assert.ok(html.includes("invalid_status"));
		assert.ok(html.includes("4ms"));
	});

	it("renders populated Run.report with summary, PR URL, commit SHA, assumptions, and criteria (AC2)", () => {
		const html = renderRunDetail({
			id: "run-1",
			taskId: "task-1",
			projectId: "proj-1",
			mode: "vibe",
			state: "merged",
			stageAttempts: 1,
			createdAt: "2026-11-18T00:00:00Z",
			updatedAt: "2026-11-18T00:10:00Z",
			report: {
				summary: "Added agent report and tool_call UI",
				pr_url: "https://github.com/kilip/opus/pull/137",
				commit_sha: "deadbeef",
				assumptions: ["Existing SSE endpoint emits tool_call events"],
				acceptance_criteria: [
					{
						id: "AC1",
						criterion: "tool_call events rendered",
						met: true,
						evidence: "bun test passed",
					},
					{
						id: "AC2",
						criterion: "manual verification",
						met: false,
						evidence: "pending review",
					},
				],
			} as unknown as RunItem["report"],
		});

		assert.ok(html.includes("Agent Report"));
		assert.ok(html.includes("Added agent report and tool_call UI"));
		assert.ok(html.includes("https://github.com/kilip/opus/pull/137"));
		assert.ok(html.includes("deadbeef"));
		assert.ok(html.includes("Existing SSE endpoint emits tool_call events"));
		assert.ok(html.includes("AC1: tool_call events rendered"));
		assert.ok(html.includes("Met"));
		assert.ok(html.includes("bun test passed"));
		assert.ok(html.includes("Unmet"));
		assert.ok(html.includes("pending review"));
	});

	it("renders clear empty state when Run.report is null/absent (AC2)", () => {
		const html = renderRunDetail({
			id: "run-2",
			taskId: "task-2",
			projectId: "proj-1",
			mode: "vibe",
			state: "running",
			stageAttempts: 1,
			createdAt: "2026-11-18T00:00:00Z",
			updatedAt: "2026-11-18T00:05:00Z",
		});

		assert.ok(html.includes("Agent Report"));
		assert.ok(html.includes("No report submitted for this run yet."));
	});
});
