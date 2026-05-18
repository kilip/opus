import { createElement, type ElementType, type HTMLAttributes } from 'react';
import { cn } from '@/shared/lib/utils';

const headingStyles = {
  1: 'text-[2rem] font-bold',
  2: 'text-2xl font-semibold',
  3: 'text-xl font-medium',
  4: 'text-lg font-medium',
} as const;

export type HeadingProps = HTMLAttributes<HTMLHeadingElement> & {
  level?: 1 | 2 | 3 | 4;
};

/**
 * Semantic heading with brand type scale.
 */
export function Heading({
  level = 2,
  className,
  children,
  ...props
}: HeadingProps) {
  const Tag = `h${level}` as ElementType;
  return createElement(
    Tag,
    {
      className: cn(
        'font-sans tracking-tight text-foreground',
        headingStyles[level],
        className,
      ),
      ...props,
    },
    children,
  );
}

const textVariants = {
  body: 'text-base leading-relaxed text-foreground',
  muted: 'text-sm leading-relaxed text-muted',
  small: 'text-xs leading-normal text-muted',
} as const;

export type TextVariant = keyof typeof textVariants;

export type TextProps = HTMLAttributes<HTMLParagraphElement> & {
  variant?: TextVariant;
  as?: 'p' | 'span' | 'div';
};

/**
 * Body copy using the editorial serif stack.
 */
export function Text({
  variant = 'body',
  as: Component = 'p',
  className,
  ...props
}: TextProps) {
  return (
    <Component
      className={cn('font-serif', textVariants[variant], className)}
      {...props}
    />
  );
}

export type CodeProps = HTMLAttributes<HTMLElement>;

/**
 * Monospace text for logs, IDs, and configuration values.
 */
export function Code({ className, ...props }: CodeProps) {
  return (
    <code
      className={cn(
        'rounded-badge bg-subtle/80 px-1.5 py-0.5 font-mono text-sm text-foreground',
        className,
      )}
      {...props}
    />
  );
}
