import { createRootRoute, Outlet } from '@tanstack/react-router';
import { AppShell } from '@/shared/components/layout';

/**
 * RootRoute component defining the global application layout.
 * Includes the offline status banner, dashboard chrome, and the content outlet.
 */
export const Route = createRootRoute({
  component: RootLayout,
});

function RootLayout() {
  return (
    <AppShell>
      <Outlet />
    </AppShell>
  );
}
