import { Monitor, Moon, Sun } from 'lucide-react';
import { Button } from '@/shared/components/ui/button';
import { type ThemePreference, useTheme } from '@/shared/hooks/useTheme';
import { cn } from '@/shared/lib/utils';

const cycle: ThemePreference[] = ['light', 'dark', 'system'];

const labels: Record<ThemePreference, string> = {
  light: 'Light theme',
  dark: 'Dark theme',
  system: 'System theme',
};

/**
 * Cycles light, dark, and system theme preferences on the document root.
 */
export function ThemeToggle({ className }: { className?: string }) {
  const { preference, setPreference } = useTheme();

  const next = () => {
    const index = cycle.indexOf(preference);
    const following = cycle[(index + 1) % cycle.length];
    setPreference(following);
  };

  const Icon =
    preference === 'light' ? Sun : preference === 'dark' ? Moon : Monitor;

  return (
    <Button
      type="button"
      variant="ghost"
      size="icon"
      className={cn('shrink-0', className)}
      onClick={next}
      aria-label={labels[preference]}
      title={labels[preference]}
    >
      <Icon className="h-4 w-4" aria-hidden />
    </Button>
  );
}
