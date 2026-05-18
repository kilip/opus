import type { ReactNode } from 'react';
import { Heading, Text } from '@/shared/components/ui';
import { cn } from '@/shared/lib/utils';

type DemoSectionProps = {
  id: string;
  title: string;
  description: string;
  children: ReactNode;
  className?: string;
};

/**
 * Anchored section for the component showcase page.
 */
export function DemoSection({
  id,
  title,
  description,
  children,
  className,
}: DemoSectionProps) {
  return (
    <section
      id={id}
      className={cn('scroll-mt-28 space-y-6', className)}
      aria-labelledby={`${id}-title`}
    >
      <header className="space-y-2">
        <Heading level={2} id={`${id}-title`}>
          {title}
        </Heading>
        <Text variant="muted" className="max-w-2xl">
          {description}
        </Text>
      </header>
      {children}
    </section>
  );
}
