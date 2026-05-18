import { WifiOff } from 'lucide-react';
import { useNetworkStatus } from '../hooks/useNetworkStatus';

/**
 * OfflineBanner component to alert the user when network connectivity is lost.
 * Serves cached data message and sync indication.
 */
export function OfflineBanner() {
  const { isOnline } = useNetworkStatus();

  if (isOnline) return null;

  return (
    <div
      role="alert"
      className="flex items-center justify-center gap-2 border-b border-brand-secondary/30 bg-brand-secondary/15 px-4 py-2.5 text-center font-sans text-sm font-medium text-brand-dark dark:text-brand-light"
    >
      <WifiOff className="h-4 w-4 shrink-0" aria-hidden="true" />
      <span>
        You are offline. Serving data from cache. Actions will sync when online.
      </span>
    </div>
  );
}
