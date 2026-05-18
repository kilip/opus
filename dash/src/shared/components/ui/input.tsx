import { forwardRef, type InputHTMLAttributes } from 'react';
import { cn } from '@/shared/lib/utils';

export type InputProps = InputHTMLAttributes<HTMLInputElement>;

/**
 * Standard text input with editorial serif styling and brand focus ring.
 */
export const Input = forwardRef<HTMLInputElement, InputProps>(
  ({ className, type = 'text', ...props }, ref) => (
    <input
      ref={ref}
      type={type}
      className={cn(
        'flex h-10 w-full rounded-btn border border-border bg-card px-3 py-2',
        'font-serif text-sm text-foreground',
        'placeholder:text-muted',
        'transition-all duration-200',
        'focus:border-brand-primary focus:outline-none focus:ring-2 focus:ring-brand-primary/20',
        'disabled:cursor-not-allowed disabled:opacity-50',
        className,
      )}
      {...props}
    />
  ),
);
Input.displayName = 'Input';
