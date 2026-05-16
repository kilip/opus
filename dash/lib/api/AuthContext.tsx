"use client";
import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useState,
} from "react";
import { apiClient, setAuthToken } from "./client";
import type { AuthResponse } from "./types";

interface AuthContextValue {
  accessToken: string | null;
  setAccessToken: (token: string | null) => void;
  isLoading: boolean;
}

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [accessToken, _setAccessToken] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  const setAccessToken = useCallback((token: string | null) => {
    _setAccessToken(token);
    setAuthToken(token);
  }, []);

  // Silent Refresh on mount to satisfy ARCHITECTURE.md (Memory JS + HttpOnly Cookie)
  useEffect(() => {
    const performSilentRefresh = async () => {
      try {
        const response = await apiClient.post<AuthResponse>("/auth/refresh");
        if (response.success && response.data) {
          setAccessToken(response.data.accessToken);
        }
      } catch (error) {
        console.error("Silent refresh error:", error);
      } finally {
        setIsLoading(false);
      }
    };

    performSilentRefresh();
  }, [setAccessToken]);

  return (
    <AuthContext.Provider
      value={{
        accessToken,
        setAccessToken,
        isLoading,
      }}
    >
      {children}
    </AuthContext.Provider>
  );
}

export function useAuthContext() {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuthContext must be used within AuthProvider");
  return ctx;
}
