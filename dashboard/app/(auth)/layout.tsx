"use client";

import { useAuthContext } from "@/lib/api/AuthContext";
import { useRouter } from "next/navigation";
import { useEffect } from "react";

export default function AuthLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const { accessToken } = useAuthContext();
  const router = useRouter();

  useEffect(() => {
    if (accessToken) {
      router.push("/");
    }
  }, [accessToken, router]);

  return (
    <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-background via-background to-primary/5 p-4">
      <div className="w-full max-w-md">
        {children}
      </div>
    </div>
  );
}
