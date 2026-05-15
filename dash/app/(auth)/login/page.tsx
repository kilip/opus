"use client";

import { GitBranch, Globe, Mail } from "lucide-react";
import Link from "next/link";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Separator } from "@/components/ui/separator";

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";
const DEV_MODE = process.env.NEXT_PUBLIC_DEV_MODE === "true";

export default function LoginPage() {
  return (
    <Card className="border-none shadow-2xl bg-background/60 backdrop-blur-xl">
      <CardHeader className="space-y-4 items-center text-center pb-8">
        <div className="h-12 w-12 bg-primary rounded-xl flex items-center justify-center text-primary-foreground font-black text-2xl shadow-lg shadow-primary/20">
          O
        </div>
        <div className="space-y-1">
          <CardTitle className="text-3xl font-bold tracking-tight">
            Opus
          </CardTitle>
          <CardDescription className="text-base">
            Your 24/7 autonomous AI assistant
          </CardDescription>
        </div>
      </CardHeader>
      <CardContent className="space-y-6">
        <div className="grid grid-cols-2 gap-4">
          <Button variant="outline" className="h-12 font-medium" asChild>
            <Link href={`${API_BASE_URL}/auth/google`}>
              <Globe className="mr-2 h-5 w-5" />
              Google
            </Link>
          </Button>
          <Button variant="outline" className="h-12 font-medium" asChild>
            <Link href={`${API_BASE_URL}/auth/github`}>
              <GitBranch className="mr-2 h-5 w-5" />
              GitHub
            </Link>
          </Button>
        </div>

        {DEV_MODE && (
          <div className="space-y-6">
            <div className="relative">
              <div className="absolute inset-0 flex items-center">
                <Separator />
              </div>
              <div className="relative flex justify-center text-xs uppercase">
                <span className="bg-background px-2 text-muted-foreground font-semibold tracking-wider">
                  Development Only
                </span>
              </div>
            </div>

            <form className="space-y-4">
              <div className="space-y-2">
                <Label
                  htmlFor="email"
                  className="text-xs font-bold uppercase tracking-wider text-muted-foreground ml-1"
                >
                  Email Address
                </Label>
                <Input
                  id="email"
                  type="email"
                  placeholder="name@example.com"
                  className="h-12 bg-muted/30 border-none focus-visible:ring-primary/30"
                />
              </div>
              <Button
                type="button"
                className="w-full h-12 font-bold text-base shadow-lg shadow-primary/20"
              >
                <Mail className="mr-2 h-5 w-5" />
                Sign in with Email
              </Button>
            </form>
          </div>
        )}
      </CardContent>
    </Card>
  );
}
