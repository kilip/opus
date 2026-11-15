import { queryOptions, useQuery } from "@tanstack/react-query";

import { api, unwrap } from "~/shared/api/client";
import { AUTH_ERRORS } from "./errors";
import { authKeys } from "./keys";
import type { ProviderItem, SetupStatus, User } from "./types";

export const sessionQueryOptions = () =>
	queryOptions({
		queryKey: authKeys.me(),
		queryFn: async ({ signal }): Promise<User> =>
			unwrap(api.GET("/auth/me", { signal }), AUTH_ERRORS),
		retry: false,
	});

export const setupStatusQueryOptions = () =>
	queryOptions({
		queryKey: authKeys.setupStatus(),
		queryFn: async ({ signal }): Promise<SetupStatus> =>
			unwrap(api.GET("/setup/status", { signal }), AUTH_ERRORS),
	});

export const providersQueryOptions = () =>
	queryOptions({
		queryKey: authKeys.providers(),
		queryFn: async ({ signal }): Promise<ProviderItem[]> => {
			const data = await unwrap<ProviderItem[]>(
				api.GET("/auth/providers", { signal }),
				AUTH_ERRORS,
			);
			return data ?? [];
		},
	});

export function useSession() {
	return useQuery(sessionQueryOptions());
}

export function useSetupStatus() {
	return useQuery(setupStatusQueryOptions());
}

export function useProviders() {
	return useQuery(providersQueryOptions());
}
