import { forwardRef, type TextareaHTMLAttributes } from 'react';
import { cn } from '@/shared/lib/utils';

export type TextareaProps = TextareaHTMLAttributes<HTMLTextAreaElement>;

/**
 * Multi-line text input with editorial serif styling.
 */
export const Textarea = forwardRef<HTMLTextAreaElement, TextareaProps>(
  ({ className, ...props }, ref) => (
    <textarea
      ref={ref}
      className={cn(
        'flex min-h-[100px] w-full resize-y rounded-btn border border-border bg-card px-3 py-2',
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
Textarea.displayName = 'Textarea';
