import { ChevronDown } from 'lucide-react';
import { forwardRef, type SelectHTMLAttributes } from 'react';
import { cn } from '@/shared/lib/utils';

export type SelectProps = SelectHTMLAttributes<HTMLSelectElement>;

/**
 * Native select styled with brand tokens and a custom chevron.
 */
export const Select = forwardRef<HTMLSelectElement, SelectProps>(
  ({ className, children, ...props }, ref) => (
    <div className="relative w-full">
      <select
        ref={ref}
        className={cn(
          'flex h-10 w-full appearance-none rounded-btn border border-border bg-card',
          'px-3 py-2 pr-9 font-serif text-sm text-foreground',
          'transition-all duration-200',
          'focus:border-brand-primary focus:outline-none focus:ring-2 focus:ring-brand-primary/20',
          'disabled:cursor-not-allowed disabled:opacity-50',
          className,
        )}
        {...props}
      >
        {children}
      </select>
      <ChevronDown
        className="pointer-events-none absolute top-1/2 right-2.5 h-4 w-4 -translate-y-1/2 text-muted"
        aria-hidden
      />
    </div>
  ),
);
Select.displayName = 'Select';
