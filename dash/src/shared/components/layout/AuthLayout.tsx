import type { ReactNode } from 'react';

type AuthLayoutProps = {
  children: ReactNode;
};

/**
 * Minimal layout for authentication pages (Login, Setup).
 * Focused on the brand vibe without the dashboard chrome.
 */
export function AuthLayout({ children }: AuthLayoutProps) {
  return (
    <div className="grain-overlay relative min-h-screen flex flex-col items-center justify-center bg-background p-6">
      {/* Subtle background decoration */}
      <div
        className="absolute top-0 left-1/2 -translate-x-1/2 w-full max-w-4xl h-96 opacity-20 pointer-events-none"
        style={{
          background:
            'radial-gradient(circle at center, var(--color-brand-primary) 0%, transparent 70%)',
          filter: 'blur(80px)',
        }}
      />

      <main className="relative z-10 w-full max-w-[400px] demo-reveal">
        {children}
      </main>

      {/* Footer / Branding */}
      <footer
        className="relative z-10 mt-12 text-center demo-reveal"
        style={{ animationDelay: '0.2s' }}
      >
        <p className="font-serif text-sm text-muted">
          Opus — Your 24/7 autonomous AI assistant.
        </p>
      </footer>
    </div>
  );
}
