import assert from "node:assert";
import { describe, it } from "node:test";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createElement } from "react";
import { renderToString } from "react-dom/server";
import { MemoryRouter } from "react-router";

import { AUTH_ERRORS } from "../api/errors";
import { authKeys } from "../api/keys";
import type { Account, ProviderItem } from "../api/types";
import { authSettingsRoutes } from "../routes";
import ConnectedAccountsPage from "./ConnectedAccountsPage";

function renderConnectedAccountsPage({
  accounts = [],
  providers = [],
  url = "/settings/accounts",
}: {
  accounts?: Account[];
  providers?: ProviderItem[];
  url?: string;
} = {}) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  queryClient.setQueryData(authKeys.accounts(), accounts);
  queryClient.setQueryData(authKeys.providers(), providers);

  return renderToString(
    createElement(
      QueryClientProvider,
      { client: queryClient },
      createElement(
        MemoryRouter,
        { initialEntries: [url] },
        createElement(ConnectedAccountsPage),
      ),
    ),
  );
}

describe("ConnectedAccountsPage (AC1, AC2, AC3, AC4)", () => {
  it("exports authSettingsRoutes for settings/accounts (AC1)", () => {
    assert.strictEqual(authSettingsRoutes.length, 1);
    const route = authSettingsRoutes[0];
    assert.strictEqual(route.path, "settings/accounts");
    assert.strictEqual(
      route.file,
      "features/auth/pages/ConnectedAccountsPage.tsx",
    );
  });

  it("lists linked accounts with provider, email and disconnect button (AC2, AC3)", () => {
    const html = renderConnectedAccountsPage({
      accounts: [
        {
          id: "acc-1",
          provider: "github",
          email: "octocat@github.com",
          created_at: "2026-01-01T00:00:00Z",
        },
      ],
      providers: [
        { name: "github", label: "GitHub" },
        { name: "google", label: "Google" },
      ],
    });

    // Linked account item rendered
    assert.match(html, /GitHub/);
    assert.match(html, /octocat@github\.com/);
    assert.match(html, /Disconnect/);
    assert.match(html, /Disconnect GitHub/);

    // Unlinked provider has Connect button
    assert.match(html, /Google/);
    assert.match(html, /href="\/api\/auth\/oauth\/google\?link=1"/);
    assert.match(html, /Connect/);

    // Linked provider does NOT show Connect button
    assert.doesNotMatch(html, /href="\/api\/auth\/oauth\/github\?link=1"/);
  });

  it("shows empty state when no accounts are linked (AC2)", () => {
    const html = renderConnectedAccountsPage({
      accounts: [],
      providers: [{ name: "google", label: "Google" }],
    });

    assert.match(html, /No connected accounts yet\./);
    assert.match(html, /href="\/api\/auth\/oauth\/google\?link=1"/);
  });

  it("shows all connected message when all providers are linked (AC2)", () => {
    const html = renderConnectedAccountsPage({
      accounts: [{ id: "acc-1", provider: "google", email: "user@gmail.com" }],
      providers: [{ name: "google", label: "Google" }],
    });

    assert.match(html, /All available providers are connected\./);
    assert.doesNotMatch(html, /\?link=1/);
  });

  it("renders readable error alert for ?error=ACCOUNT_ALREADY_LINKED (AC4)", () => {
    const html = renderConnectedAccountsPage({
      url: "/settings/accounts?error=ACCOUNT_ALREADY_LINKED",
    });

    assert.match(html, /role="alert"/);
    assert.match(html, new RegExp(AUTH_ERRORS.ACCOUNT_ALREADY_LINKED));
  });

  it("renders readable error alert for unknown or other errors (AC4)", () => {
    const html = renderConnectedAccountsPage({
      url: "/settings/accounts?error=PROVIDER_NOT_FOUND",
    });

    assert.match(html, /role="alert"/);
    assert.match(html, new RegExp(AUTH_ERRORS.PROVIDER_NOT_FOUND));
  });
});
