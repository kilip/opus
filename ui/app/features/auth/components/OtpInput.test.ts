import assert from "node:assert";
import { describe, it } from "node:test";
import { createElement } from "react";
import { renderToString } from "react-dom/server";

import { OtpInput } from "./OtpInput";

describe("OtpInput component (AC2, AC7)", () => {
	it("renders 6-digit numeric input with required accessibility and autofill attributes", () => {
		const html = renderToString(
			createElement(OtpInput, {
				id: "otp-test-id",
				value: "123",
				onChange: () => {},
				"aria-invalid": "false",
			}),
		);

		assert.match(html, /id="otp-test-id"/);
		assert.match(html, /inputMode="numeric"/i);
		assert.match(html, /autoComplete="one-time-code"/i);
		assert.match(html, /maxLength="6"/i);
		assert.match(html, /value="123"/);
	});

	it("reflects aria-invalid when provided", () => {
		const html = renderToString(
			createElement(OtpInput, {
				id: "otp-err-id",
				value: "",
				onChange: () => {},
				"aria-invalid": "true",
			}),
		);

		assert.match(html, /aria-invalid="true"/);
	});
});
