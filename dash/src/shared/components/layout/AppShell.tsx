import { type ReactNode, useState } from 'react';
import { DashboardHeader } from '@/shared/components/layout/DashboardHeader';
import { DashboardSidebar } from '@/shared/components/layout/DashboardSidebar';
import { MobileNavDrawer } from '@/shared/components/layout/MobileNavDrawer';
import { OfflineBanner } from '@/shared/components/OfflineBanner';

type AppShellProps = {
  children: ReactNode;
};

/**
 * Root dashboard chrome: sidebar, header, mobile drawer, and content region.
 */
export function AppShell({ children }: AppShellProps) {
  const [mobileNavOpen, setMobileNavOpen] = useState(false);

  return (
    <div className="grain-overlay min-h-screen bg-background text-foreground">
      <OfflineBanner />
      <div className="flex min-h-screen">
        <DashboardSidebar className="hidden lg:flex" />
        <MobileNavDrawer
          open={mobileNavOpen}
          onClose={() => setMobileNavOpen(false)}
        />

        <div className="flex min-w-0 flex-1 flex-col">
          <DashboardHeader onMenuOpen={() => setMobileNavOpen(true)} />
          <main className="flex-1 px-4 py-6 sm:px-6 sm:py-8 lg:px-10">
            <div className="mx-auto w-full max-w-6xl">{children}</div>
          </main>
        </div>
      </div>
    </div>
  );
}
