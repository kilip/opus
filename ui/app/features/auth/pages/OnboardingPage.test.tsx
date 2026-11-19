import assert from "node:assert";
import { describe, it } from "node:test";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createElement } from "react";
import { renderToString } from "react-dom/server";
import { MemoryRouter } from "react-router";

import { authKeys } from "../api/keys";
import OnboardingPage from "./OnboardingPage";

describe("OnboardingPage", () => {
  it("renders setup token and email form when needs_setup is true", () => {
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
    queryClient.setQueryData(authKeys.setupStatus(), { needs_setup: true });
    queryClient.setQueryData(authKeys.me(), null);

    const html = renderToString(
      createElement(
        QueryClientProvider,
        { client: queryClient },
        createElement(
          MemoryRouter,
          { initialEntries: ["/onboarding"] },
          createElement(OnboardingPage),
        ),
      ),
    );

    assert.match(html, /Welcome to Opus/);
    assert.match(html, /Owner email/);
    assert.match(html, /Continue/);
  });
});
