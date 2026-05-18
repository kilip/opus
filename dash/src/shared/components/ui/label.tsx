import { forwardRef, type LabelHTMLAttributes } from 'react';
import { cn } from '@/shared/lib/utils';

export type LabelProps = LabelHTMLAttributes<HTMLLabelElement> & {
  required?: boolean;
};

/**
 * Form label with optional required indicator.
 */
export const Label = forwardRef<HTMLLabelElement, LabelProps>(
  ({ className, children, required, ...props }, ref) => (
    // biome-ignore lint/a11y/noLabelWithoutControl: htmlFor is supplied by Field or the consumer
    <label
      ref={ref}
      className={cn(
        'font-sans text-sm font-medium leading-none text-foreground',
        className,
      )}
      {...props}
    >
      {children}
      {required ? (
        <span className="ml-0.5 text-brand-primary" aria-hidden>
          *
        </span>
      ) : null}
    </label>
  ),
);
Label.displayName = 'Label';
