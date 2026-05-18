import { Check } from 'lucide-react';
import { forwardRef, type InputHTMLAttributes } from 'react';
import { cn } from '@/shared/lib/utils';

export type CheckboxProps = Omit<
  InputHTMLAttributes<HTMLInputElement>,
  'type'
> & {
  label?: string;
};

/**
 * Branded checkbox with optional inline label.
 */
export const Checkbox = forwardRef<HTMLInputElement, CheckboxProps>(
  ({ className, label, id, disabled, ...props }, ref) => {
    const inputId =
      id ??
      (label
        ? `checkbox-${label.replace(/\s+/g, '-').toLowerCase()}`
        : undefined);

    return (
      <label
        htmlFor={inputId}
        className={cn(
          'group inline-flex cursor-pointer items-start gap-2.5',
          disabled && 'cursor-not-allowed opacity-50',
          className,
        )}
      >
        <span className="relative mt-0.5 flex h-4 w-4 shrink-0 items-center justify-center">
          <input
            ref={ref}
            id={inputId}
            type="checkbox"
            disabled={disabled}
            className="peer sr-only"
            {...props}
          />
          <span
            className={cn(
              'flex h-4 w-4 items-center justify-center rounded-badge border border-border bg-card',
              'transition-all duration-200',
              'peer-focus-visible:ring-2 peer-focus-visible:ring-brand-primary/30',
              'peer-checked:border-brand-primary peer-checked:bg-brand-primary',
              'peer-checked:[&_svg]:opacity-100',
              'peer-disabled:opacity-50',
            )}
            aria-hidden
          >
            <Check className="h-2.5 w-2.5 text-brand-light opacity-0 transition-opacity" />
          </span>
        </span>
        {label ? (
          <span className="font-serif text-sm leading-snug text-foreground">
            {label}
          </span>
        ) : null}
      </label>
    );
  },
);
Checkbox.displayName = 'Checkbox';
