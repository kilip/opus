"use client";

import {
  LayoutDashboard,
  MessageSquare,
  Settings,
  Users,
  Workflow,
  Zap,
} from "lucide-react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { cn } from "@/lib/utils";

const navigation = [
  { name: "Dashboard", href: "/dashboard", icon: LayoutDashboard },
  { name: "WhatsApp", href: "/dashboard/whatsapp", icon: MessageSquare },
  { name: "Agents", href: "/dashboard/agents", icon: Workflow },
  { name: "Team", href: "/dashboard/team", icon: Users },
  { name: "Settings", href: "/dashboard/settings", icon: Settings },
];

export function Sidebar() {
  const pathname = usePathname();

  return (
    <aside className="w-64 border-r bg-opus-light/50 dark:bg-opus-dark/50 flex flex-col h-screen sticky top-0">
      <div className="p-6">
        <Link href="/dashboard" className="flex items-center gap-3 group">
          <div className="bg-opus-terracotta text-white p-2 rounded-xl group-hover:rotate-6 transition-transform">
            <Zap className="h-6 w-6 fill-current" />
          </div>
          <span className="font-heading font-bold text-2xl tracking-tight text-opus-dark dark:text-opus-light">
            Opus
          </span>
        </Link>
      </div>

      <nav className="flex-1 px-4 space-y-1">
        {navigation.map((item) => {
          const isActive = pathname === item.href;
          return (
            <Link
              key={item.name}
              href={item.href}
              className={cn(
                "flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm font-medium transition-all group",
                isActive
                  ? "bg-opus-terracotta text-white shadow-sm"
                  : "text-opus-gray-mid hover:text-opus-dark dark:hover:text-opus-light hover:bg-opus-gray-light dark:hover:bg-opus-dark/80",
              )}
            >
              <item.icon
                className={cn(
                  "h-5 w-5",
                  isActive
                    ? "text-white"
                    : "text-opus-gray-mid group-hover:text-opus-dark dark:group-hover:text-opus-light",
                )}
              />
              <span className="font-heading">{item.name}</span>
            </Link>
          );
        })}
      </nav>

      <div className="p-4 border-t">
        <div className="bg-opus-sage/10 dark:bg-opus-sage/20 p-4 rounded-2xl border border-opus-sage/20">
          <p className="text-xs font-heading font-semibold text-opus-sage uppercase tracking-wider mb-1">
            Pro Plan
          </p>
          <p className="text-sm font-body text-opus-dark/70 dark:text-opus-light/70 mb-3">
            Unlock advanced agents and multi-user WhatsApp.
          </p>
          <button
            type="button"
            className="w-full bg-opus-sage text-white py-2 rounded-xl text-xs font-heading font-bold hover:bg-opus-sage/90 transition-colors"
          >
            Upgrade Now
          </button>
        </div>
      </div>
    </aside>
  );
}
