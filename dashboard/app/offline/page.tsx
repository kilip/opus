"use client";

import { WifiOff } from "lucide-react";
import { Button } from "@/components/ui/button";

export default function OfflinePage() {
  return (
    <div className="flex min-h-screen flex-col items-center justify-center p-6 text-center">
      <div className="mb-4 rounded-full bg-muted p-6">
        <WifiOff className="h-12 w-12 text-muted-foreground" />
      </div>
      <h1 className="text-2xl font-bold tracking-tight">You are offline</h1>
      <p className="mt-2 text-muted-foreground">
        Please check your internet connection and try again.
      </p>
      <div className="mt-6">
        <Button onClick={() => window.location.reload()}>
          Retry Connection
        </Button>
      </div>
    </div>
  );
}
