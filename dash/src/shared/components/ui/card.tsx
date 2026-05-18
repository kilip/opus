import { forwardRef, type HTMLAttributes } from 'react';
import { cn } from '@/shared/lib/utils';

export type CardProps = HTMLAttributes<HTMLDivElement> & {
  interactive?: boolean;
};

/**
 * Content container with editorial card styling per brand guidelines.
 */
export const Card = forwardRef<HTMLDivElement, CardProps>(
  ({ className, interactive = false, children, ...props }, ref) => (
    <div
      ref={ref}
      className={cn(
        'rounded-card border border-border bg-card text-card-foreground shadow-card',
        interactive &&
          'transition-all duration-200 hover:-translate-y-0.5 hover:border-muted hover:shadow-card-hover',
        className,
      )}
      {...props}
    >
      {children}
    </div>
  ),
);
Card.displayName = 'Card';

/**
 * CardHeader groups title and description at the top of a card.
 */
export function CardHeader({
  className,
  ...props
}: HTMLAttributes<HTMLDivElement>) {
  return (
    <div
      className={cn('flex flex-col gap-1.5 p-6 pb-0', className)}
      {...props}
    />
  );
}

/**
 * CardTitle renders a semibold sans heading for card sections.
 */
export function CardTitle({
  className,
  ...props
}: HTMLAttributes<HTMLHeadingElement>) {
  return (
    <h3
      className={cn('font-sans text-lg font-semibold leading-none', className)}
      {...props}
    />
  );
}

/**
 * CardDescription renders muted serif supporting text.
 */
export function CardDescription({
  className,
  ...props
}: HTMLAttributes<HTMLParagraphElement>) {
  return (
    <p className={cn('font-serif text-sm text-muted', className)} {...props} />
  );
}

/**
 * CardContent holds the main card body below the header.
 */
export function CardContent({
  className,
  ...props
}: HTMLAttributes<HTMLDivElement>) {
  return <div className={cn('p-6', className)} {...props} />;
}

/**
 * CardFooter aligns actions at the bottom of a card.
 */
export function CardFooter({
  className,
  ...props
}: HTMLAttributes<HTMLDivElement>) {
  return (
    <div
      className={cn('flex items-center gap-3 p-6 pt-0', className)}
      {...props}
    />
  );
}
