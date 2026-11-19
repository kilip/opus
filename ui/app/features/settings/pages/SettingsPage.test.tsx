import assert from "node:assert";
import { describe, it } from "node:test";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createElement } from "react";
import { renderToString } from "react-dom/server";
import { MemoryRouter } from "react-router";
import { settingsKeys } from "../api/keys";
import type { GitHubTokenStatus } from "../api/types";
import { settingsApp } from "../app";
import { settingsRoutes } from "../routes";
import SettingsPage from "./SettingsPage";

function renderSettingsPage(status?: GitHubTokenStatus) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  if (status) {
    queryClient.setQueryData(settingsKeys.githubToken(), status);
  }

  return renderToString(
    createElement(
      QueryClientProvider,
      { client: queryClient },
      createElement(
        MemoryRouter,
        { initialEntries: ["/settings"] },
        createElement(SettingsPage),
      ),
    ),
  );
}

describe("SettingsPage (AC5, AC6, AC7)", () => {
  it("exports settingsRoutes and enabled settingsApp", () => {
    assert.strictEqual(settingsApp.enabled, true);
    assert.strictEqual(settingsApp.path, "/settings");
    assert.strictEqual(settingsRoutes.length, 1);
    assert.strictEqual(settingsRoutes[0].path, "settings");
    assert.strictEqual(
      settingsRoutes[0].file,
      "features/settings/pages/SettingsPage.tsx",
    );
  });

  it("renders masked input, save button, and unconfigured state", () => {
    const html = renderSettingsPage({
      configured: false,
      envOverride: false,
      needsReentry: false,
    });

    assert.match(html, /type="password"/);
    assert.match(html, />Save</);
    assert.match(html, /No stored GitHub token configured/);
    assert.doesNotMatch(html, />Remove</);
  });

  it("renders configured last4, Remove button, and envOverride notice (AC5)", () => {
    const html = renderSettingsPage({
      configured: true,
      last4: "4321",
      updatedAt: "2026-11-19T10:00:00Z",
      envOverride: true,
      needsReentry: false,
    });

    assert.match(html, /••••4321/);
    assert.match(html, />Remove</);
    assert.match(html, /GH_TOKEN/);
    assert.match(html, /overrides the stored token/);
  });

  it("renders re-entry notice when needsReentry is true (AC6)", () => {
    const html = renderSettingsPage({
      configured: false,
      last4: "9999",
      envOverride: false,
      needsReentry: true,
    });

    assert.match(html, /re-enter your GitHub token/);
    assert.match(html, />Remove</);
  });
});
