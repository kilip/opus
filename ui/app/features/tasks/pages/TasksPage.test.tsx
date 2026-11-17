import assert from "node:assert";
import { describe, it } from "node:test";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createElement } from "react";
import { renderToString } from "react-dom/server";
import { MemoryRouter } from "react-router";

import { taskKeys } from "../api/keys";
import type { Project, Task } from "../api/types";
import { filterTasks, STAGES, TASK_FILTERS } from "../model";
import TasksPage from "./TasksPage";

const mockTask: Task = {
	id: "task-1",
	key: "OP-1",
	projectId: "proj-1",
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

const mockProjects: Project[] = [
	{
		id: "proj-1",
		slug: "project-alpha",
		name: "Project Alpha",
		repo_provider: "github",
		repo_ref: "org/alpha",
		tracker_provider: "github",
		tracker_ref: "org/alpha",
		default_branch: "main",
		created_at: "2026-01-01T00:00:00Z",
		updated_at: "2026-01-01T00:00:00Z",
	},
	{
		id: "proj-2",
		slug: "project-beta",
		name: "Project Beta",
		repo_provider: "github",
		repo_ref: "org/beta",
		tracker_provider: "github",
		tracker_ref: "org/beta",
		default_branch: "main",
		created_at: "2026-01-01T00:00:00Z",
		updated_at: "2026-01-01T00:00:00Z",
	},
];

function renderTasksPage({
	tasks,
	projects = mockProjects,
	url = "/tasks",
	queryState,
}: {
	tasks?: Task[];
	projects?: Project[];
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

	if (projects !== undefined) {
		queryClient.setQueryData(taskKeys.projects(), projects);
	}

	const urlObj = new URL(url, "http://localhost");
	const projectId = urlObj.searchParams.get("projectId") ?? undefined;

	if (queryState) {
		queryClient
			.getQueryCache()
			.build(queryClient, { queryKey: taskKeys.list({ projectId }) })
			.setState({
				status: queryState.status,
				error: queryState.error,
				errorUpdatedAt: Date.now(),
				fetchStatus: "idle",
			});
	} else if (tasks !== undefined) {
		queryClient.setQueryData(taskKeys.list({ projectId }), tasks);
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

describe("TasksPage (AC1, AC2, AC3, AC4, AC6)", () => {
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

	it("renders project filter options and filters tasks (AC6)", () => {
		const html = renderTasksPage({
			tasks: [mockTask],
			projects: mockProjects,
			url: "/tasks?projectId=proj-1",
		});

		assert.match(html, /aria-label="Filter by project"/);
		assert.match(
			html,
			/<option[^>]*value="proj-1"[^>]*>Project Alpha<\/option>/,
		);
		assert.match(
			html,
			/<option[^>]*value="proj-2"[^>]*>Project Beta<\/option>/,
		);
		assert.match(html, /Test task/);
	});

	it("renders blocked stage column in kanban, badge in list, and filters blocked tasks (AC7)", () => {
		const blockedTask: Task = {
			...mockTask,
			id: "task-blocked",
			key: "OP-2",
			title: "Blocked task",
			status: "blocked",
		};

		assert.ok(STAGES.some((s) => s.id === "blocked" && s.tone === "danger"));
		assert.ok(TASK_FILTERS.some((f) => f.id === "blocked"));

		const kanbanHtml = renderTasksPage({
			tasks: [mockTask, blockedTask],
			url: "/tasks",
		});
		assert.match(kanbanHtml, /aria-label="Blocked"/);
		assert.match(kanbanHtml, /Blocked task/);

		const listHtml = renderTasksPage({
			tasks: [mockTask, blockedTask],
			url: "/tasks?view=list",
		});
		assert.match(listHtml, /Blocked<\/span>/);

		const filtered = filterTasks(
			[mockTask, blockedTask],
			"",
			new Set(["blocked"]),
		);
		assert.deepStrictEqual(
			filtered.map((t) => t.id),
			["task-blocked"],
		);
	});
});
