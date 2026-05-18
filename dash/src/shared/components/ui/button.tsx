import { type ButtonHTMLAttributes, forwardRef } from 'react';
import { cn } from '@/shared/lib/utils';

const variantStyles = {
  primary:
    'bg-brand-primary text-brand-light shadow-sm hover:bg-brand-secondary active:scale-[0.98] focus-visible:ring-brand-primary/50',
  secondary:
    'bg-card text-foreground border border-border shadow-sm hover:bg-subtle hover:border-muted',
  ghost: 'text-muted hover:bg-subtle/60 hover:text-foreground',
  outline:
    'border border-border bg-transparent text-foreground hover:bg-subtle/50',
  destructive:
    'bg-brand-primary/10 text-brand-primary border border-brand-primary/20 hover:bg-brand-primary/20',
} as const;

const sizeStyles = {
  sm: 'h-8 px-3 text-xs gap-1.5',
  md: 'h-10 px-4 text-sm gap-2',
  lg: 'h-11 px-5 text-base gap-2',
  icon: 'h-10 w-10 p-0',
} as const;

export type ButtonVariant = keyof typeof variantStyles;
export type ButtonSize = keyof typeof sizeStyles;

export type ButtonProps = ButtonHTMLAttributes<HTMLButtonElement> & {
  variant?: ButtonVariant;
  size?: ButtonSize;
};

/**
 * Branded button with primary, secondary, ghost, and outline variants.
 */
export const Button = forwardRef<HTMLButtonElement, ButtonProps>(
  (
    {
      className,
      variant = 'primary',
      size = 'md',
      type = 'button',
      disabled,
      ...props
    },
    ref,
  ) => (
    <button
      ref={ref}
      type={type}
      disabled={disabled}
      className={cn(
        'inline-flex items-center justify-center rounded-btn font-sans font-medium',
        'transition-all duration-200',
        'disabled:pointer-events-none disabled:opacity-50',
        variantStyles[variant],
        sizeStyles[size],
        className,
      )}
      {...props}
    />
  ),
);
Button.displayName = 'Button';
