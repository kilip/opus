"use client";
import { useEffect, useState, useCallback } from "react";

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

export function useStream(accessToken: string | null) {
  const [output, setOutput] = useState<string>("");
  const [isConnected, setIsConnected] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const clearOutput = useCallback(() => setOutput(""), []);

  useEffect(() => {
    if (!accessToken) return;

    // EventSource does not support custom headers
    // Pass token as query param (acceptable for SSE — token is short-lived)
    const url = `${API_BASE_URL}/stream?token=${encodeURIComponent(accessToken)}`;
    const es = new EventSource(url, { withCredentials: true });

    es.onopen = () => setIsConnected(true);
    es.onmessage = (e) => setOutput((prev) => prev + e.data);
    es.onerror = () => {
      setError("Stream connection lost");
      setIsConnected(false);
      es.close();
    };

    return () => {
      es.close();
      setIsConnected(false);
    };
  }, [accessToken]);

  return { output, isConnected, error, clearOutput };
}
