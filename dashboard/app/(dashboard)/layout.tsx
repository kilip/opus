"use client";

import { AuthGuard } from "@/components/shared/AuthGuard";
import { Button } from "@/components/ui/button";
import { useLogout } from "@/lib/api/auth";
import { LogOut } from "lucide-react";
import Link from "next/link";

export default function DashboardLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const { mutate: logout } = useLogout();

  return (
    <AuthGuard>
      <div className="min-h-screen flex flex-col">
        <header className="border-b bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60 sticky top-0 z-50">
          <div className="container flex h-14 items-center justify-between">
            <Link href="/" className="flex items-center gap-2 font-bold text-xl tracking-tighter">
              <span className="bg-primary text-primary-foreground px-2 py-0.5 rounded">O</span>
              <span>Opus</span>
            </Link>
            <Button variant="ghost" size="icon" onClick={() => logout()}>
              <LogOut className="h-4 w-4" />
              <span className="sr-only">Logout</span>
            </Button>
          </div>
        </header>
        <main className="flex-1 flex flex-col">
          {children}
        </main>
      </div>
    </AuthGuard>
  );
}
