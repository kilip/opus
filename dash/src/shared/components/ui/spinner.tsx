import type { HTMLAttributes } from 'react';
import { cn } from '@/shared/lib/utils';

const sizeStyles = {
  sm: 'h-4 w-4 border-2',
  md: 'h-6 w-6 border-2',
  lg: 'h-8 w-8 border-[3px]',
} as const;

export type SpinnerSize = keyof typeof sizeStyles;

export type SpinnerProps = HTMLAttributes<HTMLDivElement> & {
  size?: SpinnerSize;
  label?: string;
};

/**
 * Indeterminate loading spinner with brand accent.
 */
export function Spinner({
  className,
  size = 'md',
  label = 'Loading',
  ...props
}: SpinnerProps) {
  return (
    <div
      role="status"
      aria-label={label}
      className={cn('inline-flex items-center justify-center', className)}
      {...props}
    >
      <span
        className={cn(
          'animate-spin rounded-full border-border border-t-brand-primary',
          sizeStyles[size],
        )}
      />
      <span className="sr-only">{label}</span>
    </div>
  );
}
