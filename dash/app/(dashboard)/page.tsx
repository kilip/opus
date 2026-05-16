"use client";

import {
  MessageSquare,
  Play,
  Square,
  Trash2,
  Users,
  Workflow,
  Zap,
} from "lucide-react";
import { useState } from "react";
import { PremiumCard } from "@/components/shared/PremiumCard";
import { StreamOutput } from "@/components/shared/StreamOutput";
import { Button } from "@/components/ui/button";
import { useAuthContext } from "@/lib/api/AuthContext";
import { useCurrentUser } from "@/lib/api/user";
import { useStream } from "@/lib/api/useStream";

export default function DashboardPage() {
  const { accessToken } = useAuthContext();
  const { data: userResponse } = useCurrentUser(accessToken);
  const [shouldConnect, setShouldConnect] = useState(false);
  const { output, isConnected, error, clearOutput } = useStream(
    shouldConnect ? accessToken : null,
  );

  const user = userResponse?.data;

  return (
    <div className="space-y-8 animate-in fade-in duration-700">
      {/* Welcome Section */}
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
        <div>
          <h1 className="font-heading text-3xl font-bold text-opus-dark dark:text-opus-light tracking-tight">
            Selamat Datang Kembali,{" "}
            <span className="text-opus-terracotta">
              {user?.name ?? "Pak Bos"}
            </span>
            !
          </h1>
          <p className="font-body text-opus-gray-mid italic mt-1">
            Opus is ready to assist you today. Everything looks good.
          </p>
        </div>
        <div className="flex items-center gap-3">
          <div className="flex -space-x-2">
            {[1, 2, 3].map((i) => (
              <div
                key={i}
                className="h-8 w-8 rounded-full border-2 border-opus-light dark:border-opus-dark bg-opus-gray-light dark:bg-opus-dark flex items-center justify-center overflow-hidden"
              >
                <div className="bg-opus-sage/20 text-opus-sage text-[10px] font-bold">
                  AG
                </div>
              </div>
            ))}
          </div>
          <span className="text-xs font-heading font-semibold text-opus-gray-mid">
            3 Agents Active
          </span>
        </div>
      </div>

      {/* Stats/Quick Actions Grid */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        <PremiumCard
          title="WhatsApp Hub"
          description="Active sessions and chats"
          className="bg-opus-sage/5 border-opus-sage/10"
        >
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3 text-opus-sage">
              <MessageSquare className="h-5 w-5" />
              <span className="text-2xl font-heading font-bold">12</span>
            </div>
            <Button
              size="sm"
              variant="ghost"
              className="text-opus-sage hover:bg-opus-sage/10 font-heading text-xs"
            >
              Manage
            </Button>
          </div>
        </PremiumCard>

        <PremiumCard
          title="AI Task Queue"
          description="Pending agent operations"
          className="bg-opus-mustard/5 border-opus-mustard/10"
        >
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3 text-opus-mustard">
              <Workflow className="h-5 w-5" />
              <span className="text-2xl font-heading font-bold">5</span>
            </div>
            <Button
              size="sm"
              variant="ghost"
              className="text-opus-mustard hover:bg-opus-mustard/10 font-heading text-xs"
            >
              View Queue
            </Button>
          </div>
        </PremiumCard>

        <PremiumCard
          title="Workspace"
          description="Total team members"
          className="bg-opus-terracotta/5 border-opus-terracotta/10"
        >
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3 text-opus-terracotta">
              <Users className="h-5 w-5" />
              <span className="text-2xl font-heading font-bold">8</span>
            </div>
            <Button
              size="sm"
              variant="ghost"
              className="text-opus-terracotta hover:bg-opus-terracotta/10 font-heading text-xs"
            >
              Invite
            </Button>
          </div>
        </PremiumCard>
      </div>

      {/* Intelligence Stream Section */}
      <PremiumCard
        gradient
        className="flex flex-col border-none"
        title="Intelligence Stream"
        description="Real-time monitoring of autonomous agent activities"
        footer={
          <div className="flex items-center justify-between w-full">
            <div className="flex items-center gap-4">
              <div className="flex items-center gap-2">
                <div
                  className={`h-2.5 w-2.5 rounded-full ${isConnected ? "bg-green-500 animate-pulse" : "bg-red-500"}`}
                />
                <span className="text-xs font-heading font-bold uppercase tracking-wider text-opus-gray-mid">
                  {isConnected ? "Connection Stable" : "Disconnected"}
                </span>
              </div>
              {error && (
                <span className="text-xs text-opus-terracotta font-body italic">
                  Error: {error}
                </span>
              )}
            </div>
            <div className="flex gap-2">
              <Button
                variant="ghost"
                size="sm"
                onClick={clearOutput}
                disabled={!output}
                className="font-heading text-xs gap-2 hover:bg-opus-terracotta/10 hover:text-opus-terracotta"
              >
                <Trash2 className="h-3.5 w-3.5" />
                Clear
              </Button>
              {!shouldConnect ? (
                <Button
                  onClick={() => setShouldConnect(true)}
                  size="sm"
                  className="bg-opus-terracotta text-white font-heading text-xs gap-2 shadow-lg shadow-opus-terracotta/20 hover:scale-105 transition-transform"
                >
                  <Play className="h-3.5 w-3.5 fill-current" />
                  Connect
                </Button>
              ) : (
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setShouldConnect(false)}
                  className="font-heading text-xs gap-2 border-opus-terracotta text-opus-terracotta hover:bg-opus-terracotta/5"
                >
                  <Square className="h-3.5 w-3.5 fill-current" />
                  Stop
                </Button>
              )}
            </div>
          </div>
        }
      >
        <div className="relative min-h-[450px] bg-opus-dark/5 dark:bg-black/20 rounded-2xl overflow-hidden border border-opus-gray-mid/10">
          <StreamOutput
            output={output}
            isConnected={isConnected}
            error={error}
          />
          {!isConnected && !output && (
            <div className="absolute inset-0 flex flex-col items-center justify-center text-center p-8">
              <div className="bg-opus-light/50 dark:bg-opus-dark/50 p-6 rounded-full mb-4 animate-bounce">
                <Zap className="h-10 w-10 text-opus-gray-mid/30" />
              </div>
              <h3 className="font-heading text-lg font-bold text-opus-gray-mid">
                Ready to Initialize
              </h3>
              <p className="font-body text-sm text-opus-gray-mid/70 max-w-xs mt-2">
                Click the connect button to start receiving live updates from
                your agents.
              </p>
            </div>
          )}
        </div>
      </PremiumCard>
    </div>
  );
}
