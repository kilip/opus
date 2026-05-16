"use client";

import { Bell, Search } from "lucide-react";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { useLogout } from "@/lib/api/auth";

export function Header() {
  const { mutate: logout } = useLogout();

  return (
    <header className="glass sticky top-0 z-40 w-full">
      <div className="flex h-16 items-center justify-between px-8">
        <div className="flex items-center gap-4 flex-1">
          <div className="relative w-96 hidden md:block">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-opus-gray-mid" />
            <input
              type="text"
              placeholder="Search anything..."
              className="w-full bg-opus-gray-light/50 dark:bg-opus-dark/30 border-none rounded-full pl-10 pr-4 py-2 text-sm focus:ring-1 focus:ring-opus-terracotta transition-all font-body"
            />
          </div>
        </div>

        <div className="flex items-center gap-4">
          <Button
            variant="ghost"
            size="icon"
            className="text-opus-gray-mid hover:text-opus-dark dark:hover:text-opus-light relative"
          >
            <Bell className="h-5 w-5" />
            <span className="absolute top-2 right-2 w-2 h-2 bg-opus-terracotta rounded-full border-2 border-opus-light dark:border-opus-dark" />
          </Button>

          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button
                variant="ghost"
                className="flex items-center gap-3 pl-2 pr-1 hover:bg-opus-gray-light dark:hover:bg-opus-dark/50 rounded-full"
              >
                <div className="flex flex-col items-end hidden sm:flex">
                  <span className="text-xs font-heading font-bold text-opus-dark dark:text-opus-light">
                    Pak Bos
                  </span>
                  <span className="text-[10px] font-body text-opus-gray-mid">
                    Administrator
                  </span>
                </div>
                <Avatar className="h-9 w-9 border-2 border-opus-terracotta/20">
                  <AvatarImage src="" />
                  <AvatarFallback className="bg-opus-terracotta/10 text-opus-terracotta font-heading font-bold">
                    PB
                  </AvatarFallback>
                </Avatar>
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" className="w-56 glass mt-2">
              <DropdownMenuLabel className="font-heading">
                My Account
              </DropdownMenuLabel>
              <DropdownMenuSeparator />
              <DropdownMenuItem className="font-body cursor-pointer">
                Profile Settings
              </DropdownMenuItem>
              <DropdownMenuItem className="font-body cursor-pointer">
                Billing
              </DropdownMenuItem>
              <DropdownMenuSeparator />
              <DropdownMenuItem
                className="font-body text-opus-terracotta cursor-pointer focus:bg-opus-terracotta/10 focus:text-opus-terracotta"
                onClick={() => logout()}
              >
                Logout
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </div>
    </header>
  );
}
