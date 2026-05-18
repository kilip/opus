import opusLogo from '@/assets/opus.svg';
import { mainNavItems } from '@/shared/components/layout/nav-config';
import { SidebarNav } from '@/shared/components/layout/SidebarNav';
import { cn } from '@/shared/lib/utils';

type DashboardSidebarProps = {
  className?: string;
  onNavigate?: () => void;
};

/**
 * Desktop sidebar with brand mark and primary navigation.
 */
export function DashboardSidebar({
  className,
  onNavigate,
}: DashboardSidebarProps) {
  return (
    <aside
      className={cn(
        'flex w-64 shrink-0 flex-col border-r border-border bg-sidebar text-sidebar-foreground',
        className,
      )}
    >
      <div className="flex items-center gap-3 border-b border-border px-5 py-5">
        <img src={opusLogo} alt="" className="h-9 w-9 shrink-0" aria-hidden />
        <div className="min-w-0">
          <p className="font-sans text-base font-semibold leading-tight tracking-tight">
            Opus
          </p>
          <p className="font-serif text-xs text-muted truncate">
            Autonomous assistant
          </p>
        </div>
      </div>

      <div className="flex-1 overflow-y-auto px-3 py-4">
        <p className="mb-2 px-3 font-sans text-[11px] font-medium uppercase tracking-widest text-muted">
          Workspace
        </p>
        <SidebarNav items={mainNavItems} onNavigate={onNavigate} />
      </div>

      <div className="border-t border-border px-5 py-4">
        <p className="font-serif text-xs text-muted leading-relaxed">
          Charcoal &amp; Rust — warm, focused control for your agents.
        </p>
      </div>
    </aside>
  );
}
