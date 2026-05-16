"use client";

import type * as React from "react";
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { cn } from "@/lib/utils";

interface PremiumCardProps extends React.ComponentProps<typeof Card> {
  title?: string;
  description?: string;
  footer?: React.ReactNode;
  gradient?: boolean;
}

export function PremiumCard({
  title,
  description,
  footer,
  children,
  className,
  gradient = false,
  ...props
}: PremiumCardProps) {
  return (
    <Card
      className={cn(
        "overflow-hidden border-opus-gray-mid/20 bg-white/50 dark:bg-opus-dark/50 backdrop-blur-sm transition-all duration-300 hover:shadow-xl hover:shadow-opus-terracotta/5 group",
        gradient &&
          "bg-gradient-to-br from-white/80 to-opus-light/30 dark:from-opus-dark/80 dark:to-opus-dark/30",
        className,
      )}
      {...props}
    >
      {(title || description) && (
        <CardHeader className="pb-4">
          {title && (
            <CardTitle className="font-heading text-xl font-bold text-opus-dark dark:text-opus-light group-hover:text-opus-terracotta transition-colors">
              {title}
            </CardTitle>
          )}
          {description && (
            <CardDescription className="font-body text-sm text-opus-gray-mid italic">
              {description}
            </CardDescription>
          )}
        </CardHeader>
      )}
      <CardContent
        className={cn("font-body", title || description ? "pt-0" : "pt-6")}
      >
        {children}
      </CardContent>
      {footer && (
        <CardFooter className="bg-opus-gray-light/30 dark:bg-opus-dark/30 border-t border-opus-gray-mid/10 py-3">
          {footer}
        </CardFooter>
      )}
    </Card>
  );
}
