# Branding Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement the Earthy & Warm brand guidelines (Colors, Typography, and Glassmorphism strategy) on the Next.js 16 frontend.

**Architecture:** We will update `dash/app/layout.tsx` to load Poppins and Lora using `next/font/google`. We will update `dash/app/globals.css` with the new theme colors mapped to Shadcn/ui CSS variables and setup custom Tailwind 4 `@theme` properties for fonts and specific branding colors.

**Tech Stack:** Next.js 16, Tailwind CSS v4, Shadcn/ui

---

### Task 1: Update Fonts in Root Layout

**Files:**
- Modify: `dash/app/layout.tsx`

- [ ] **Step 1: Update Font Imports and Usage**

Modify `dash/app/layout.tsx` to import and apply `Poppins` (for headings) and `Lora` (for body).

```tsx
import type { Metadata } from "next";
import { Poppins, Lora } from "next/font/google";
import "./globals.css";
import { QueryProvider } from "@/components/shared/QueryProvider";
import { AuthProvider } from "@/lib/api/AuthContext";

const poppins = Poppins({ 
  subsets: ["latin"], 
  weight: ["400", "600", "700"],
  variable: "--font-poppins",
});

const lora = Lora({ 
  subsets: ["latin"], 
  weight: ["400", "500", "600"],
  variable: "--font-lora",
});

export const metadata: Metadata = {
  title: "Opus",
  description: "Your 24/7 autonomous AI assistant",
  manifest: "/manifest.webmanifest",
  appleWebApp: {
    capable: true,
    statusBarStyle: "default",
    title: "Opus",
  },
  icons: {
    apple: "/icons/apple-touch-icon.png",
  },
};

export const viewport = {
  themeColor: "#1C1917", // Updated to new Dark color
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en" suppressHydrationWarning>
      <body className={`${lora.variable} ${poppins.variable} font-sans antialiased`}>
        <AuthProvider>
          <QueryProvider>{children}</QueryProvider>
        </AuthProvider>
      </body>
    </html>
  );
}
```

- [ ] **Step 2: Commit**

```bash
cd dash
git add app/layout.tsx
git commit -m "feat(ui): update root layout to use Poppins and Lora fonts"
```

### Task 2: Update Tailwind Theme & CSS Variables

**Files:**
- Modify: `dash/app/globals.css`

- [ ] **Step 1: Apply New Colors and Fonts to Global CSS**

Replace the `:root`, `.dark`, and `@theme inline` sections in `dash/app/globals.css`.

```css
@import "tailwindcss";
@import "tw-animate-css";
@import "shadcn/tailwind.css";

@custom-variant dark (&:is(.dark *));

@theme inline {
  --color-background: var(--background);
  --color-foreground: var(--foreground);
  --font-sans: var(--font-lora);
  --font-heading: var(--font-poppins);
  --color-sidebar-ring: var(--sidebar-ring);
  --color-sidebar-border: var(--sidebar-border);
  --color-sidebar-accent-foreground: var(--sidebar-accent-foreground);
  --color-sidebar-accent: var(--sidebar-accent);
  --color-sidebar-primary-foreground: var(--sidebar-primary-foreground);
  --color-sidebar-primary: var(--sidebar-primary);
  --color-sidebar-foreground: var(--sidebar-foreground);
  --color-sidebar: var(--sidebar);
  --color-chart-5: var(--chart-5);
  --color-chart-4: var(--chart-4);
  --color-chart-3: var(--chart-3);
  --color-chart-2: var(--chart-2);
  --color-chart-1: var(--chart-1);
  --color-ring: var(--ring);
  --color-input: var(--input);
  --color-border: var(--border);
  --color-destructive: var(--destructive);
  --color-accent-foreground: var(--accent-foreground);
  --color-accent: var(--accent);
  --color-muted-foreground: var(--muted-foreground);
  --color-muted: var(--muted);
  --color-secondary-foreground: var(--secondary-foreground);
  --color-secondary: var(--secondary);
  --color-primary-foreground: var(--primary-foreground);
  --color-primary: var(--primary);
  --color-popover-foreground: var(--popover-foreground);
  --color-popover: var(--popover);
  --color-card-foreground: var(--card-foreground);
  --color-card: var(--card);
  --radius-sm: calc(var(--radius) * 0.6);
  --radius-md: calc(var(--radius) * 0.8);
  --radius-lg: var(--radius);
  --radius-xl: calc(var(--radius) * 1.4);
  --radius-2xl: calc(var(--radius) * 1.8);
  --radius-3xl: calc(var(--radius) * 2.2);
  --radius-4xl: calc(var(--radius) * 2.6);
  
  /* Brand specific colors for manual overrides if needed */
  --color-opus-dark: #1C1917;
  --color-opus-light: #FFFCF5;
  --color-opus-mid-gray: #A8A29E;
  --color-opus-light-gray: #F5F5F4;
  --color-opus-terracotta: #C53030;
  --color-opus-mustard: #D97706;
  --color-opus-sage: #657B83;
}

:root {
  --background: #FFFCF5;
  --foreground: #1C1917;
  --card: #FFFFFF;
  --card-foreground: #1C1917;
  --popover: #FFFFFF;
  --popover-foreground: #1C1917;
  --primary: #C53030;
  --primary-foreground: #FFFCF5;
  --secondary: #F5F5F4;
  --secondary-foreground: #1C1917;
  --muted: #F5F5F4;
  --muted-foreground: #A8A29E;
  --accent: #F5F5F4;
  --accent-foreground: #1C1917;
  --destructive: #C53030;
  --border: #E5E5E5;
  --input: #E5E5E5;
  --ring: #C53030;
  --chart-1: #C53030;
  --chart-2: #D97706;
  --chart-3: #657B83;
  --chart-4: #A8A29E;
  --chart-5: #1C1917;
  --radius: 0.625rem;
  --sidebar: #FFFCF5;
  --sidebar-foreground: #1C1917;
  --sidebar-primary: #C53030;
  --sidebar-primary-foreground: #FFFCF5;
  --sidebar-accent: #F5F5F4;
  --sidebar-accent-foreground: #1C1917;
  --sidebar-border: #E5E5E5;
  --sidebar-ring: #C53030;
}

.dark {
  --background: #1C1917;
  --foreground: #FFFCF5;
  --card: #292524;
  --card-foreground: #FFFCF5;
  --popover: #292524;
  --popover-foreground: #FFFCF5;
  --primary: #C53030;
  --primary-foreground: #FFFCF5;
  --secondary: #292524;
  --secondary-foreground: #FFFCF5;
  --muted: #292524;
  --muted-foreground: #A8A29E;
  --accent: #292524;
  --accent-foreground: #FFFCF5;
  --destructive: #C53030;
  --border: #44403C;
  --input: #44403C;
  --ring: #C53030;
  --chart-1: #C53030;
  --chart-2: #D97706;
  --chart-3: #657B83;
  --chart-4: #A8A29E;
  --chart-5: #FFFCF5;
  --sidebar: #1C1917;
  --sidebar-foreground: #FFFCF5;
  --sidebar-primary: #C53030;
  --sidebar-primary-foreground: #FFFCF5;
  --sidebar-accent: #292524;
  --sidebar-accent-foreground: #FFFCF5;
  --sidebar-border: #44403C;
  --sidebar-ring: #C53030;
}

@layer base {
  * {
    @apply border-border outline-ring/50;
  }
  body {
    @apply bg-background text-foreground;
  }
  html {
    @apply font-sans;
  }
  h1, h2, h3, h4, h5, h6 {
    @apply font-heading tracking-tight;
  }
}

/* Glassmorphism Utilities */
@layer utilities {
  .glass-panel {
    @apply bg-white/70 dark:bg-black/60 backdrop-blur-md border border-white/40 dark:border-white/10 shadow-md;
  }
}
```

- [ ] **Step 2: Commit**

```bash
cd dash
git add app/globals.css
git commit -m "feat(ui): implement earthy theme colors, typography mapping and glassmorphism utility"
```