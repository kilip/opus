import assert from "node:assert";
import { describe, it } from "node:test";
import { AUTH_ERRORS, resolveAuthError } from "./api/errors";
import {
	formatCooldownLabel,
	formatRateLimitMessage,
	nextCooldownTick,
	OTP_RESEND_COOLDOWN_SECONDS,
	parseRetryAfter,
} from "./model";

describe("auth model (AC1, AC5)", () => {
	it("defines OTP_RESEND_COOLDOWN_SECONDS as 60 (AC1)", () => {
		assert.strictEqual(OTP_RESEND_COOLDOWN_SECONDS, 60);
	});

	it("decrements cooldown seconds correctly (AC5)", () => {
		assert.strictEqual(nextCooldownTick(60), 59);
		assert.strictEqual(nextCooldownTick(59), 58);
		assert.strictEqual(nextCooldownTick(2), 1);
		assert.strictEqual(nextCooldownTick(1), 0);
		assert.strictEqual(nextCooldownTick(0), 0);
		assert.strictEqual(nextCooldownTick(-5), 0);
	});

	it("formats cooldown label for active countdown and zero (AC2)", () => {
		assert.strictEqual(formatCooldownLabel(60), "Resend code in 60s");
		assert.strictEqual(formatCooldownLabel(45), "Resend code in 45s");
		assert.strictEqual(formatCooldownLabel(1), "Resend code in 1s");
		assert.strictEqual(formatCooldownLabel(0), "Resend code");
		assert.strictEqual(formatCooldownLabel(-1), "Resend code");
	});

	it("formats rate limit message using retryAfter value (AC3)", () => {
		assert.strictEqual(
			formatRateLimitMessage(60),
			"Too many requests. Please try again in 60 seconds.",
		);
		assert.strictEqual(
			formatRateLimitMessage(1),
			"Too many requests. Please try again in 1 second.",
		);
		assert.strictEqual(
			formatRateLimitMessage(undefined),
			"Too many requests. Please try again later.",
		);
		assert.strictEqual(
			formatRateLimitMessage(0),
			"Too many requests. Please try again later.",
		);
		assert.strictEqual(
			formatRateLimitMessage(-10),
			"Too many requests. Please try again later.",
		);
	});

	it("parses Retry-After header values correctly", () => {
		assert.strictEqual(parseRetryAfter("60"), 60);
		assert.strictEqual(parseRetryAfter("  30  "), 30);
		assert.strictEqual(parseRetryAfter("0"), undefined);
		assert.strictEqual(parseRetryAfter("-1"), undefined);
		assert.strictEqual(parseRetryAfter("not-a-number"), undefined);
		assert.strictEqual(parseRetryAfter(""), undefined);
		assert.strictEqual(parseRetryAfter(null), undefined);
		assert.strictEqual(parseRetryAfter(undefined), undefined);
	});
});

describe("OAuth error resolution (AC4)", () => {
	it("resolves readable message for ACCOUNT_NOT_ALLOWED", () => {
		const msg = resolveAuthError("ACCOUNT_NOT_ALLOWED");
		assert.strictEqual(msg, AUTH_ERRORS.ACCOUNT_NOT_ALLOWED);
		assert.match(msg ?? "", /not authorized/i);
	});

	it("resolves readable message for SETUP_REQUIRED", () => {
		const msg = resolveAuthError("SETUP_REQUIRED");
		assert.strictEqual(msg, AUTH_ERRORS.SETUP_REQUIRED);
		assert.match(msg ?? "", /setup is required/i);
	});

	it("resolves readable message for PROVIDER_NOT_FOUND", () => {
		const msg = resolveAuthError("PROVIDER_NOT_FOUND");
		assert.strictEqual(msg, AUTH_ERRORS.PROVIDER_NOT_FOUND);
		assert.match(msg ?? "", /provider was not found/i);
	});

	it("returns null or fallback for empty or unknown error codes", () => {
		assert.strictEqual(resolveAuthError(null), null);
		assert.strictEqual(resolveAuthError(""), null);
		assert.strictEqual(resolveAuthError(undefined), null);
		assert.strictEqual(resolveAuthError("UNKNOWN_ERROR"), "UNKNOWN_ERROR");
	});
});
