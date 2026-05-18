import { Menu } from 'lucide-react';
import { ThemeToggle } from '@/shared/components/ThemeToggle';
import { Button } from '@/shared/components/ui/button';
import { cn } from '@/shared/lib/utils';

type DashboardHeaderProps = {
  onMenuOpen: () => void;
  className?: string;
};

/**
 * Sticky top bar with mobile menu control and theme toggle.
 */
export function DashboardHeader({
  onMenuOpen,
  className,
}: DashboardHeaderProps) {
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

      <div className="flex items-center gap-2">
        <ThemeToggle />
      </div>
    </header>
  );
}
