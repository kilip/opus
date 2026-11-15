export const OTP_RESEND_COOLDOWN_SECONDS = 60;

/**
 * Decrements the cooldown seconds by 1, clamping at 0.
 */
export function nextCooldownTick(seconds: number): number {
	return Math.max(0, seconds - 1);
}

/**
 * Formats the resend button label with an active countdown if seconds > 0.
 */
export function formatCooldownLabel(seconds: number): string {
	if (seconds > 0) {
		return `Resend code in ${Math.ceil(seconds)}s`;
	}
	return "Resend code";
}

/**
 * Formats a user-friendly error message for a 429 RATE_LIMITED response
 * incorporating the Retry-After seconds value when available.
 */
export function formatRateLimitMessage(retryAfter?: number): string {
	if (
		typeof retryAfter === "number" &&
		Number.isFinite(retryAfter) &&
		retryAfter > 0
	) {
		return `Too many requests. Please try again in ${retryAfter} second${retryAfter === 1 ? "" : "s"}.`;
	}
	return "Too many requests. Please try again later.";
}

/**
 * Parses the HTTP Retry-After header into a positive number of seconds.
 * Supports integer seconds and HTTP-date formats.
 */
export function parseRetryAfter(
	headerValue?: string | null,
): number | undefined {
	if (!headerValue) return undefined;
	const trimmed = headerValue.trim();
	const seconds = Number.parseInt(trimmed, 10);
	if (Number.isFinite(seconds) && seconds > 0 && /^\d+$/.test(trimmed)) {
		return seconds;
	}
	const parsedDate = Date.parse(trimmed);
	if (!Number.isNaN(parsedDate)) {
		const diffSeconds = Math.ceil((parsedDate - Date.now()) / 1000);
		return diffSeconds > 0 ? diffSeconds : undefined;
	}
	return undefined;
}
