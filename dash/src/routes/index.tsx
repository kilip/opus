import { createFileRoute, redirect } from '@tanstack/react-router';

/**
 * Route definition for the root index endpoint.
 * Redirects immediately to the feature dashboard (`/agent`).
 */
export const Route = createFileRoute('/')({
  beforeLoad: () => {
    throw redirect({
      to: '/agent',
    });
  },
});
