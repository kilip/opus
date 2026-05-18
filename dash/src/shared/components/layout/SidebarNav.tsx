import { Link, useRouterState } from '@tanstack/react-router';
import type { NavItem } from '@/shared/components/layout/nav-config';
import { cn } from '@/shared/lib/utils';

type SidebarNavProps = {
  items: NavItem[];
  onNavigate?: () => void;
  className?: string;
};

/**
 * Vertical navigation list for dashboard sidebar and mobile drawer.
 */
export function SidebarNav({ items, onNavigate, className }: SidebarNavProps) {
  const pathname = useRouterState({ select: (s) => s.location.pathname });

  return (
    <nav className={cn('flex flex-col gap-1', className)} aria-label="Main">
      {items.map((item) => {
        const isActive =
          pathname === item.to || pathname.startsWith(`${item.to}/`);
        const Icon = item.icon;

        if (item.disabled) {
          return (
            <span
              key={item.to}
              className="flex items-center gap-3 rounded-btn px-3 py-2.5 font-sans text-sm text-muted/60 cursor-not-allowed"
              aria-disabled
            >
              <Icon className="h-4 w-4 shrink-0 opacity-50" aria-hidden />
              {item.label}
              <span className="ml-auto text-[10px] uppercase tracking-wider text-muted/50">
                Soon
              </span>
            </span>
          );
        }

        return (
          <Link
            key={item.to}
            to={item.to}
            onClick={onNavigate}
            className={cn(
              'flex items-center gap-3 rounded-btn px-3 py-2.5 font-sans text-sm transition-all duration-200',
              isActive
                ? 'bg-brand-primary text-brand-light shadow-sm'
                : 'text-sidebar-foreground/80 hover:bg-subtle/80 hover:text-sidebar-foreground',
            )}
            aria-current={isActive ? 'page' : undefined}
          >
            <Icon className="h-4 w-4 shrink-0" aria-hidden />
            {item.label}
          </Link>
        );
      })}
    </nav>
  );
}
