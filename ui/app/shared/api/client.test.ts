import assert from "node:assert";
import { describe, it } from "node:test";
import {
  buildExpiredLoginUrl,
  handleUnauthorized,
  isAuthOrSetupPath,
  setActiveQueryClient,
  setUnauthorizedRedirect,
} from "./client";

describe("401 handling logic", () => {
  it("detects auth and setup paths correctly", () => {
    assert.strictEqual(isAuthOrSetupPath("/api/auth/me"), true);
    assert.strictEqual(isAuthOrSetupPath("/api/auth/logout"), true);
    assert.strictEqual(
      isAuthOrSetupPath("http://localhost:3000/api/auth/otp/request"),
      true,
    );
    assert.strictEqual(isAuthOrSetupPath("/api/setup/status"), true);
    assert.strictEqual(isAuthOrSetupPath("/setup/status"), true);
    assert.strictEqual(isAuthOrSetupPath("/auth/me"), true);

    assert.strictEqual(isAuthOrSetupPath("/api/tasks"), false);
    assert.strictEqual(isAuthOrSetupPath("/api/tasks/123"), false);
    assert.strictEqual(
      isAuthOrSetupPath("http://localhost:3000/api/tasks"),
      false,
    );
  });

  it("clears session cache and navigates to /login on 401 for normal API calls", () => {
    let removeQueriesCalled = 0;
    let removedQueryKey: readonly unknown[] | undefined;
    let redirectedTo: string | undefined;

    setActiveQueryClient({
      removeQueries: ({ queryKey }) => {
        removeQueriesCalled++;
        removedQueryKey = queryKey;
      },
    });
    setUnauthorizedRedirect((to) => {
      redirectedTo = to;
    });

    handleUnauthorized("/api/tasks");

    assert.strictEqual(removeQueriesCalled, 1);
    assert.deepStrictEqual(removedQueryKey, ["session"]);
    assert.strictEqual(redirectedTo, "/login?expired=1");
  });

  it("preserves next path and search in expired login URL", () => {
    assert.strictEqual(
      buildExpiredLoginUrl("/tasks?status=open"),
      "/login?next=%2Ftasks%3Fstatus%3Dopen&expired=1",
    );
    assert.strictEqual(buildExpiredLoginUrl("/login"), "/login?expired=1");
  });

  it("ignores 401 from auth or setup endpoints", () => {
    let removeQueriesCalled = 0;
    let redirected = false;

    setActiveQueryClient({
      removeQueries: () => {
        removeQueriesCalled++;
      },
    });
    setUnauthorizedRedirect(() => {
      redirected = true;
    });

    handleUnauthorized("/api/auth/me");
    handleUnauthorized("/api/setup/status");

    assert.strictEqual(removeQueriesCalled, 0);
    assert.strictEqual(redirected, false);
  });

  it("prevents redirect loops when already on /login", () => {
    let removeQueriesCalled = 0;
    let assignedUrl: string | null = null;

    const originalWindow = globalThis.window;
    // biome-ignore lint/suspicious/noExplicitAny: window mock in test
    (globalThis as any).window = {
      location: {
        pathname: "/login",
        assign: (url: string) => {
          assignedUrl = url;
        },
      },
    };

    try {
      setActiveQueryClient({
        removeQueries: () => {
          removeQueriesCalled++;
        },
      });
      setUnauthorizedRedirect((to: string) => {
        if (typeof window !== "undefined" && window.location.pathname !== to) {
          window.location.assign(to);
        }
      });

      handleUnauthorized("/api/tasks");

      assert.strictEqual(removeQueriesCalled, 0);
      assert.strictEqual(assignedUrl, null);
    } finally {
      // biome-ignore lint/suspicious/noExplicitAny: restore window
      (globalThis as any).window = originalWindow;
    }
  });
});
