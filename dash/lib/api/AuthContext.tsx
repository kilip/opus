"use client";
import { usePathname, useRouter } from "next/navigation";
import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useState,
} from "react";
import { logger } from "@/lib/logger";
import { apiClient, setAuthToken, setOnUnauthorized } from "./client";
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
  const router = useRouter();
  const pathname = usePathname();

  const setAccessToken = useCallback((token: string | null) => {
    _setAccessToken(token);
    setAuthToken(token);
  }, []);

  // Register the unauthorized handler
  useEffect(() => {
    setOnUnauthorized(() => {
      setAccessToken(null);
      router.push("/login");
    });
  }, [setAccessToken, router]);

  // Silent Refresh on mount to satisfy ARCHITECTURE.md (Memory JS + HttpOnly Cookie)
  useEffect(() => {
    // Jangan refresh kalau di halaman login
    if (pathname === "/login") {
      setIsLoading(false);
      return;
    }

    const performSilentRefresh = async () => {
      try {
        const response = await apiClient.post<AuthResponse>("/auth/refresh");
        if (response.success && response.data) {
          setAccessToken(response.data.accessToken);
        } else {
          throw new Error(JSON.stringify(response.error));
        }
      } catch (error) {
        let errorMessage = String(error);
        if (error instanceof Error) {
          errorMessage = error.message;
        }

        // Cek apakah error disebabkan oleh session expired/unauthorized
        let isUnauthorized = false;
        try {
          const errObj = JSON.parse(errorMessage);
          if (
            errObj?.code === "UNAUTHORIZED" ||
            errObj?.message?.includes("expired")
          ) {
            isUnauthorized = true;
          }
        } catch (_e) {
          // Fallback: cek jika string message mengandung UNAUTHORIZED
          if (
            errorMessage.includes("UNAUTHORIZED") ||
            errorMessage.includes("expired")
          ) {
            isUnauthorized = true;
          }
        }

        if (isUnauthorized) {
          // Panggil logout untuk hapus cookie HttpOnly di backend
          apiClient.post("/auth/logout").catch(() => {});
          setAccessToken(null);
          // Paksa hard redirect agar middleware mendeteksi ketiadaan cookie
          window.location.href = "/login";
        } else {
          logger.error("Silent refresh error", errorMessage);
        }
      } finally {
        setIsLoading(false);
      }
    };

    performSilentRefresh();
  }, [setAccessToken, pathname]);

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
