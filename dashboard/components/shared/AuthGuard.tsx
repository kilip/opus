"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { useAuthContext } from "@/lib/api/AuthContext";
import { useRefreshToken } from "@/lib/api/auth";

export function AuthGuard({ children }: { children: React.ReactNode }) {
  const router = useRouter();
  const { accessToken, setAccessToken } = useAuthContext();
  const { mutate: refresh, isPending } = useRefreshToken();

  useEffect(() => {
    if (!accessToken) {
      // Attempt silent refresh via HttpOnly cookie
      refresh(undefined, {
        onSuccess: (data) => {
          if (data.success && data.data) {
            setAccessToken(data.data.accessToken);
          } else {
            router.push("/login");
          }
        },
        onError: () => router.push("/login"),
      });
    }
  }, [accessToken, refresh, setAccessToken, router]);

  if (isPending || !accessToken) {
    return (
      <div className="flex h-screen items-center justify-center">
        <div className="flex flex-col items-center gap-4">
          <div className="h-8 w-8 animate-spin rounded-full border-4 border-primary border-t-transparent" />
          <p className="text-sm text-muted-foreground animate-pulse">
            Verifying your session...
          </p>
        </div>
      </div>
    );
  }

  return <>{children}</>;
}
