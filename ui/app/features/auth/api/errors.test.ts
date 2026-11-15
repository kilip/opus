import assert from "node:assert";
import { describe, it } from "node:test";

import { resolveErrorMessage } from "~/shared/api/errors";
import { AUTH_ERRORS } from "./errors";

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
});
