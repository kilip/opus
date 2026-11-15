import { ApiError } from "~/shared/api/client";
import { formatRateLimitMessage } from "../model";

export const AUTH_ERRORS: Record<string, string> = {
	INVALID_OTP: "Invalid verification code. Please try again.",
	OTP_EXPIRED: "The verification code has expired. Please request a new one.",
	TOO_MANY_ATTEMPTS: "Too many attempts. Please try again later.",
	SETUP_CLOSED: "Setup has already been completed.",
};

export class RateLimitError extends ApiError {
	readonly retryAfter?: number;

	constructor(retryAfter?: number, message?: string) {
		super(429, "RATE_LIMITED", message ?? formatRateLimitMessage(retryAfter));
		this.name = "RateLimitError";
		this.retryAfter = retryAfter;
	}
}
