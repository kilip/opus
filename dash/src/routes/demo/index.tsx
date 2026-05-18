import { createFileRoute } from '@tanstack/react-router';
import { DemoPage } from '@/features/demo/components/DemoPage';

/**
 * Route definition for the design system showcase at /demo.
 */
export const Route = createFileRoute('/demo/')({
  component: DemoPage,
});
