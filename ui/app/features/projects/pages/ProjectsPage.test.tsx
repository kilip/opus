import assert from "node:assert";
import { describe, it } from "node:test";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderToString } from "react-dom/server";
import { MemoryRouter } from "react-router";

import { apps } from "~/apps";
import { ApiError, api, unwrap } from "~/shared/api/client";
import { PROJECT_ERRORS } from "../api/errors";
import { projectKeys } from "../api/keys";
import type {
	CreateProjectRequest,
	Project,
	ProjectNodeResponse,
	UpdateProjectNodePathRequest,
	UpdateProjectRequest,
} from "../api/types";
import { projectsApp } from "../app";
import { EditProjectModal } from "../components/EditProjectModal";
import { projectsRoutes } from "../routes";
import ProjectsPage from "./ProjectsPage";

function renderWithClient(
	ui: React.ReactElement,
	{
		projects,
		queryState,
		nodeMappings,
	}: {
		projects?: Project[];
		queryState?: { status: "error"; error: Error };
		nodeMappings?: { projectId: string; mappings: ProjectNodeResponse[] };
	} = {},
) {
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
			.build(queryClient, { queryKey: projectKeys.list() })
			.setState({
				status: queryState.status,
				error: queryState.error,
				errorUpdatedAt: Date.now(),
				fetchStatus: "idle",
			});
	} else if (projects !== undefined) {
		queryClient.setQueryData(projectKeys.list(), projects);
	}

	if (nodeMappings) {
		queryClient.setQueryData(
			projectKeys.nodes(nodeMappings.projectId),
			nodeMappings.mappings,
		);
	}

	return {
		queryClient,
		html: renderToString(
			<QueryClientProvider client={queryClient}>
				<MemoryRouter>{ui}</MemoryRouter>
			</QueryClientProvider>,
		),
	};
}

describe("Projects Feature (AC1, AC2, AC3, AC4, AC5)", () => {
	it("registers route /projects and configures apps switcher entry (AC1)", () => {
		assert.strictEqual(projectsRoutes.length, 1);
		assert.strictEqual(projectsRoutes[0].path, "projects");
		assert.strictEqual(
			projectsRoutes[0].file,
			"features/projects/pages/ProjectsPage.tsx",
		);

		assert.strictEqual(projectsApp.id, "projects");
		assert.strictEqual(projectsApp.name, "Projects");
		assert.strictEqual(projectsApp.path, "/projects");
		assert.strictEqual(projectsApp.icon, "folder");
		assert.strictEqual(projectsApp.enabled, true);

		const registeredInApps = apps.find((a) => a.id === "projects");
		assert.ok(registeredInApps);
		assert.strictEqual(registeredInApps?.enabled, true);
	});

	it("renders loading, error and empty states for projects list (AC1)", () => {
		// Loading state
		const loading = renderWithClient(<ProjectsPage />);
		assert.match(loading.html, /Loading projects…/);

		// Error state
		const error = renderWithClient(<ProjectsPage />, {
			queryState: {
				status: "error",
				error: new Error("Failed to load projects from server"),
			},
		});
		assert.match(error.html, /Could not load projects/);
		assert.match(error.html, /Failed to load projects from server/);
		assert.match(error.html, />Retry<\/button>/);

		// Empty state
		const empty = renderWithClient(<ProjectsPage />, { projects: [] });
		assert.match(empty.html, /No projects yet/);
		assert.match(empty.html, /Create Project<\/button>/);
	});

	it("lists projects with slug, name, repo and default branch (AC1)", () => {
		const mockProjects: Project[] = [
			{
				id: "01933a3c-0001-7000-8000-000000000001",
				slug: "opus-web",
				name: "Opus Web",
				repo_provider: "github",
				repo_ref: "kilip/opus",
				default_branch: "main",
			},
			{
				id: "01933a3c-0002-7000-8000-000000000002",
				slug: "agent-runner",
				name: "Agent Runner",
				repo_provider: "github",
				repo_ref: "kilip/runner",
				default_branch: "develop",
			},
		];

		const { html } = renderWithClient(<ProjectsPage />, {
			projects: mockProjects,
		});

		// Name rendered
		assert.match(html, /Opus Web/);
		assert.match(html, /Agent Runner/);

		// Slug rendered
		assert.match(html, /opus-web/);
		assert.match(html, /agent-runner/);

		// Repo ref rendered
		assert.match(html, /kilip\/opus/);
		assert.match(html, /kilip\/runner/);

		// Default branch rendered
		assert.match(html, /main/);
		assert.match(html, /develop/);

		// Edit button per project rendered
		const editMatches = html.match(/>Edit<\/button>/g);
		assert.strictEqual(editMatches?.length, 2);
	});

	it("Create form submits slug, name, owner/name repo, default branch, fixed github provider (AC2)", async () => {
		const originalPost = api.POST;
		let postedBody: CreateProjectRequest | undefined;

		try {
			api.POST = (async (
				path: string,
				options: { body: CreateProjectRequest },
			) => {
				if (path === "/projects") {
					postedBody = options.body;
					return {
						data: {
							data: {
								id: "01933a3c-new-project-0001",
								...options.body,
							},
						},
						response: new Response(JSON.stringify({}), { status: 201 }),
					};
				}
				return originalPost(path as never, options as never);
			}) as never;

			const queryClient = new QueryClient();
			const mutation = queryClient.getMutationCache().build(queryClient, {
				mutationFn: async (req: CreateProjectRequest) =>
					unwrap<Project>(api.POST("/projects", { body: req }), PROJECT_ERRORS),
			});

			const payload: CreateProjectRequest = {
				slug: "new-proj",
				name: "New Project",
				repo_provider: "github",
				repo_ref: "owner/repo",
				default_branch: "main",
			};

			const result = (await mutation.execute(payload)) as Project;
			assert.strictEqual(result.id, "01933a3c-new-project-0001");
			assert.strictEqual(postedBody?.slug, "new-proj");
			assert.strictEqual(postedBody?.name, "New Project");
			assert.strictEqual(postedBody?.repo_provider, "github");
			assert.strictEqual(postedBody?.repo_ref, "owner/repo");
			assert.strictEqual(postedBody?.default_branch, "main");
		} finally {
			api.POST = originalPost;
		}
	});

	it("Create form maps API validation errors (400, 409) using PROJECT_ERRORS (AC2)", async () => {
		const originalPost = api.POST;

		try {
			// Test 409 PROJECT_DUPLICATE_SLUG
			api.POST = (async () => ({
				error: { error: "PROJECT_DUPLICATE_SLUG" },
				response: new Response(
					JSON.stringify({ error: "PROJECT_DUPLICATE_SLUG" }),
					{ status: 409 },
				),
			})) as never;

			const queryClient = new QueryClient();
			const mutation409 = queryClient.getMutationCache().build(queryClient, {
				mutationFn: async (req: CreateProjectRequest) =>
					unwrap<Project>(api.POST("/projects", { body: req }), PROJECT_ERRORS),
			});

			await assert.rejects(
				async () => {
					await mutation409.execute({
						slug: "existing-slug",
						name: "Dup",
						repo_provider: "github",
						repo_ref: "owner/dup",
					});
				},
				(err: unknown) => {
					assert.ok(err instanceof ApiError);
					assert.strictEqual(err.status, 409);
					assert.strictEqual(err.code, "PROJECT_DUPLICATE_SLUG");
					assert.strictEqual(
						err.message,
						PROJECT_ERRORS.PROJECT_DUPLICATE_SLUG,
					);
					return true;
				},
			);

			// Test 400 PROJECT_INVALID_SLUG
			api.POST = (async () => ({
				error: { error: "PROJECT_INVALID_SLUG" },
				response: new Response(
					JSON.stringify({ error: "PROJECT_INVALID_SLUG" }),
					{ status: 400 },
				),
			})) as never;

			const mutation400 = queryClient.getMutationCache().build(queryClient, {
				mutationFn: async (req: CreateProjectRequest) =>
					unwrap<Project>(api.POST("/projects", { body: req }), PROJECT_ERRORS),
			});

			await assert.rejects(
				async () => {
					await mutation400.execute({
						slug: "INVALID_SLUG!",
						name: "Bad Slug",
						repo_provider: "github",
						repo_ref: "owner/repo",
					});
				},
				(err: unknown) => {
					assert.ok(err instanceof ApiError);
					assert.strictEqual(err.status, 400);
					assert.strictEqual(err.code, "PROJECT_INVALID_SLUG");
					assert.strictEqual(err.message, PROJECT_ERRORS.PROJECT_INVALID_SLUG);
					return true;
				},
			);
		} finally {
			api.POST = originalPost;
		}
	});

	it("Edit modal renders slug and repo as read-only / disabled (AC3)", () => {
		const mockProject: Project = {
			id: "project-1",
			slug: "readonly-slug",
			name: "Editable Name",
			repo_provider: "github",
			repo_ref: "owner/readonly-repo",
			default_branch: "main",
			tracker_provider: "github",
			tracker_ref: "owner/readonly-repo",
		};

		const { html } = renderWithClient(
			<EditProjectModal
				project={mockProject}
				isOpen={true}
				onClose={() => {}}
			/>,
			{
				nodeMappings: { projectId: "project-1", mappings: [] },
			},
		);

		// Assert slug is rendered disabled and readonly
		assert.match(
			html,
			/<input[^>]*id="edit-project-slug"[^>]*readOnly[^>]*disabled/,
		);
		assert.match(html, /value="readonly-slug"/);

		// Assert repo is rendered disabled and readonly
		assert.match(
			html,
			/<input[^>]*id="edit-project-repo"[^>]*readOnly[^>]*disabled/,
		);
		assert.match(html, /value="owner\/readonly-repo"/);

		// Assert name, default branch, tracker fields are present
		assert.match(html, /id="edit-project-name"/);
		assert.match(html, /id="edit-project-branch"/);
		assert.match(html, /id="edit-project-tracker-provider"/);
		assert.match(html, /id="edit-project-tracker-ref"/);

		// Assert Save Changes button is present
		assert.match(html, />Save Changes<\/button>/);
	});

	it("Edit form calls update endpoint with editable fields without slug or repo_ref (AC3)", async () => {
		const originalPatch = api.PATCH;
		let patchedBody: UpdateProjectRequest | undefined;
		let patchedId: string | undefined;

		try {
			api.PATCH = (async (
				path: string,
				options: {
					params: { path: { id: string } };
					body: UpdateProjectRequest;
				},
			) => {
				if (path === "/projects/{id}") {
					patchedId = options.params.path.id;
					patchedBody = options.body;
					return {
						data: {
							data: {
								id: patchedId,
								slug: "original-slug",
								...options.body,
							},
						},
						response: new Response(JSON.stringify({}), { status: 200 }),
					};
				}
				return originalPatch(path as never, options as never);
			}) as never;

			const queryClient = new QueryClient();
			const mutation = queryClient.getMutationCache().build(queryClient, {
				mutationFn: async ({
					id,
					...patch
				}: UpdateProjectRequest & { id: string }) =>
					unwrap<Project>(
						api.PATCH("/projects/{id}", {
							params: { path: { id } },
							body: patch,
						}),
						PROJECT_ERRORS,
					),
			});

			await mutation.execute({
				id: "project-edit-1",
				name: "Updated Name",
				default_branch: "master",
				tracker_provider: "linear",
				tracker_ref: "ENG",
			});

			assert.strictEqual(patchedId, "project-edit-1");
			assert.strictEqual(patchedBody?.name, "Updated Name");
			assert.strictEqual(patchedBody?.default_branch, "master");
			assert.strictEqual(patchedBody?.tracker_provider, "linear");
			assert.strictEqual(patchedBody?.tracker_ref, "ENG");
			// Crucial: slug and repo_ref must NOT be in the patch payload
			assert.strictEqual("slug" in (patchedBody ?? {}), false);
			assert.strictEqual("repo_ref" in (patchedBody ?? {}), false);
		} finally {
			api.PATCH = originalPatch;
		}
	});

	it("Delete requires confirmation and maps 409 conflict error when project has tasks (AC3)", async () => {
		const originalDelete = api.DELETE;

		try {
			api.DELETE = (async () => ({
				error: { error: "PROJECT_HAS_TASKS" },
				response: new Response(JSON.stringify({ error: "PROJECT_HAS_TASKS" }), {
					status: 409,
				}),
			})) as never;

			const queryClient = new QueryClient();
			const mutation = queryClient.getMutationCache().build(queryClient, {
				mutationFn: async (id: string) => {
					const { error, response } = await api.DELETE("/projects/{id}", {
						params: { path: { id } },
					});
					if (!response.ok) {
						const code =
							error &&
							typeof error === "object" &&
							"error" in error &&
							typeof error.error === "string"
								? error.error
								: "PROJECT_HAS_TASKS";
						throw new ApiError(
							response.status,
							code,
							PROJECT_ERRORS[code] ?? code,
						);
					}
				},
			});

			await assert.rejects(
				async () => {
					await mutation.execute("project-with-tasks");
				},
				(err: unknown) => {
					assert.ok(err instanceof ApiError);
					assert.strictEqual(err.status, 409);
					assert.strictEqual(err.code, "PROJECT_HAS_TASKS");
					assert.strictEqual(
						err.message,
						"Project cannot be deleted because it still has associated tasks.",
					);
					return true;
				},
			);
		} finally {
			api.DELETE = originalDelete;
		}
	});

	it("Node path mappings: renders nodes with path inputs, calls PUT to save and clear path (AC4)", async () => {
		const mockNodes: ProjectNodeResponse[] = [
			{
				nodeId: "node-master-1",
				nodeName: "Master Node",
				nodeRole: "master",
				nodeStatus: "online",
				path: "/home/opus/project",
			},
			{
				nodeId: "node-worker-2",
				nodeName: "Worker Node",
				nodeRole: "worker",
				nodeStatus: "offline",
				path: undefined,
			},
		];

		const mockProject: Project = {
			id: "project-node-test",
			slug: "node-test",
			name: "Node Test",
			repo_provider: "github",
			repo_ref: "owner/test",
			default_branch: "main",
		};

		const { html } = renderWithClient(
			<EditProjectModal
				project={mockProject}
				isOpen={true}
				onClose={() => {}}
			/>,
			{
				nodeMappings: {
					projectId: "project-node-test",
					mappings: mockNodes,
				},
			},
		);

		// Assert nodes are displayed
		assert.match(html, /Master Node/);
		assert.match(html, /Worker Node/);

		// Assert path on master is displayed
		assert.match(html, /value="\/home\/opus\/project"/);

		// Assert save buttons exist
		assert.match(html, />Save<\/button>/);

		// Test API PUT calls for setting and clearing path
		const originalPut = api.PUT;
		let lastPutPath: UpdateProjectNodePathRequest | undefined;
		let lastNodeId: string | undefined;

		try {
			api.PUT = (async (
				path: string,
				options: {
					params: { path: { id: string; nodeId: string } };
					body: UpdateProjectNodePathRequest;
				},
			) => {
				if (path === "/projects/{id}/nodes/{nodeId}") {
					lastNodeId = options.params.path.nodeId;
					lastPutPath = options.body;
					return {
						data: {
							data: {
								nodeId: lastNodeId,
								path: options.body.path,
							},
						},
						response: new Response(JSON.stringify({}), { status: 200 }),
					};
				}
				return originalPut(path as never, options as never);
			}) as never;

			const queryClient = new QueryClient();
			const mutation = queryClient.getMutationCache().build(queryClient, {
				mutationFn: async ({
					projectId,
					nodeId,
					path,
				}: {
					projectId: string;
					nodeId: string;
					path: string | null;
				}) =>
					unwrap<ProjectNodeResponse>(
						api.PUT("/projects/{id}/nodes/{nodeId}", {
							params: { path: { id: projectId, nodeId } },
							body: { path: path ?? "" },
						}),
						PROJECT_ERRORS,
					),
			});

			// 1. Setting path
			await mutation.execute({
				projectId: "project-node-test",
				nodeId: "node-worker-2",
				path: "/data/workspace/proj",
			});
			assert.strictEqual(lastNodeId, "node-worker-2");
			assert.strictEqual(lastPutPath?.path, "/data/workspace/proj");

			// 2. Clearing path (sends empty string "")
			await mutation.execute({
				projectId: "project-node-test",
				nodeId: "node-master-1",
				path: "",
			});
			assert.strictEqual(lastNodeId, "node-master-1");
			assert.strictEqual(lastPutPath?.path, "");
		} finally {
			api.PUT = originalPut;
		}
	});

	it("Node path mappings: maps 400 PROJECT_INVALID_PATH error (AC4)", async () => {
		const originalPut = api.PUT;

		try {
			api.PUT = (async () => ({
				error: { error: "PROJECT_INVALID_PATH" },
				response: new Response(
					JSON.stringify({ error: "PROJECT_INVALID_PATH" }),
					{ status: 400 },
				),
			})) as never;

			const queryClient = new QueryClient();
			const mutation = queryClient.getMutationCache().build(queryClient, {
				mutationFn: async ({
					projectId,
					nodeId,
					path,
				}: {
					projectId: string;
					nodeId: string;
					path: string | null;
				}) =>
					unwrap<ProjectNodeResponse>(
						api.PUT("/projects/{id}/nodes/{nodeId}", {
							params: { path: { id: projectId, nodeId } },
							body: { path: path ?? "" },
						}),
						PROJECT_ERRORS,
					),
			});

			await assert.rejects(
				async () => {
					await mutation.execute({
						projectId: "p1",
						nodeId: "n1",
						path: "relative/path/not/allowed",
					});
				},
				(err: unknown) => {
					assert.ok(err instanceof ApiError);
					assert.strictEqual(err.status, 400);
					assert.strictEqual(err.code, "PROJECT_INVALID_PATH");
					assert.strictEqual(err.message, PROJECT_ERRORS.PROJECT_INVALID_PATH);
					return true;
				},
			);
		} finally {
			api.PUT = originalPut;
		}
	});

	it("No imports from sibling ~/features/* in projects feature (AC5)", async () => {
		const { existsSync, readdirSync, readFileSync, statSync } = await import(
			"node:fs"
		);
		const { join } = await import("node:path");

		const baseDir = existsSync("ui/app/features/projects")
			? "ui/app/features/projects"
			: "app/features/projects";

		function collectFiles(dir: string): string[] {
			let results: string[] = [];
			const list = readdirSync(dir);
			for (const file of list) {
				const fullPath = join(dir, file);
				const stat = statSync(fullPath);
				if (stat?.isDirectory()) {
					results = results.concat(collectFiles(fullPath));
				} else if (fullPath.endsWith(".ts") || fullPath.endsWith(".tsx")) {
					results.push(fullPath);
				}
			}
			return results;
		}

		const projectFiles = collectFiles(baseDir);
		assert.ok(projectFiles.length >= 7, "expected projects feature files");

		for (const file of projectFiles) {
			const content = readFileSync(file, "utf-8");
			// Check for imports from ~/features/<anything-other-than-projects>
			const invalidImport = content.match(
				/from\s+["']~\/features\/(?!projects\/)[^"']+["']/,
			);
			assert.strictEqual(
				invalidImport,
				null,
				`File ${file} has illegal sibling feature import: ${invalidImport?.[0]}`,
			);
		}
	});
});
