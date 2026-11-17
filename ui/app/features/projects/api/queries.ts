import { queryOptions, useQuery } from "@tanstack/react-query";

import { api, unwrap } from "~/shared/api/client";
import { PROJECT_ERRORS } from "./errors";
import { projectKeys } from "./keys";
import type { Project, ProjectNodeResponse } from "./types";

export const projectsQueryOptions = () =>
	queryOptions({
		queryKey: projectKeys.list(),
		queryFn: async ({ signal }): Promise<Project[]> =>
			unwrap(api.GET("/projects", { signal }), PROJECT_ERRORS),
	});

export function useProjects() {
	return useQuery(projectsQueryOptions());
}

export const projectNodesQueryOptions = (projectId: string) =>
	queryOptions({
		queryKey: projectKeys.nodes(projectId),
		queryFn: async ({ signal }): Promise<ProjectNodeResponse[]> =>
			unwrap(
				api.GET("/projects/{id}/nodes", {
					params: { path: { id: projectId } },
					signal,
				}),
				PROJECT_ERRORS,
			),
		enabled: Boolean(projectId),
	});

export function useProjectNodeMappings(projectId: string) {
	return useQuery(projectNodesQueryOptions(projectId));
}
