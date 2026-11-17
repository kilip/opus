import { queryOptions, useQuery } from "@tanstack/react-query";

import { api, unwrap } from "~/shared/api/client";
import { PROJECT_ERRORS } from "./errors";
import { projectKeys } from "./keys";
import type {
	Project,
	ProjectNodeResponse,
	TrackerSettingsResponse,
	TrackerStatusResponse,
} from "./types";

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

export const projectTrackerQueryOptions = (projectId: string) =>
	queryOptions({
		queryKey: projectKeys.tracker(projectId),
		queryFn: async ({ signal }): Promise<TrackerStatusResponse> =>
			unwrap(
				api.GET("/projects/{id}/tracker", {
					params: { path: { id: projectId } },
					signal,
				}),
				PROJECT_ERRORS,
			),
		enabled: Boolean(projectId),
	});

export function useProjectTracker(projectId: string) {
	return useQuery(projectTrackerQueryOptions(projectId));
}

export const trackerSettingsQueryOptions = () =>
	queryOptions({
		queryKey: projectKeys.trackerSettings(),
		queryFn: async ({ signal }): Promise<TrackerSettingsResponse> =>
			unwrap(api.GET("/tracker/settings", { signal }), PROJECT_ERRORS),
	});

export function useTrackerSettings() {
	return useQuery(trackerSettingsQueryOptions());
}
