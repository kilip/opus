import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "react-router";

import { ApiError, api, unwrap } from "~/shared/api/client";
import { resolveErrorMessage } from "~/shared/api/errors";
import { parseRetryAfter } from "../model";
import { AUTH_ERRORS, RateLimitError } from "./errors";
import { authKeys } from "./keys";
import type { OTPRequestBody, OTPVerifyBody, User } from "./types";

async function handleOtpRequest(
	requestPromise: Promise<{
		data?: unknown;
		error?: unknown;
		response: Response;
	}>,
): Promise<string> {
	const res = await requestPromise;
	if (res.response.status === 429) {
		const retryAfter = parseRetryAfter(res.response.headers.get("Retry-After"));
		throw new RateLimitError(retryAfter);
	}
	return unwrap<string>(res, AUTH_ERRORS);
}

export function useRequestOTP() {
	return useMutation({
		mutationFn: async (req: OTPRequestBody): Promise<string> =>
			handleOtpRequest(
				api.POST("/auth/otp/request", {
					body: req,
					parseAs: "text" as never,
				}),
			),
	});
}

export function useVerifyOTP() {
	const queryClient = useQueryClient();
	return useMutation({
		mutationFn: async (req: OTPVerifyBody): Promise<User> =>
			unwrap(
				api.POST("/auth/otp/verify", {
					body: req,
				}),
				AUTH_ERRORS,
			),
		onSuccess: (user) => {
			queryClient.setQueryData(authKeys.me(), user);
			queryClient.invalidateQueries({ queryKey: authKeys.all });
		},
	});
}

export function useRequestSetupOTP() {
	return useMutation({
		mutationFn: async (req: OTPRequestBody): Promise<string> =>
			handleOtpRequest(
				api.POST("/setup/otp", {
					body: req,
					parseAs: "text" as never,
				}),
			),
	});
}

export function useCompleteSetup() {
	const queryClient = useQueryClient();
	return useMutation({
		mutationFn: async (req: OTPVerifyBody): Promise<User> =>
			unwrap(
				api.POST("/setup", {
					body: req,
				}),
				AUTH_ERRORS,
			),
		onSuccess: (user) => {
			queryClient.setQueryData(authKeys.me(), user);
			queryClient.invalidateQueries({ queryKey: authKeys.all });
		},
	});
}

export function useLogout() {
	const queryClient = useQueryClient();
	const navigate = useNavigate();

	return useMutation({
		mutationFn: async () => {
			const { response } = await api.POST("/auth/logout");
			if (!response.ok) {
				throw new Error("Logout failed");
			}
		},
		onSettled: () => {
			queryClient.removeQueries({ queryKey: authKeys.all });
			navigate("/login");
		},
	});
}

export function useDisconnectAccount() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: async (id: string) => {
			const { error, response } = await api.DELETE("/auth/accounts/{id}", {
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
							: "REQUEST_FAILED";
				throw new ApiError(
					response.status,
					code,
					resolveErrorMessage(code, AUTH_ERRORS),
				);
			}
		},
		onSuccess: () => {
			queryClient.invalidateQueries({ queryKey: authKeys.accounts() });
		},
	});
}
