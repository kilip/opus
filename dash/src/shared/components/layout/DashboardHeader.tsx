import { useNavigate, useRouteContext } from '@tanstack/react-router';
import { LogOut, Menu, User } from 'lucide-react';
import { logout } from '@/features/auth/api';
import { ThemeToggle } from '@/shared/components/ThemeToggle';
import { Button } from '@/shared/components/ui/button';
import { cn } from '@/shared/lib/utils';

type DashboardHeaderProps = {
  onMenuOpen: () => void;
  className?: string;
};

/**
 * Sticky top bar with mobile menu control, theme toggle, and user actions.
 */
export function DashboardHeader({
  onMenuOpen,
  className,
}: DashboardHeaderProps) {
  const navigate = useNavigate();
  const context = useRouteContext({ from: '__root__' });
  const user = context.user;

  const handleLogout = async () => {
    await logout();
    navigate({ to: '/login' });
  };

  return (
    <header
      className={cn(
        'sticky top-0 z-40 flex h-14 shrink-0 items-center justify-between gap-4',
        'border-b border-border bg-background/90 px-4 backdrop-blur-md sm:px-6',
        'lg:h-16',
        className,
      )}
    >
      <div className="flex items-center gap-3 min-w-0">
        <Button
          type="button"
          variant="ghost"
          size="icon"
          className="lg:hidden"
          onClick={onMenuOpen}
          aria-label="Open navigation menu"
        >
          <Menu className="h-5 w-5" aria-hidden />
        </Button>
        <div className="lg:hidden min-w-0">
          <p className="font-sans text-sm font-semibold truncate">Opus</p>
        </div>
      </div>

      <div className="flex items-center gap-3">
        <ThemeToggle />

        <div className="h-6 w-px bg-border mx-1 hidden sm:block" />

        <div className="flex items-center gap-3 pl-1">
          <div className="hidden sm:block text-right">
            <p className="font-sans text-xs font-semibold leading-none">
              {user?.name || 'User'}
            </p>
            <p className="font-serif text-[10px] text-muted leading-tight mt-1 capitalize">
              {user?.role || 'Guest'}
            </p>
          </div>

          <div className="h-8 w-8 rounded-full bg-brand-subtle flex items-center justify-center border border-border overflow-hidden">
            {user?.avatar_url ? (
              <img
                src={user.avatar_url}
                alt=""
                className="h-full w-full object-cover"
              />
            ) : (
              <User className="h-4 w-4 text-muted" />
            )}
          </div>

          <Button
            variant="ghost"
            size="icon"
            className="h-8 w-8 text-muted hover:text-brand-primary"
            onClick={handleLogout}
            title="Sign out"
          >
            <LogOut className="h-4 w-4" />
          </Button>
        </div>
      </div>
    </header>
  );
}
