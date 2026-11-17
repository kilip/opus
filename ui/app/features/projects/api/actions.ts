import { useMutation, useQueryClient } from "@tanstack/react-query";

import { ApiError, api, unwrap } from "~/shared/api/client";
import { resolveErrorMessage } from "~/shared/api/errors";
import { PROJECT_ERRORS } from "./errors";
import { projectKeys } from "./keys";
import type {
	CreateProjectRequest,
	Project,
	ProjectNodeResponse,
	UpdateProjectRequest,
} from "./types";

export function useCreateProject() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: async (req: CreateProjectRequest): Promise<Project> =>
			unwrap(
				api.POST("/projects", {
					body: req,
				}),
				PROJECT_ERRORS,
			),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: projectKeys.all });
		},
	});
}

export function useUpdateProject() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: async ({
			id,
			...patch
		}: UpdateProjectRequest & { id: string }): Promise<Project> =>
			unwrap(
				api.PATCH("/projects/{id}", {
					params: { path: { id } },
					body: patch,
				}),
				PROJECT_ERRORS,
			),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: projectKeys.all });
		},
	});
}

export function useDeleteProject() {
	const queryClient = useQueryClient();

	return useMutation({
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
						: response.status === 404
							? "RESOURCE_NOT_FOUND"
							: response.status === 409
								? "PROJECT_HAS_TASKS"
								: "REQUEST_FAILED";
				throw new ApiError(
					response.status,
					code,
					resolveErrorMessage(code, PROJECT_ERRORS),
				);
			}
		},
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: projectKeys.all });
		},
	});
}

export function useSetProjectNodePath() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: async ({
			projectId,
			nodeId,
			path,
		}: {
			projectId: string;
			nodeId: string;
			path: string | null;
		}): Promise<ProjectNodeResponse> =>
			unwrap(
				api.PUT("/projects/{id}/nodes/{nodeId}", {
					params: { path: { id: projectId, nodeId } },
					body: { path: path ?? "" },
				}),
				PROJECT_ERRORS,
			),
		onSuccess: (_data, variables) => {
			queryClient.invalidateQueries({
				queryKey: projectKeys.nodes(variables.projectId),
			});
		},
	});
}
