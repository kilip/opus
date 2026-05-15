"use client";

import { Play, Square } from "lucide-react";
import { useState } from "react";
import { StreamOutput } from "@/components/shared/StreamOutput";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
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
    <div className="container py-6 space-y-6 flex-1 flex flex-col">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <Avatar className="h-12 w-12 border-2 border-primary/20">
            <AvatarImage src={user?.avatarUrl} alt={user?.name} />
            <AvatarFallback className="bg-primary/10 text-primary">
              {user?.name?.slice(0, 2).toUpperCase() ?? "OP"}
            </AvatarFallback>
          </Avatar>
          <div>
            <h1 className="text-2xl font-bold tracking-tight">
              Welcome back, {user?.name ?? "Explorer"}
            </h1>
            <p className="text-sm text-muted-foreground">{user?.email}</p>
          </div>
        </div>
        <div className="flex gap-2">
          {!shouldConnect ? (
            <Button onClick={() => setShouldConnect(true)} className="gap-2">
              <Play className="h-4 w-4 fill-current" />
              Connect Stream
            </Button>
          ) : (
            <Button
              variant="outline"
              onClick={() => setShouldConnect(false)}
              className="gap-2"
            >
              <Square className="h-4 w-4 fill-current" />
              Disconnect
            </Button>
          )}
          <Button variant="ghost" onClick={clearOutput} disabled={!output}>
            Clear
          </Button>
        </div>
      </div>

      <Card className="flex-1 flex flex-col border-none bg-muted/50 overflow-hidden">
        <CardHeader className="py-3 border-b bg-background/50">
          <CardTitle className="text-sm font-medium flex items-center justify-between">
            Intelligence Stream
            <span className="flex items-center gap-1.5">
              <span
                className={`h-2 w-2 rounded-full ${isConnected ? "bg-green-500 animate-pulse" : "bg-red-500"}`}
              />
              <span className="text-[10px] uppercase tracking-wider text-muted-foreground font-bold">
                {isConnected ? "Live" : "Offline"}
              </span>
            </span>
          </CardTitle>
        </CardHeader>
        <CardContent className="flex-1 p-0 relative min-h-[400px]">
          <StreamOutput
            output={output}
            isConnected={isConnected}
            error={error}
          />
        </CardContent>
      </Card>
    </div>
  );
}
