import type { QueryClient } from '@tanstack/react-query';
import {
  createRootRouteWithContext,
  Outlet,
  redirect,
  useLocation,
} from '@tanstack/react-router';
import { authQueries } from '@/features/auth/api';
import { AppShell } from '@/shared/components/layout';

type RootContext = {
  queryClient: QueryClient;
};

/**
 * RootRoute component defining the global application layout.
 * Consistent with ADR-011 for authentication guards.
 */
export const Route = createRootRouteWithContext<RootContext>()({
  beforeLoad: async ({ context, location }) => {
    // Skip auth check for demo routes if they exist, or specific public routes
    if (location.pathname.startsWith('/demo')) {
      return;
    }

    try {
      // Resolve current auth state from the server
      const user = await context.queryClient.fetchQuery(authQueries.me());

      // If authenticated and trying to access login, redirect to dashboard
      if (location.pathname === '/login') {
        throw redirect({ to: '/agent' });
      }

      return { user };
    } catch (error) {
      // If redirect was already thrown, re-throw it
      if (error instanceof Object && 'to' in error) throw error;

      // If not authenticated and not already on login, redirect to login
      if (location.pathname !== '/login') {
        throw redirect({ to: '/login' });
      }
    }
  },
  component: RootLayout,
});

function RootLayout() {
  const location = useLocation();
  const isLoginPage = location.pathname === '/login';

  // If it's the login page, render without the dashboard shell
  if (isLoginPage) {
    return <Outlet />;
  }

  return (
    <AppShell>
      <Outlet />
    </AppShell>
  );
}
