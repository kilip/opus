"use client";

import {
  AlertCircle,
  CheckCircle2,
  Loader2,
  LogOut,
  PlusCircle,
  Settings2,
  Smartphone,
  Unplug,
} from "lucide-react";
import Image from "next/image";
import { useCallback, useEffect, useState } from "react";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { useAuthContext } from "@/lib/api/AuthContext";
import { apiClient } from "@/lib/api/client";
import { logger } from "@/lib/logger";

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

type ConnectionStatus =
  | "LOADING"
  | "UNAUTHENTICATED"
  | "PAIRING"
  | "CONNECTED"
  | "DISCONNECTED"
  | "ERROR";

export default function WhatsAppSettingsPage() {
  const { accessToken } = useAuthContext();
  const [status, setStatus] = useState<ConnectionStatus>("LOADING");
  const [jid, setJid] = useState("");
  const [qrCode, setQrCode] = useState("");
  const [isLoading, setIsLoading] = useState(false);

  const fetchStatus = useCallback(async () => {
    try {
      const res = await apiClient.get<{ status: string; jid: string }>(
        "/whatsapp/status",
      );
      if (res.success && res.data) {
        setStatus((res.data.status as ConnectionStatus) || "UNAUTHENTICATED");
        setJid(res.data.jid || "");
      }
    } catch (_err) {
      setStatus("ERROR");
    }
  }, []);

  useEffect(() => {
    if (accessToken) {
      fetchStatus();
    }
  }, [accessToken, fetchStatus]);

  useEffect(() => {
    if (!accessToken) return;

    const url = `${API_BASE_URL}/stream?token=${encodeURIComponent(accessToken)}`;
    const eventSource = new EventSource(url);

    eventSource.addEventListener("wa_qr_update", (e: MessageEvent) => {
      try {
        const data = JSON.parse(e.data);
        setQrCode(data.qr);
        setStatus("PAIRING");
      } catch (err) {
        logger.error("Failed to parse QR update", err);
      }
    });

    eventSource.addEventListener("wa_qr_event", (e: MessageEvent) => {
      try {
        const data = JSON.parse(e.data);
        if (data.event === "timeout" || data.event === "expired") {
          // Clear old QR to show loading state while waiting for next one
          setQrCode("");
        }
      } catch (err) {
        logger.error("Failed to parse QR event", err);
      }
    });

    eventSource.addEventListener("wa_connected", () => {
      setStatus("CONNECTED");
      setQrCode("");
      fetchStatus();
    });

    eventSource.addEventListener("wa_disconnected", () => {
      setStatus("DISCONNECTED");
      setJid("");
      setQrCode("");
    });

    return () => eventSource.close();
  }, [fetchStatus, accessToken]);

  const handleConnect = async () => {
    setIsLoading(true);
    try {
      const res = await apiClient.post("/whatsapp/connect");
      if (res.success) {
        // Give immediate feedback that we are waiting for QR
        setStatus("PAIRING");
      }
    } catch (err) {
      logger.error("Failed to initiate connection", err);
      setStatus("ERROR");
    } finally {
      setIsLoading(false);
    }
  };

  const handleDisconnect = async () => {
    setIsLoading(true);
    try {
      const res = await apiClient.post("/whatsapp/disconnect");
      if (res.success) {
        setStatus("UNAUTHENTICATED");
        setJid("");
        setQrCode("");
      }
    } catch (err) {
      logger.error("Failed to disconnect", err);
      setStatus("ERROR");
    } finally {
      setIsLoading(false);
    }
  };

  if (status === "LOADING") {
    return <SettingsSkeleton />;
  }

  return (
    <div className="max-w-4xl mx-auto space-y-8 animate-in fade-in duration-500 p-6 lg:p-8">
      <div className="flex flex-col gap-2">
        <h1 className="text-4xl font-heading font-bold text-[#1C1917] dark:text-[#FFFCF5]">
          WhatsApp Settings
        </h1>
        <p className="text-lg font-body text-stone-500 dark:text-stone-400">
          Manage your WhatsApp bridge connection and automation.
        </p>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-8">
        <Card className="md:col-span-2 border-stone-200 dark:border-stone-800 shadow-sm overflow-hidden bg-white dark:bg-stone-900/50">
          <CardHeader className="border-b border-stone-100 dark:border-stone-800 bg-stone-50/50 dark:bg-stone-900/30">
            <div className="flex items-center justify-between">
              <div className="space-y-1">
                <CardTitle className="flex items-center gap-2 font-heading text-xl">
                  <Smartphone className="h-5 w-5 text-[#C53030]" />
                  Connection Hub
                </CardTitle>
                <CardDescription className="font-body">
                  Current instance status and control.
                </CardDescription>
              </div>
              <StatusBadge status={status} />
            </div>
          </CardHeader>

          <CardContent className="p-8">
            {status === "CONNECTED" ? (
              <div className="space-y-6">
                <div className="flex items-center gap-4 p-4 rounded-xl bg-green-50 dark:bg-green-900/10 border border-green-100 dark:border-green-900/20">
                  <div className="h-12 w-12 rounded-full bg-green-100 dark:bg-green-900/30 flex items-center justify-center">
                    <CheckCircle2 className="h-6 w-6 text-green-600 dark:text-green-400" />
                  </div>
                  <div>
                    <p className="font-heading font-semibold text-green-900 dark:text-green-100">
                      Successfully Linked
                    </p>
                    <p className="font-body text-sm text-green-700 dark:text-green-300 opacity-80">
                      Active as: <span className="font-mono">{jid}</span>
                    </p>
                  </div>
                </div>
                <div className="p-4 rounded-xl border border-stone-100 dark:border-stone-800 bg-stone-50/30 dark:bg-stone-900/10 space-y-3">
                  <h4 className="font-heading font-semibold text-sm uppercase tracking-wider text-stone-400">
                    Active Features
                  </h4>
                  <ul className="grid grid-cols-2 gap-3 text-sm font-body text-stone-600 dark:text-stone-300">
                    <li className="flex items-center gap-2">
                      <div className="h-1.5 w-1.5 rounded-full bg-[#C53030]" />
                      Auto-reply Agent
                    </li>
                    <li className="flex items-center gap-2">
                      <div className="h-1.5 w-1.5 rounded-full bg-[#C53030]" />
                      Broadcast Manager
                    </li>
                    <li className="flex items-center gap-2">
                      <div className="h-1.5 w-1.5 rounded-full bg-[#C53030]" />
                      Real-time Sync
                    </li>
                    <li className="flex items-center gap-2">
                      <div className="h-1.5 w-1.5 rounded-full bg-[#C53030]" />
                      AI Message Analysis
                    </li>
                  </ul>
                </div>
              </div>
            ) : status === "PAIRING" ? (
              <div className="flex flex-col items-center justify-center space-y-6 py-4">
                {qrCode ? (
                  <div
                    className="relative group animate-in zoom-in duration-300"
                    key={qrCode}
                  >
                    <div className="absolute -inset-1 bg-gradient-to-r from-[#C53030] to-amber-500 rounded-2xl blur opacity-25 group-hover:opacity-40 transition duration-1000 group-hover:duration-200"></div>
                    <div className="relative p-4 bg-white dark:bg-stone-800 rounded-xl border border-stone-200 dark:border-stone-700 shadow-xl">
                      <Image
                        src={`https://api.qrserver.com/v1/create-qr-code/?size=240x240&data=${encodeURIComponent(qrCode)}`}
                        alt="WhatsApp QR Code"
                        width={240}
                        height={240}
                        className="w-[240px] h-[240px]"
                      />
                    </div>
                  </div>
                ) : (
                  <div className="flex flex-col items-center justify-center space-y-4">
                    <div className="h-[272px] w-[272px] rounded-xl bg-stone-100 dark:bg-stone-900 animate-pulse flex items-center justify-center border border-stone-200 dark:border-stone-800">
                      <Loader2 className="h-10 w-10 text-stone-300 animate-spin" />
                    </div>
                  </div>
                )}
                <div className="text-center space-y-2">
                  <p className="font-heading font-bold text-lg text-stone-800 dark:text-stone-200">
                    {qrCode ? "Scan to Link Device" : "Generating QR Code..."}
                  </p>
                  <p className="font-body text-sm text-stone-500 max-w-xs mx-auto">
                    {qrCode
                      ? "Open WhatsApp on your phone, tap Menu or Settings and select Linked Devices."
                      : "Please wait while we request a new security code from WhatsApp."}
                  </p>
                </div>
              </div>
            ) : (
              <div className="flex flex-col items-center justify-center py-12 space-y-4 text-center">
                <div className="h-20 w-20 rounded-full bg-stone-100 dark:bg-stone-800 flex items-center justify-center border-2 border-dashed border-stone-300 dark:border-stone-700">
                  <Unplug className="h-8 w-8 text-stone-400" />
                </div>
                <div className="space-y-1">
                  <h3 className="font-heading font-bold text-xl">
                    Not Connected
                  </h3>
                  <p className="font-body text-stone-500 max-w-xs">
                    Start by initiating a new connection to bridge your WhatsApp
                    to Opus.
                  </p>
                </div>
              </div>
            )}
          </CardContent>

          <CardFooter className="border-t border-stone-100 dark:border-stone-800 p-6 bg-stone-50/30 dark:bg-stone-900/20">
            {status === "CONNECTED" ? (
              <Button
                variant="outline"
                className="w-full sm:w-auto ml-auto border-red-200 hover:bg-red-50 hover:text-red-600 dark:border-red-900/30 dark:hover:bg-red-900/20 transition-all duration-300 gap-2"
                onClick={handleDisconnect}
                disabled={isLoading}
              >
                {isLoading ? (
                  <Loader2 className="h-4 w-4 animate-spin" />
                ) : (
                  <LogOut className="h-4 w-4" />
                )}
                Disconnect Session
              </Button>
            ) : (
              <Button
                className="w-full sm:w-auto ml-auto bg-[#C53030] hover:bg-red-700 text-white shadow-lg shadow-red-900/10 transition-all duration-300 gap-2 px-8"
                onClick={handleConnect}
                disabled={isLoading || status === "PAIRING"}
              >
                {isLoading ? (
                  <Loader2 className="h-4 w-4 animate-spin" />
                ) : (
                  <PlusCircle className="h-4 w-4" />
                )}
                Initiate Connection
              </Button>
            )}
          </CardFooter>
        </Card>

        <div className="space-y-6">
          <Card className="border-stone-200 dark:border-stone-800 shadow-sm bg-stone-50/50 dark:bg-stone-900/20">
            <CardHeader className="pb-3">
              <CardTitle className="text-sm font-heading font-bold uppercase tracking-widest text-stone-400 flex items-center gap-2">
                <Settings2 className="h-4 w-4" />
                Configuration
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="space-y-1">
                <p className="text-xs font-bold text-stone-400">SESSION ID</p>
                <p className="font-mono text-sm break-all bg-white dark:bg-stone-900 p-2 rounded border border-stone-100 dark:border-stone-800">
                  {accessToken?.slice(0, 24)}...
                </p>
              </div>
              <div className="space-y-1">
                <p className="text-xs font-bold text-stone-400">API NODE</p>
                <div className="flex items-center gap-2 text-sm font-body text-stone-600 dark:text-stone-300">
                  <div className="h-2 w-2 rounded-full bg-[#C53030] animate-pulse" />
                  {API_BASE_URL}
                </div>
              </div>
            </CardContent>
          </Card>

          <div className="p-4 rounded-xl border border-amber-100 dark:border-amber-900/30 bg-amber-50/50 dark:bg-amber-900/10 text-amber-900 dark:text-amber-200 text-sm flex gap-3">
            <AlertCircle className="h-5 w-5 shrink-0 text-amber-600 dark:text-amber-400" />
            <div className="space-y-1">
              <p className="font-bold font-heading">Important Note</p>
              <p className="font-body opacity-80 leading-relaxed">
                Stay active! Sessions might expire if unused for 14 days. Don't
                forget to link in "Multi-device" mode.
              </p>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

function StatusBadge({ status }: { status: ConnectionStatus }) {
  const config = {
    CONNECTED: {
      color: "bg-green-500",
      text: "Connected",
      lightBg: "bg-green-50 text-green-700 border-green-100",
      darkBg:
        "dark:bg-green-900/20 dark:text-green-400 dark:border-green-900/30",
    },
    PAIRING: {
      color: "bg-amber-500",
      text: "Action Required",
      lightBg: "bg-amber-50 text-amber-700 border-amber-100",
      darkBg:
        "dark:bg-amber-900/20 dark:text-amber-400 dark:border-amber-900/30",
    },
    UNAUTHENTICATED: {
      color: "bg-stone-400",
      text: "Unauthenticated",
      lightBg: "bg-stone-50 text-stone-700 border-stone-100",
      darkBg: "dark:bg-stone-800 dark:text-stone-400 dark:border-stone-800",
    },
    DISCONNECTED: {
      color: "bg-stone-400",
      text: "Disconnected",
      lightBg: "bg-stone-50 text-stone-700 border-stone-100",
      darkBg: "dark:bg-stone-800 dark:text-stone-400 dark:border-stone-800",
    },
    ERROR: {
      color: "bg-red-500",
      text: "Error",
      lightBg: "bg-red-50 text-red-700 border-red-100",
      darkBg: "dark:bg-red-900/20 dark:text-red-400 dark:border-red-900/30",
    },
    LOADING: {
      color: "bg-blue-500",
      text: "Checking...",
      lightBg: "bg-blue-50 text-blue-700 border-blue-100",
      darkBg: "dark:bg-blue-900/20 dark:text-blue-400 dark:border-blue-900/30",
    },
  };

  const current = config[status];

  return (
    <div
      className={`flex items-center gap-2 px-3 py-1 rounded-full text-xs font-bold border transition-colors duration-500 ${current.lightBg} ${current.darkBg}`}
    >
      <div className={`h-2 w-2 rounded-full ${current.color} animate-pulse`} />
      {current.text}
    </div>
  );
}

function SettingsSkeleton() {
  return (
    <div className="max-w-4xl mx-auto space-y-8 p-6 lg:p-8">
      <div className="space-y-3">
        <Skeleton className="h-10 w-64 bg-stone-200 dark:bg-stone-800" />
        <Skeleton className="h-6 w-96 bg-stone-100 dark:bg-stone-900" />
      </div>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-8">
        <div className="md:col-span-2 space-y-4">
          <Skeleton className="h-[400px] w-full rounded-xl bg-stone-100 dark:bg-stone-900" />
        </div>
        <div className="space-y-6">
          <Skeleton className="h-48 w-full rounded-xl bg-stone-100 dark:bg-stone-900" />
          <Skeleton className="h-32 w-full rounded-xl bg-stone-100 dark:bg-stone-900" />
        </div>
      </div>
    </div>
  );
}
