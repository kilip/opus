"use client";

import {
  AlertCircle,
  CheckCircle2,
  Loader2,
  LogOut,
  MessageSquare,
  QrCode,
} from "lucide-react";
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

export default function WhatsAppSettingsPage() {
  const [status, setStatus] = useState<
    | "LOADING"
    | "UNAUTHENTICATED"
    | "PAIRING"
    | "CONNECTED"
    | "DISCONNECTED"
    | "ERROR"
  >("LOADING");
  const [jid, setJid] = useState("");
  const [qrCode, setQrCode] = useState("");
  const [isLoading, setIsLoading] = useState(false);

  const fetchStatus = useCallback(async () => {
    try {
      const res = await fetch("/api/whatsapp/status");
      const data = await res.json();
      setStatus(data.status || "UNAUTHENTICATED");
      setJid(data.jid || "");
    } catch (_err) {
      setStatus("ERROR");
    }
  }, []);

  useEffect(() => {
    // Fetch initial status
    fetchStatus();

    // Listen to SSE (assuming generic /api/stream endpoint based on MEMORY.md decision)
    // Actually, AGENTS.md says SSE is on /stream
    const eventSource = new EventSource("/api/stream");

    eventSource.addEventListener("wa_qr_update", (e: MessageEvent) => {
      setQrCode(e.data);
      setStatus("PAIRING");
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
  }, [fetchStatus]);

  const handleConnect = async () => {
    setIsLoading(true);
    try {
      await fetch("/api/whatsapp/connect", { method: "POST" });
    } finally {
      setIsLoading(false);
    }
  };

  const handleDisconnect = async () => {
    setIsLoading(true);
    try {
      await fetch("/api/whatsapp/disconnect", { method: "POST" });
    } finally {
      setIsLoading(false);
    }
  };

  if (status === "LOADING") {
    return (
      <div className="flex-1 flex items-center justify-center p-6">
        <div className="flex flex-col items-center gap-4">
          <Loader2 className="h-12 w-12 animate-spin text-primary" />
          <p className="text-muted-foreground animate-pulse font-medium">
            Checking connection status...
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className="flex-1 overflow-auto p-6 lg:p-8 space-y-8 bg-slate-50/50">
      <div className="max-w-4xl mx-auto space-y-8">
        <div className="space-y-1">
          <h1 className="text-3xl font-bold tracking-tight">
            WhatsApp Settings
          </h1>
          <p className="text-muted-foreground">
            Manage your WhatsApp integration and device connection.
          </p>
        </div>

        <div className="grid gap-6 md:grid-cols-2">
          {/* Connection Status Card */}
          <Card className="shadow-sm border-slate-200">
            <CardHeader>
              <div className="flex items-center justify-between">
                <CardTitle className="text-lg flex items-center gap-2">
                  <MessageSquare className="h-5 w-5 text-primary" />
                  Connection Status
                </CardTitle>
                <div
                  className={`px-2.5 py-0.5 rounded-full text-xs font-semibold uppercase tracking-wider ${
                    status === "CONNECTED"
                      ? "bg-green-100 text-green-700"
                      : status === "PAIRING"
                        ? "bg-blue-100 text-blue-700 animate-pulse"
                        : "bg-slate-100 text-slate-700"
                  }`}
                >
                  {status}
                </div>
              </div>
              <CardDescription>
                View and manage your active WhatsApp session.
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-6">
              {status === "CONNECTED" ? (
                <div className="flex items-center gap-4 p-4 rounded-xl bg-green-50 border border-green-100">
                  <div className="h-12 w-12 rounded-full bg-green-200 flex items-center justify-center text-green-700">
                    <CheckCircle2 className="h-7 w-7" />
                  </div>
                  <div>
                    <p className="text-sm font-medium text-green-900">
                      Successfully Connected
                    </p>
                    <p className="text-xs text-green-700 font-mono">{jid}</p>
                  </div>
                </div>
              ) : (
                <div className="flex items-center gap-4 p-4 rounded-xl bg-slate-100 border border-slate-200">
                  <div className="h-12 w-12 rounded-full bg-slate-200 flex items-center justify-center text-slate-500">
                    <AlertCircle className="h-7 w-7" />
                  </div>
                  <div>
                    <p className="text-sm font-medium text-slate-900">
                      Not Connected
                    </p>
                    <p className="text-xs text-slate-500">
                      Connect your device to start messaging.
                    </p>
                  </div>
                </div>
              )}
            </CardContent>
            <CardFooter className="bg-slate-50/50 border-t p-4">
              {status === "CONNECTED" ? (
                <Button
                  onClick={handleDisconnect}
                  variant="destructive"
                  className="w-full sm:w-auto ml-auto gap-2"
                  disabled={isLoading}
                >
                  <LogOut className="h-4 w-4" />
                  Disconnect Device
                </Button>
              ) : (
                <Button
                  onClick={handleConnect}
                  className="w-full sm:w-auto ml-auto gap-2 shadow-sm"
                  disabled={isLoading}
                >
                  <QrCode className="h-4 w-4" />
                  Connect New Device
                </Button>
              )}
            </CardFooter>
          </Card>

          {/* QR Code Card (Only when pairing) */}
          {status === "PAIRING" && qrCode && (
            <Card className="shadow-lg border-primary/20 bg-primary/5">
              <CardHeader>
                <CardTitle className="text-lg flex items-center gap-2">
                  <QrCode className="h-5 w-5 text-primary" />
                  Pair Device
                </CardTitle>
                <CardDescription>
                  Scan this QR code with WhatsApp on your phone.
                </CardDescription>
              </CardHeader>
              <CardContent className="flex flex-col items-center justify-center space-y-6 py-10">
                <div className="p-4 bg-white rounded-2xl shadow-sm border border-slate-200">
                  {/* Using raw pre for text QR as requested in plan, but styled better */}
                  <pre className="bg-white p-2 text-[6px] leading-[6px] font-mono whitespace-pre overflow-hidden select-all">
                    {qrCode}
                  </pre>
                </div>
                <div className="text-center space-y-2">
                  <p className="text-sm font-medium">Waiting for scan...</p>
                  <p className="text-xs text-muted-foreground max-w-[200px]">
                    Open WhatsApp {">"} Settings {">"} Linked Devices {">"} Link
                    a Device.
                  </p>
                </div>
              </CardContent>
            </Card>
          )}
        </div>
      </div>
    </div>
  );
}
