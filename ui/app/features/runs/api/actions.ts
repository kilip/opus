import { useMutation, useQueryClient } from "@tanstack/react-query";

import { api, unwrap } from "~/shared/api/client";
import { RUN_ERRORS } from "./errors";
import { runKeys } from "./keys";
import type { RunItem } from "./types";

export function useCancelRun() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: async ({
			id,
			reason,
		}: {
			id: string;
			reason?: string;
		}): Promise<RunItem> =>
			unwrap(
				api.POST("/runs/{id}/cancel", {
					params: { path: { id } },
					body: reason ? { reason } : {},
				}),
				RUN_ERRORS,
			),
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: runKeys.all });
		},
	});
}
