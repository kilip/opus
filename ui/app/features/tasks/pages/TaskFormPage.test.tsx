import assert from "node:assert";
import { describe, it } from "node:test";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createElement } from "react";
import { renderToString } from "react-dom/server";
import { MemoryRouter, Route, Routes } from "react-router";

import { taskKeys } from "../api/keys";
import type { Project, Task, TrackerPR } from "../api/types";
import { TaskList } from "../components/TaskList";
import TaskFormPage from "./TaskFormPage";

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

const mockTask: Task = {
	id: "task-1",
	key: "OP-1",
	projectId: "proj-1",
	title: "Existing Task",
	description: "Existing Description",
	status: "todo",
	priority: "1",
	source: "manual",
	type: "feat",
	labels: [],
	createdAt: "2026-01-01T00:00:00Z",
	updatedAt: "2026-01-01T00:00:00Z",
};

function renderTaskFormPage({
	taskId = "new",
	task,
	projects = mockProjects,
	prs,
}: {
	taskId?: string;
	task?: Task;
	projects?: Project[];
	prs?: TrackerPR[];
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
	if (task !== undefined) {
		queryClient.setQueryData(taskKeys.detail(task.id), task);
	}
	if (task !== undefined && prs !== undefined) {
		queryClient.setQueryData(taskKeys.prs(task.id), prs);
	}

	return renderToString(
		createElement(
			QueryClientProvider,
			{ client: queryClient },
			createElement(
				MemoryRouter,
				{ initialEntries: [`/tasks/${taskId}`] },
				createElement(
					Routes,
					null,
					createElement(Route, {
						path: "/tasks/:taskId",
						element: createElement(TaskFormPage),
					}),
				),
			),
		),
	);
}

describe("TaskFormPage Project Picker (AC6)", () => {
	it("renders required project picker with projects from query on create form", () => {
		const html = renderTaskFormPage({ taskId: "new" });

		assert.match(html, /<select[^>]*required/);
		assert.match(html, /<option[^>]*value=""[^>]*>Select project…<\/option>/);
		assert.match(
			html,
			/<option[^>]*value="proj-1"[^>]*>Project Alpha \(project-alpha\)<\/option>/,
		);
		assert.match(
			html,
			/<option[^>]*value="proj-2"[^>]*>Project Beta \(project-beta\)<\/option>/,
		);
	});

	it("disables project select when editing an existing task", () => {
		const html = renderTaskFormPage({
			taskId: "task-1",
			task: mockTask,
		});

		assert.match(html, /<select[^>]*disabled/);
		assert.match(html, /value="proj-1"/);
	});
});

describe("Task GitHub Context & Linked PRs (AC1, AC2, AC4)", () => {
	const ghTask: Task = {
		...mockTask,
		id: "task-gh",
		key: "OP-104",
		source: "github",
		externalId: "kilip/opus#104",
		externalUrl: "https://github.com/kilip/opus/issues/104",
		sourceStatus: "open",
	};

	const mockPRs: TrackerPR[] = [
		{
			id: "pr-1",
			projectId: "proj-1",
			taskId: "task-gh",
			number: 105,
			state: "open",
			isDraft: true,
			reviewState: "REVIEW_REQUIRED",
			checksSummary: "pending",
			url: "https://github.com/kilip/opus/pull/105",
			createdAt: "2026-11-17T01:00:00Z",
			updatedAt: "2026-11-17T02:00:00Z",
		},
	];

	it("renders external link, sourceStatus, and linked PRs for github task", () => {
		const html = renderTaskFormPage({
			taskId: "task-gh",
			task: ghTask,
			prs: mockPRs,
		});

		assert.match(
			html,
			/href="https:\/\/github\.com\/kilip\/opus\/issues\/104"[^>]*target="_blank"[^>]*rel="noopener noreferrer"/,
		);
		assert.match(html, />open</);
		assert.match(html, /Pull Requests/);
		assert.match(
			html,
			/href="https:\/\/github\.com\/kilip\/opus\/pull\/105"[^>]*target="_blank"[^>]*rel="noopener noreferrer"[^>]*>#(?:<!-- -->)?105<\/a>/,
		);
		assert.match(html, />draft</);
		assert.match(html, />REVIEW_REQUIRED</);
		assert.match(html, />checks: (?:<!-- -->)?pending</);
	});

	it("omits external link and PR section for task without source", () => {
		const html = renderTaskFormPage({
			taskId: "task-1",
			task: mockTask,
		});

		assert.doesNotMatch(html, /rel="noopener noreferrer"/);
		assert.doesNotMatch(html, /Pull Requests/);
	});

	it("renders external link and sourceStatus in TaskList for sourced tasks only", () => {
		const html = renderToString(
			createElement(
				MemoryRouter,
				null,
				createElement(TaskList, { tasks: [ghTask, mockTask] }),
			),
		);

		assert.match(
			html,
			/href="https:\/\/github\.com\/kilip\/opus\/issues\/104"[^>]*target="_blank"[^>]*rel="noopener noreferrer"[^>]*>kilip\/opus#104<\/a>/,
		);
		assert.match(html, />open</);
	});
});
