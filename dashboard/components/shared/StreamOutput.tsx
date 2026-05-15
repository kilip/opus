"use client";

import { useEffect, useRef } from "react";

interface StreamOutputProps {
  output: string;
  isConnected: boolean;
  error: string | null;
}

export function StreamOutput({ output, isConnected, error }: StreamOutputProps) {
  const scrollRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight;
    }
  }, [output]);

  return (
    <div 
      ref={scrollRef}
      className="absolute inset-0 overflow-y-auto p-4 font-mono text-sm selection:bg-primary/30 scroll-smooth"
    >
      {!output && !error && (
        <div className="h-full flex items-center justify-center text-muted-foreground animate-pulse">
          No output yet. Click Connect to start streaming.
        </div>
      )}
      
      {output && (
        <pre className="whitespace-pre-wrap break-all leading-relaxed">
          {output}
          {isConnected && (
            <span className="inline-block w-2 h-4 ml-1 bg-primary animate-pulse align-middle" />
          )}
        </pre>
      )}

      {error && (
        <div className="mt-4 p-3 rounded bg-destructive/10 border border-destructive/20 text-destructive text-xs">
          <strong>Error:</strong> {error}
        </div>
      )}
    </div>
  );
}
