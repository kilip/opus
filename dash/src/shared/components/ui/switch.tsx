import { type ButtonHTMLAttributes, forwardRef } from 'react';
import { cn } from '@/shared/lib/utils';

export type SwitchProps = Omit<
  ButtonHTMLAttributes<HTMLButtonElement>,
  'role' | 'type'
> & {
  checked?: boolean;
  onCheckedChange?: (checked: boolean) => void;
  label?: string;
};

/**
 * Toggle switch for boolean settings.
 */
export const Switch = forwardRef<HTMLButtonElement, SwitchProps>(
  (
    {
      className,
      checked = false,
      onCheckedChange,
      label,
      disabled,
      id,
      onClick,
      ...props
    },
    ref,
  ) => {
    const switchId =
      id ??
      (label
        ? `switch-${label.replace(/\s+/g, '-').toLowerCase()}`
        : undefined);

    const toggle = () => {
      if (!disabled) {
        onCheckedChange?.(!checked);
      }
    };

    return (
      <div className={cn('inline-flex items-center gap-3', className)}>
        <button
          ref={ref}
          id={switchId}
          type="button"
          role="switch"
          aria-checked={checked}
          aria-label={label}
          disabled={disabled}
          onClick={(e) => {
            onClick?.(e);
            toggle();
          }}
          className={cn(
            'relative inline-flex h-6 w-11 shrink-0 rounded-full border border-border',
            'transition-all duration-200',
            'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-primary/40',
            'disabled:cursor-not-allowed disabled:opacity-50',
            checked ? 'bg-brand-primary border-brand-primary' : 'bg-subtle',
          )}
          {...props}
        >
          <span
            className={cn(
              'pointer-events-none absolute top-0.5 left-0.5 h-4 w-4 rounded-full bg-card shadow-sm',
              'transition-transform duration-200',
              checked && 'translate-x-5',
            )}
          />
        </button>
        {label ? (
          <label
            htmlFor={switchId}
            className="cursor-pointer font-serif text-sm text-foreground"
          >
            {label}
          </label>
        ) : null}
      </div>
    );
  },
);
Switch.displayName = 'Switch';
