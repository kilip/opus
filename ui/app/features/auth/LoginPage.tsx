import { useId } from "react";
import { useNavigate } from "react-router";

import { Button } from "~/shared/components/Button";

export const meta = () => [{ title: "Log in · Opus" }];

export default function LoginPage() {
  const navigate = useNavigate();
  const emailId = useId();
  const passwordId = useId();

  return (
    <main className="flex min-h-dvh items-center justify-center bg-bg p-4">
      <form
        onSubmit={(e) => {
          e.preventDefault();
          // TODO: call the auth endpoint once the Go API exposes one.
          navigate("/tasks");
        }}
        className="flex w-full max-w-sm flex-col gap-5 rounded-xl border border-line bg-surface p-8 shadow-pop"
      >
        <div className="flex flex-col items-center gap-3 text-center">
          <img src="/icon.svg" alt="" width={56} height={56} />
          <div>
            <h1 className="text-xl font-semibold">Log in to Opus</h1>
            <p className="text-muted">
              Mission control for your software factory
            </p>
          </div>
        </div>

        <div className="flex flex-col gap-1.5">
          <label htmlFor={emailId} className="font-medium text-muted">
            Email
          </label>
          <input
            id={emailId}
            type="email"
            required
            autoComplete="username"
            className="field"
            placeholder="you@company.com"
          />
        </div>

        <div className="flex flex-col gap-1.5">
          <label htmlFor={passwordId} className="font-medium text-muted">
            Password
          </label>
          <input
            id={passwordId}
            type="password"
            required
            autoComplete="current-password"
            className="field"
          />
        </div>

        <Button variant="primary" type="submit" className="h-10">
          Log in
        </Button>

        <p className="text-center text-xs text-muted">
          Demo mode: authentication is not connected yet.
        </p>
      </form>
    </main>
  );
}
