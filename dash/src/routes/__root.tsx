import { createRootRoute, Outlet } from '@tanstack/react-router';
import { OfflineBanner } from '@/shared/components/OfflineBanner';

/**
 * RootRoute component defining the global application layout.
 * Includes the offline status banner, the global header, and the content outlet.
 */
export const Route = createRootRoute({
  component: () => (
    <div className="min-h-screen flex flex-col bg-slate-950 text-slate-100 antialiased selection:bg-indigo-500 selection:text-white">
      <OfflineBanner />
      <header className="sticky top-0 z-50 flex items-center justify-between border-b border-slate-800/80 bg-slate-950/85 px-6 py-4 backdrop-blur-md">
        <div className="flex items-center gap-3">
          <div className="h-2 w-2 rounded-full bg-emerald-500 animate-pulse" />
          <h1 className="text-lg font-bold tracking-tight text-white">Opus Dash</h1>
        </div>
      </header>
      <main className="flex-1 p-6 max-w-7xl mx-auto w-full">
        <Outlet />
      </main>
    </div>
  ),
});
