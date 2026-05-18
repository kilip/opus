import type { HTMLAttributes, ReactNode } from 'react';
import { cn } from '@/shared/lib/utils';

const variantStyles = {
  info: 'border-border bg-subtle/50 text-foreground',
  success:
    'border-brand-success/25 bg-brand-success/10 text-brand-success [&_p]:text-brand-success/90',
  warning:
    'border-brand-secondary/25 bg-brand-secondary/10 text-brand-secondary [&_p]:text-brand-secondary/90',
  error:
    'border-brand-primary/25 bg-brand-primary/10 text-brand-primary [&_p]:text-brand-primary/90',
} as const;

export type AlertVariant = keyof typeof variantStyles;

export type AlertProps = HTMLAttributes<HTMLDivElement> & {
  variant?: AlertVariant;
  title?: string;
  icon?: ReactNode;
};

/**
 * Inline alert for status, warnings, and contextual messages.
 */
export function Alert({
  className,
  variant = 'info',
  title,
  icon,
  children,
  ...props
}: AlertProps) {
  return (
    <div
      role="alert"
      className={cn(
        'flex gap-3 rounded-card border p-4',
        'font-serif text-sm leading-relaxed',
        variantStyles[variant],
        className,
      )}
      {...props}
    >
      {icon ? <div className="shrink-0 pt-0.5">{icon}</div> : null}
      <div className="min-w-0 flex-1 space-y-1">
        {title ? (
          <p className="font-sans text-sm font-semibold text-inherit">
            {title}
          </p>
        ) : null}
        {children ? <div>{children}</div> : null}
      </div>
    </div>
  );
}
