import assert from "node:assert";
import { describe, it } from "node:test";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createElement } from "react";
import { renderToString } from "react-dom/server";
import { MemoryRouter } from "react-router";

import { taskKeys } from "../api/keys";
import type { Task } from "../api/types";
import TasksPage from "./TasksPage";

const mockTask: Task = {
	id: "task-1",
	key: "OP-1",
	title: "Test task",
	description: "Test description",
	status: "todo",
	priority: "1",
	source: "manual",
	type: "feat",
	labels: [],
	createdAt: "2026-01-01T00:00:00Z",
	updatedAt: "2026-01-01T00:00:00Z",
};

function renderTasksPage({
	tasks,
	url = "/tasks",
	queryState,
}: {
	tasks?: Task[];
	url?: string;
	queryState?: { status: "error"; error: Error };
} = {}) {
	const queryClient = new QueryClient({
		defaultOptions: {
			queries: {
				retry: false,
				retryOnMount: false,
				refetchOnMount: false,
			},
		},
	});

	if (queryState) {
		queryClient
			.getQueryCache()
			.build(queryClient, { queryKey: taskKeys.list() })
			.setState({
				status: queryState.status,
				error: queryState.error,
				errorUpdatedAt: Date.now(),
				fetchStatus: "idle",
			});
	} else if (tasks !== undefined) {
		queryClient.setQueryData(taskKeys.list(), tasks);
	}

	return renderToString(
		createElement(
			QueryClientProvider,
			{ client: queryClient },
			createElement(
				MemoryRouter,
				{ initialEntries: [url] },
				createElement(TasksPage),
			),
		),
	);
}

describe("TasksPage (AC1, AC2, AC3, AC4)", () => {
	it("shows empty state with heading and create task link in kanban view (AC1)", () => {
		const html = renderTasksPage({ tasks: [], url: "/tasks" });

		assert.match(html, /<h2[^>]*>No tasks yet<\/h2>/);
		assert.match(html, /href="\/tasks\/new"[^>]*>Create task<\/a>/);
	});

	it("shows empty state with heading and create task link in list view (AC2)", () => {
		const html = renderTasksPage({ tasks: [], url: "/tasks?view=list" });

		assert.match(html, /<h2[^>]*>No tasks yet<\/h2>/);
		assert.match(html, /href="\/tasks\/new"[^>]*>Create task<\/a>/);
	});

	it("does not show empty state when at least one task exists in kanban view (AC3)", () => {
		const html = renderTasksPage({ tasks: [mockTask], url: "/tasks" });

		assert.doesNotMatch(html, /No tasks yet/);
		assert.match(html, /Test task/);
	});

	it("does not show empty state when at least one task exists in list view (AC3)", () => {
		const html = renderTasksPage({
			tasks: [mockTask],
			url: "/tasks?view=list",
		});

		assert.doesNotMatch(html, /No tasks yet/);
		assert.match(html, /Test task/);
	});

	it("renders loading state when query is pending (AC4)", () => {
		const html = renderTasksPage();

		assert.match(html, /Loading tasks…/);
		assert.doesNotMatch(html, /No tasks yet/);
	});

	it("renders error state when query fails (AC4)", () => {
		const html = renderTasksPage({
			queryState: {
				status: "error",
				error: new Error("Failed to load tasks"),
			},
		});

		assert.match(html, /Could not load tasks/);
		assert.match(html, /Failed to load tasks/);
		assert.doesNotMatch(html, /No tasks yet/);
	});
});
