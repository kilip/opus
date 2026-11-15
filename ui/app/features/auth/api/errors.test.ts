import assert from "node:assert";
import { describe, it } from "node:test";

import { ApiError } from "~/shared/api/client";
import { resolveErrorMessage } from "~/shared/api/errors";
import { AUTH_ERRORS, RateLimitError } from "./errors";

describe("auth errors mapping (AC4)", () => {
	it("maps INVALID_OTP to readable message", () => {
		const msg = resolveErrorMessage("INVALID_OTP", AUTH_ERRORS);
		assert.strictEqual(msg, AUTH_ERRORS.INVALID_OTP);
		assert.match(msg, /invalid/i);
	});

	it("maps OTP_EXPIRED to readable message", () => {
		const msg = resolveErrorMessage("OTP_EXPIRED", AUTH_ERRORS);
		assert.strictEqual(msg, AUTH_ERRORS.OTP_EXPIRED);
		assert.match(msg, /expired/i);
	});

	it("maps TOO_MANY_ATTEMPTS to readable message", () => {
		const msg = resolveErrorMessage("TOO_MANY_ATTEMPTS", AUTH_ERRORS);
		assert.strictEqual(msg, AUTH_ERRORS.TOO_MANY_ATTEMPTS);
		assert.match(msg, /too many attempts/i);
	});

	it("maps SETUP_CLOSED to readable message", () => {
		const msg = resolveErrorMessage("SETUP_CLOSED", AUTH_ERRORS);
		assert.strictEqual(msg, AUTH_ERRORS.SETUP_CLOSED);
		assert.match(msg, /completed/i);
	});

	it("maps ACCOUNT_NOT_ALLOWED to readable message", () => {
		const msg = resolveErrorMessage("ACCOUNT_NOT_ALLOWED", AUTH_ERRORS);
		assert.strictEqual(msg, AUTH_ERRORS.ACCOUNT_NOT_ALLOWED);
		assert.match(msg, /not authorized/i);
	});

	it("maps SETUP_REQUIRED to readable message", () => {
		const msg = resolveErrorMessage("SETUP_REQUIRED", AUTH_ERRORS);
		assert.strictEqual(msg, AUTH_ERRORS.SETUP_REQUIRED);
		assert.match(msg, /setup is required/i);
	});

	it("maps PROVIDER_NOT_FOUND to readable message", () => {
		const msg = resolveErrorMessage("PROVIDER_NOT_FOUND", AUTH_ERRORS);
		assert.strictEqual(msg, AUTH_ERRORS.PROVIDER_NOT_FOUND);
		assert.match(msg, /provider was not found/i);
	});

	it("RateLimitError creates 429 RATE_LIMITED error with retry-after message (AC3)", () => {
		const errWithRetry = new RateLimitError(45);
		assert.ok(errWithRetry instanceof ApiError);
		assert.strictEqual(errWithRetry.status, 429);
		assert.strictEqual(errWithRetry.code, "RATE_LIMITED");
		assert.strictEqual(errWithRetry.retryAfter, 45);
		assert.strictEqual(
			errWithRetry.message,
			"Too many requests. Please try again in 45 seconds.",
		);

		const errWithoutRetry = new RateLimitError();
		assert.strictEqual(errWithoutRetry.status, 429);
		assert.strictEqual(errWithoutRetry.code, "RATE_LIMITED");
		assert.strictEqual(errWithoutRetry.retryAfter, undefined);
		assert.strictEqual(
			errWithoutRetry.message,
			"Too many requests. Please try again later.",
		);
	});
});
