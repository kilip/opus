import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "react-router";

import { api, unwrap } from "~/shared/api/client";
import { AUTH_ERRORS } from "./errors";
import { authKeys } from "./keys";
import type { OTPRequestBody, OTPVerifyBody, User } from "./types";

export function useRequestOTP() {
	return useMutation({
		mutationFn: async (req: OTPRequestBody): Promise<string> =>
			unwrap(
				api.POST("/auth/otp/request", {
					body: req,
					parseAs: "text" as never,
				}),
				AUTH_ERRORS,
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
			unwrap(
				api.POST("/setup/otp", {
					body: req,
					parseAs: "text" as never,
				}),
				AUTH_ERRORS,
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
