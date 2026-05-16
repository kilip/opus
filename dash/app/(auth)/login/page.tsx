"use client";

import { ArrowRight, GitBranch, Globe, Mail } from "lucide-react";
import Image from "next/image";
import Link from "next/link";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Separator } from "@/components/ui/separator";

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";
const DEV_MODE = process.env.NEXT_PUBLIC_DEV_MODE === "true";

export default function LoginPage() {
  return (
    <div className="flex flex-col lg:flex-row min-h-screen">
      {/* Left Side: Visual & Branding */}
      <div className="relative hidden lg:flex lg:w-1/2 bg-opus-dark flex-col justify-center p-20 overflow-hidden">
        <Image
          src="/login-bg.png"
          alt="Opus background"
          fill
          className="object-cover opacity-50"
          priority
        />
        <div className="absolute inset-0 bg-opus-dark/40" />
        <div className="absolute inset-0 bg-gradient-to-t from-opus-dark/60 via-transparent to-transparent" />

        <div className="relative z-10 max-w-xl">
          <div className="mb-10 inline-flex items-center justify-center h-16 w-16 bg-opus-terracotta rounded-2xl text-opus-light font-black text-3xl shadow-2xl animate-in fade-in zoom-in duration-700">
            O
          </div>
          <h1 className="text-4xl font-heading font-bold text-opus-light mb-8 leading-tight animate-in slide-in-from-bottom duration-700 delay-100">
            Work with focus <br />
            <span className="text-opus-mustard">Grow with Opus</span>
          </h1>
          <p className="text-1xl font-body text-opus-gray-light/90 leading-relaxed animate-in slide-in-from-bottom duration-700 delay-200">
            Your personal AI ecosystem for seamless automation and intelligent
            workflows. Experience a new standard of human-AI collaboration.
          </p>
        </div>
      </div>

      {/* Right Side: Login Form */}
      <div className="flex-1 flex items-center justify-center p-8 bg-opus-light">
        <div className="w-full max-w-[400px] space-y-10 animate-in fade-in duration-1000">
          <div className="lg:hidden flex flex-col items-center text-center space-y-4 mb-12">
            <div className="h-14 w-14 bg-opus-terracotta rounded-2xl flex items-center justify-center text-opus-light font-black text-2xl shadow-xl shadow-opus-terracotta/20">
              O
            </div>
            <h2 className="text-3xl font-heading font-bold text-opus-dark">
              Opus
            </h2>
          </div>

          <div className="space-y-2">
            <h2 className="text-4xl font-heading font-bold text-opus-dark tracking-tight">
              Welcome back
            </h2>
            <p className="text-opus-gray-mid font-body text-lg">
              Choose your preferred sign-in method
            </p>
          </div>

          <div className="grid gap-4">
            <Button
              variant="outline"
              className="h-14 border-2 border-opus-gray-light hover:bg-opus-gray-light hover:text-opus-dark transition-all text-base font-heading font-semibold rounded-xl"
              asChild
            >
              <Link href={`${API_BASE_URL}/auth/google`}>
                <Globe className="mr-3 h-5 w-5 text-opus-mustard" />
                Continue with Google
              </Link>
            </Button>
            <Button
              variant="outline"
              className="h-14 border-2 border-opus-gray-light hover:bg-opus-gray-light hover:text-opus-dark transition-all text-base font-heading font-semibold rounded-xl"
              asChild
            >
              <Link href={`${API_BASE_URL}/auth/github`}>
                <GitBranch className="mr-3 h-5 w-5 text-opus-sage" />
                Continue with GitHub
              </Link>
            </Button>
          </div>

          {DEV_MODE && (
            <div className="space-y-8">
              <div className="relative">
                <div className="absolute inset-0 flex items-center">
                  <Separator className="bg-opus-gray-light" />
                </div>
                <div className="relative flex justify-center text-xs uppercase">
                  <span className="bg-opus-light px-4 text-opus-gray-mid font-bold tracking-widest">
                    Dev Sandbox
                  </span>
                </div>
              </div>

              <form className="space-y-6">
                <div className="space-y-3">
                  <Label
                    htmlFor="email"
                    className="text-sm font-heading font-bold text-opus-dark uppercase tracking-wider ml-1"
                  >
                    Email Address
                  </Label>
                  <Input
                    id="email"
                    type="email"
                    placeholder="Enter your email"
                    className="h-14 bg-opus-gray-light/50 border-transparent focus:border-opus-terracotta focus:ring-opus-terracotta/10 transition-all rounded-xl text-lg px-6"
                  />
                </div>
                <Button
                  type="button"
                  className="w-full h-14 bg-opus-terracotta hover:bg-opus-terracotta/90 text-opus-light font-heading font-bold text-lg rounded-xl shadow-xl shadow-opus-terracotta/20 transition-all group"
                >
                  Sign in with Email
                  <ArrowRight className="ml-2 h-5 w-5 transition-transform group-hover:translate-x-1" />
                </Button>
              </form>
            </div>
          )}

          <p className="text-center text-opus-gray-mid font-body text-sm pt-8">
            By continuing, you agree to our{" "}
            <Link
              href="/terms"
              className="text-opus-dark font-semibold hover:underline"
            >
              Terms of Service
            </Link>{" "}
            and{" "}
            <Link
              href="/privacy"
              className="text-opus-dark font-semibold hover:underline"
            >
              Privacy Policy
            </Link>
            .
          </p>
        </div>
      </div>
    </div>
  );
}
