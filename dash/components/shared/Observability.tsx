/**
 * dash/components/shared/Observability.tsx
 * Client-side component for global event listeners.
 */
"use client";

import { useEffect } from "react";
import { logger } from "@/lib/logger";

export function Observability({ children }: { children: React.ReactNode }) {
  useEffect(() => {
    const handleRejection = (event: PromiseRejectionEvent) => {
      logger.error("Unhandled Promise Rejection", event.reason);
    };

    window.addEventListener("unhandledrejection", handleRejection);
    return () =>
      window.removeEventListener("unhandledrejection", handleRejection);
  }, []);

  return <>{children}</>;
}
