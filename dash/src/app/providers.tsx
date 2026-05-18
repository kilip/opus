import { type QueryClient, QueryClientProvider } from '@tanstack/react-query';
import type { ReactNode } from 'react';
import { ThemeProvider } from '@/shared/hooks/useTheme';

type ProvidersProps = {
  children: ReactNode;
  queryClient: QueryClient;
};

/**
 * Root Providers component to wrap the application with necessary state providers.
 * Consistent with ADR-003 and ADR-011.
 */
export function Providers({ children, queryClient }: ProvidersProps) {
  return (
    <ThemeProvider>
      <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
    </ThemeProvider>
  );
}
