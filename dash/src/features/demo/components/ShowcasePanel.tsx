import type { ReactNode } from 'react';
import { cn } from '@/shared/lib/utils';

type ShowcasePanelProps = {
  label: string;
  children: ReactNode;
  className?: string;
  compact?: boolean;
};

/**
 * Preview surface for a single component variant on the demo page.
 */
export function ShowcasePanel({
  label,
  children,
  className,
  compact = false,
}: ShowcasePanelProps) {
  return (
    <figure
      className={cn(
        'group overflow-hidden rounded-card border border-border bg-card shadow-card',
        'transition-all duration-200 hover:border-muted hover:shadow-card-hover',
        className,
      )}
    >
      <figcaption className="border-b border-border bg-subtle/40 px-4 py-2">
        <span className="font-mono text-xs tracking-wide text-muted uppercase">
          {label}
        </span>
      </figcaption>
      <div
        className={cn(
          'flex flex-wrap items-center gap-3',
          compact ? 'p-4' : 'p-6',
        )}
      >
        {children}
      </div>
    </figure>
  );
}
