import type { ReactNode } from 'react';
import { cn } from '@/shared/lib/utils';
import { Label } from './label';

export type FieldProps = {
  label?: string;
  htmlFor?: string;
  hint?: string;
  error?: string;
  required?: boolean;
  children: ReactNode;
  className?: string;
};

/**
 * Form field wrapper combining label, control, hint, and error text.
 */
export function Field({
  label,
  htmlFor,
  hint,
  error,
  required,
  children,
  className,
}: FieldProps) {
  const hintId = hint ? `${htmlFor}-hint` : undefined;
  const errorId = error ? `${htmlFor}-error` : undefined;

  return (
    <div className={cn('flex flex-col gap-2', className)}>
      {label ? (
        <Label htmlFor={htmlFor} required={required}>
          {label}
        </Label>
      ) : null}
      {children}
      {hint && !error ? (
        <p id={hintId} className="font-serif text-xs text-muted">
          {hint}
        </p>
      ) : null}
      {error ? (
        <p
          id={errorId}
          role="alert"
          className="font-serif text-xs text-brand-primary"
        >
          {error}
        </p>
      ) : null}
    </div>
  );
}
