import type { ReactNode } from "react";
import { Navigate, Outlet, useLocation } from "react-router";

import { ApiError } from "../api/client";
import { ErrorState } from "../components/ErrorState";
import { useSession, useSetupStatus } from "./session";

export function RequireAuth({ children }: { children?: ReactNode }) {
  const location = useLocation();
  const setupQuery = useSetupStatus();
  const sessionQuery = useSession();

  if (setupQuery.isLoading || sessionQuery.isLoading) {
    return (
      <div
        role="status"
        aria-live="polite"
        className="flex min-h-dvh items-center justify-center bg-bg p-6 text-sm text-muted"
      >
        Loading...
      </div>
    );
  }

  if (setupQuery.isError) {
    return (
      <main className="flex min-h-dvh items-center justify-center bg-bg p-6">
        <ErrorState
          title="Unable to reach server"
          message={
            setupQuery.error instanceof Error
              ? setupQuery.error.message
              : "Could not check server status."
          }
          onRetry={() => {
            void setupQuery.refetch();
          }}
        />
      </main>
    );
  }

  if (setupQuery.data?.needs_setup) {
    return <Navigate to="/onboarding" replace />;
  }

  if (sessionQuery.isError) {
    const isUnauthorized =
      sessionQuery.error instanceof ApiError &&
      sessionQuery.error.status === 401;
    if (!isUnauthorized) {
      return (
        <main className="flex min-h-dvh items-center justify-center bg-bg p-6">
          <ErrorState
            title="Unable to verify session"
            message={
              sessionQuery.error instanceof Error
                ? sessionQuery.error.message
                : "Could not verify your session."
            }
            onRetry={() => {
              void sessionQuery.refetch();
            }}
          />
        </main>
      );
    }
  }

  if (!sessionQuery.data || sessionQuery.isError) {
    const next = `${location.pathname}${location.search}`;
    const loginTo = next ? `/login?next=${encodeURIComponent(next)}` : "/login";
    return <Navigate to={loginTo} replace />;
  }

  return (children ?? <Outlet />) as ReactNode;
}

export default RequireAuth;
