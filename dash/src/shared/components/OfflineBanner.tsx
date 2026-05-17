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
      className="flex items-center justify-center gap-2 bg-amber-500 text-amber-950 px-4 py-2.5 text-center text-sm font-semibold transition-all duration-300 animate-pulse border-b border-amber-600 shadow-sm"
    >
      <WifiOff className="h-4 w-4 shrink-0" aria-hidden="true" />
      <span>
        You are offline. Serving data from cache. Actions will sync when online.
      </span>
    </div>
  );
}
