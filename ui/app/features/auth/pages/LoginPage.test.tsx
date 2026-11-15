import assert from "node:assert";
import { describe, it } from "node:test";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createElement } from "react";
import { renderToString } from "react-dom/server";
import { MemoryRouter } from "react-router";

import { AUTH_ERRORS } from "../api/errors";
import { authKeys } from "../api/keys";
import LoginPage from "./LoginPage";

function renderLoginPage({
	providers = [],
	url = "/login",
}: {
	providers?: { name: string; label: string }[];
	url?: string;
} = {}) {
	const queryClient = new QueryClient({
		defaultOptions: { queries: { retry: false } },
	});
	queryClient.setQueryData(authKeys.setupStatus(), { needs_setup: false });
	queryClient.setQueryData(authKeys.me(), null);
	queryClient.setQueryData(authKeys.providers(), providers);

	return renderToString(
		createElement(
			QueryClientProvider,
			{ client: queryClient },
			createElement(
				MemoryRouter,
				{ initialEntries: [url] },
				createElement(LoginPage),
			),
		),
	);
}

describe("LoginPage (AC2, AC4)", () => {
	it("renders OAuth link buttons to /api/auth/oauth/<name> when providers exist (AC2)", () => {
		const html = renderLoginPage({
			providers: [
				{ name: "github", label: "GitHub" },
				{ name: "google", label: "Google" },
			],
		});

		assert.match(html, /href="\/api\/auth\/oauth\/github"/);
		assert.match(html, /Continue with GitHub/);
		assert.match(html, /href="\/api\/auth\/oauth\/google"/);
		assert.match(html, /Continue with Google/);
		assert.match(html, /<a /);
	});

	it("does not render provider section or divider when providers list is empty (AC2)", () => {
		const html = renderLoginPage({ providers: [] });

		assert.doesNotMatch(html, /\/api\/auth\/oauth\//);
		assert.doesNotMatch(html, /Continue with GitHub/);
		assert.doesNotMatch(html, /Continue with Google/);
	});

	it("renders readable error banner for ?error=ACCOUNT_NOT_ALLOWED (AC4)", () => {
		const html = renderLoginPage({
			url: "/login?error=ACCOUNT_NOT_ALLOWED",
		});

		assert.match(html, /role="alert"/);
		assert.match(html, new RegExp(AUTH_ERRORS.ACCOUNT_NOT_ALLOWED));
	});

	it("renders readable error banner for ?error=SETUP_REQUIRED (AC4)", () => {
		const html = renderLoginPage({
			url: "/login?error=SETUP_REQUIRED",
		});

		assert.match(html, /role="alert"/);
		assert.match(html, new RegExp(AUTH_ERRORS.SETUP_REQUIRED));
	});

	it("renders readable error banner for ?error=PROVIDER_NOT_FOUND (AC4)", () => {
		const html = renderLoginPage({
			url: "/login?error=PROVIDER_NOT_FOUND",
		});

		assert.match(html, /role="alert"/);
		assert.match(html, new RegExp(AUTH_ERRORS.PROVIDER_NOT_FOUND));
	});
});
