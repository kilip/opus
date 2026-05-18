import type { HTMLAttributes } from 'react';
import { cn } from '@/shared/lib/utils';

const variantStyles = {
  success: 'bg-brand-success/10 text-brand-success border-brand-success/20',
  alert: 'bg-brand-primary/10 text-brand-primary border-brand-primary/20',
  muted: 'bg-subtle/80 text-muted border-border',
  outline: 'bg-transparent text-foreground border-border',
} as const;

export type BadgeVariant = keyof typeof variantStyles;

export type BadgeProps = HTMLAttributes<HTMLSpanElement> & {
  variant?: BadgeVariant;
};

/**
 * Compact status label for online, alert, and neutral states.
 */
export function Badge({ className, variant = 'muted', ...props }: BadgeProps) {
  return (
    <span
      className={cn(
        'inline-flex items-center rounded-badge border px-2 py-0.5',
        'font-sans text-xs font-medium',
        variantStyles[variant],
        className,
      )}
      {...props}
    />
  );
}
