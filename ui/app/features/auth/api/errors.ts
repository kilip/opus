import { ApiError } from "~/shared/api/client";
import { resolveErrorMessage } from "~/shared/api/errors";
import { formatRateLimitMessage } from "../model";

export const AUTH_ERRORS: Record<string, string> = {
	INVALID_OTP: "Invalid verification code. Please try again.",
	OTP_EXPIRED: "The verification code has expired. Please request a new one.",
	TOO_MANY_ATTEMPTS: "Too many attempts. Please try again later.",
	SETUP_CLOSED: "Setup has already been completed.",
	ACCOUNT_NOT_ALLOWED: "Your account is not authorized to sign in.",
	SETUP_REQUIRED: "Initial setup is required before signing in.",
	PROVIDER_NOT_FOUND: "The requested authentication provider was not found.",
	ACCOUNT_ALREADY_LINKED: "This account is already linked to another user.",
	ACCOUNT_NOT_FOUND: "The requested account was not found.",
};

export function resolveAuthError(code?: string | null): string | null {
	if (!code) return null;
	return resolveErrorMessage(code, AUTH_ERRORS);
}

export class RateLimitError extends ApiError {
	readonly retryAfter?: number;

	constructor(retryAfter?: number, message?: string) {
		super(429, "RATE_LIMITED", message ?? formatRateLimitMessage(retryAfter));
		this.name = "RateLimitError";
		this.retryAfter = retryAfter;
	}
}
