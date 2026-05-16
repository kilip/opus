"use client";

import { Loader2, MessageSquare, Send, ShieldCheck, User } from "lucide-react";
import { useState } from "react";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

export default function WhatsAppChatPage() {
  const [targetJid, setTargetJid] = useState("");
  const [message, setMessage] = useState("");
  const [statusMsg, setStatusMsg] = useState<{
    text: string;
    type: "info" | "success" | "error";
  } | null>(null);
  const [isSending, setIsSending] = useState(false);

  const handleSend = async () => {
    if (!targetJid || !message) return;
    setIsSending(true);
    setStatusMsg({ text: "Sending your message...", type: "info" });

    try {
      const res = await fetch("/api/whatsapp/send", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ target_jid: targetJid, message }),
      });

      if (res.ok) {
        setStatusMsg({ text: "Message sent successfully!", type: "success" });
        setMessage("");
      } else {
        const data = await res.json();
        setStatusMsg({
          text: data.error || "Failed to send message.",
          type: "error",
        });
      }
    } catch (_e) {
      setStatusMsg({ text: "An error occurred while sending.", type: "error" });
    } finally {
      setIsSending(false);
    }
  };

  return (
    <div className="flex-1 overflow-auto p-6 lg:p-8 bg-slate-50/50">
      <div className="max-w-2xl mx-auto space-y-8">
        <div className="space-y-1">
          <h1 className="text-3xl font-bold tracking-tight">Quick Message</h1>
          <p className="text-muted-foreground">
            Send a direct WhatsApp message to any number.
          </p>
        </div>

        <Card className="shadow-lg border-slate-200">
          <CardHeader className="border-b bg-slate-50/50">
            <CardTitle className="text-lg flex items-center gap-2">
              <MessageSquare className="h-5 w-5 text-primary" />
              New Conversation
            </CardTitle>
            <CardDescription>
              Enter the recipient's WhatsApp ID (JID) and your message.
            </CardDescription>
          </CardHeader>
          <CardContent className="p-6 space-y-6">
            <div className="space-y-2">
              <Label
                htmlFor="target_jid"
                className="text-sm font-semibold flex items-center gap-2"
              >
                <User className="h-4 w-4" />
                Recipient JID
              </Label>
              <Input
                id="target_jid"
                type="text"
                placeholder="e.g. 628123456789@s.whatsapp.net"
                value={targetJid}
                onChange={(e) => setTargetJid(e.target.value)}
                className="bg-white"
              />
              <p className="text-[10px] text-muted-foreground ml-1 flex items-center gap-1">
                <ShieldCheck className="h-3 w-3" />
                Format: [phone_number]@s.whatsapp.net
              </p>
            </div>

            <div className="space-y-2">
              <Label
                htmlFor="message"
                className="text-sm font-semibold flex items-center gap-2"
              >
                <MessageSquare className="h-4 w-4" />
                Message Content
              </Label>
              <textarea
                id="message"
                className="flex min-h-[160px] w-full rounded-md border border-input bg-white px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
                placeholder="Type your message here..."
                value={message}
                onChange={(e) => setMessage(e.target.value)}
              />
            </div>
          </CardContent>
          <CardFooter className="bg-slate-50/50 border-t p-4 flex flex-col gap-4">
            <Button
              onClick={handleSend}
              className="w-full sm:w-auto ml-auto gap-2 shadow-md px-8"
              disabled={isSending || !targetJid || !message}
            >
              {isSending ? (
                <>
                  <Loader2 className="h-4 w-4 animate-spin" />
                  Sending...
                </>
              ) : (
                <>
                  <Send className="h-4 w-4" />
                  Send Message
                </>
              )}
            </Button>

            {statusMsg && (
              <div
                className={`w-full p-3 rounded-lg text-sm font-medium border ${
                  statusMsg.type === "success"
                    ? "bg-green-50 text-green-700 border-green-100"
                    : statusMsg.type === "error"
                      ? "bg-red-50 text-red-700 border-red-100"
                      : "bg-blue-50 text-blue-700 border-blue-100"
                }`}
              >
                {statusMsg.text}
              </div>
            )}
          </CardFooter>
        </Card>

        {/* Tip/Info Card */}
        <div className="p-4 rounded-xl border border-amber-100 bg-amber-50 text-amber-800 text-sm flex gap-3">
          <ShieldCheck className="h-5 w-5 shrink-0" />
          <div>
            <p className="font-semibold">Pro Tip</p>
            <p className="opacity-90">
              Make sure your WhatsApp is connected in Settings before sending
              messages. Use international format without '+' sign.
            </p>
          </div>
        </div>
      </div>
    </div>
  );
}
