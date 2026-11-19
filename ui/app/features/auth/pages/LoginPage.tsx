import { type FormEvent, useEffect, useId, useRef, useState } from "react";
import { Navigate, useNavigate, useSearchParams } from "react-router";

import { ApiError } from "~/shared/api/client";
import { Button, buttonClass } from "~/shared/components/Button";
import { Icon, type IconName, isIconName } from "~/shared/components/Icon";
import { useRequestOTP, useVerifyOTP } from "../api/actions";
import { RateLimitError, resolveAuthError } from "../api/errors";
import { useProviders, useSession, useSetupStatus } from "../api/queries";
import { OtpInput } from "../components/OtpInput";
import {
  formatCooldownLabel,
  nextCooldownTick,
  OTP_RESEND_COOLDOWN_SECONDS,
} from "../model";

export const meta = () => [{ title: "Log in · Opus" }];

export function resolveSafeNextPath(raw: string | null): string {
  if (!raw) return "/tasks";
  if (!raw.startsWith("/") || raw.startsWith("//") || raw.startsWith("/\\")) {
    return "/tasks";
  }
  try {
    const parsed = new URL(raw, "http://localhost");
    if (parsed.origin !== "http://localhost") {
      return "/tasks";
    }
    return `${parsed.pathname}${parsed.search}${parsed.hash}`;
  } catch {
    return "/tasks";
  }
}

export default function LoginPage() {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const emailId = useId();
  const codeId = useId();
  const codeInputRef = useRef<HTMLInputElement>(null);

  const [step, setStep] = useState<1 | 2>(1);
  const [email, setEmail] = useState("");
  const [code, setCode] = useState("");
  const [error, setError] = useState<string | null>(() =>
    resolveAuthError(searchParams.get("error")),
  );
  const [cooldown, setCooldown] = useState(0);

  const { data: setupStatus, isLoading: isSetupLoading } = useSetupStatus();
  const { data: user, isLoading: isSessionLoading } = useSession();
  const { data: providers } = useProviders();

  const requestOTP = useRequestOTP();
  const verifyOTP = useVerifyOTP();

  const isPending = requestOTP.isPending || verifyOTP.isPending;

  useEffect(() => {
    const urlError = resolveAuthError(searchParams.get("error"));
    if (urlError) {
      setError(urlError);
    }
  }, [searchParams]);

  useEffect(() => {
    if (step === 2) {
      codeInputRef.current?.focus();
    }
  }, [step]);

  useEffect(() => {
    if (cooldown <= 0) return;
    const timer = setTimeout(() => {
      setCooldown((prev) => nextCooldownTick(prev));
    }, 1000);
    return () => clearTimeout(timer);
  }, [cooldown]);

  if (isSetupLoading || isSessionLoading) {
    return null;
  }

  if (setupStatus?.needs_setup) {
    return <Navigate to="/onboarding" replace />;
  }

  const nextTarget = resolveSafeNextPath(searchParams.get("next"));
  const isSessionExpired = searchParams.get("expired") === "1";

  if (user) {
    return <Navigate to={nextTarget} replace />;
  }

  const handleRequestOTP = async (e: FormEvent) => {
    e.preventDefault();
    setError(null);
    const cleanEmail = email.trim();
    if (!cleanEmail) {
      return;
    }

    try {
      await requestOTP.mutateAsync({ email: cleanEmail });
      setCooldown(OTP_RESEND_COOLDOWN_SECONDS);
      setStep(2);
    } catch (err) {
      if (err instanceof RateLimitError) {
        if (err.retryAfter && err.retryAfter > 0) {
          setCooldown(err.retryAfter);
        }
        setError(err.message);
      } else if (err instanceof ApiError) {
        setError(err.message);
      } else {
        setError("Failed to send verification code. Please try again.");
      }
    }
  };

  const handleResendOTP = async () => {
    if (cooldown > 0 || isPending) return;
    setError(null);
    try {
      await requestOTP.mutateAsync({ email: email.trim() });
      setCooldown(OTP_RESEND_COOLDOWN_SECONDS);
    } catch (err) {
      if (err instanceof RateLimitError) {
        if (err.retryAfter && err.retryAfter > 0) {
          setCooldown(err.retryAfter);
        }
        setError(err.message);
      } else if (err instanceof ApiError) {
        setError(err.message);
      } else {
        setError("Failed to resend code.");
      }
    }
  };

  const handleVerifyOTP = async (e: FormEvent) => {
    e.preventDefault();
    setError(null);
    const cleanCode = code.trim();
    if (cleanCode.length !== 6) {
      return;
    }

    try {
      await verifyOTP.mutateAsync({
        email: email.trim(),
        code: cleanCode,
      });
      navigate(nextTarget, { replace: true });
    } catch (err) {
      if (err instanceof ApiError) {
        setError(err.message);
      } else {
        setError("Failed to verify code. Please try again.");
      }
    }
  };

  return (
    <main className="flex min-h-dvh items-center justify-center bg-bg p-4">
      <form
        onSubmit={step === 1 ? handleRequestOTP : handleVerifyOTP}
        className="flex w-full max-w-sm flex-col gap-5 rounded-xl border border-line bg-surface p-8 shadow-pop"
      >
        <div className="flex flex-col items-center gap-3 text-center">
          <img src="/icon.svg" alt="" width={56} height={56} />
          <div>
            <h1 className="text-xl font-semibold">
              {step === 1 ? "Log in to Opus" : "Enter verification code"}
            </h1>
            <p className="text-sm text-muted">
              {step === 1
                ? "Mission control for your software factory"
                : `We sent a 6-digit code to ${email}`}
            </p>
          </div>
        </div>

        {isSessionExpired && !error && (
          <div
            role="status"
            className="rounded-md border border-line bg-surface-2 p-3 text-sm text-fg"
          >
            Your session expired. Please log in again.
          </div>
        )}

        {error && (
          <div
            role="alert"
            className="rounded-md border border-danger/30 bg-danger/10 p-3 text-sm text-danger"
          >
            {error}
          </div>
        )}

        {step === 1 ? (
          <>
            {providers && providers.length > 0 && (
              <>
                <div className="flex flex-col gap-2">
                  {providers.map((provider) => {
                    const name = provider.name ?? "";
                    const lowerName = name.toLowerCase();
                    const iconName = isIconName(lowerName)
                      ? (lowerName as IconName)
                      : undefined;
                    return (
                      <a
                        key={name}
                        href={`/api/auth/oauth/${name}`}
                        className={buttonClass(
                          "secondary",
                          "h-10 w-full gap-2",
                        )}
                      >
                        {iconName && <Icon name={iconName} size={18} />}
                        <span>{`Continue with ${provider.label || name}`}</span>
                      </a>
                    );
                  })}
                </div>

                <div className="relative flex items-center justify-center">
                  <div className="w-full border-t border-line" />
                  <span className="bg-surface px-2 text-xs uppercase tracking-wider text-muted">
                    or
                  </span>
                </div>
              </>
            )}

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
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                disabled={isPending}
              />
            </div>

            <Button
              variant="primary"
              type="submit"
              className="h-10"
              disabled={isPending || !email.trim()}
            >
              {isPending ? "Sending code..." : "Continue with Email"}
            </Button>
          </>
        ) : (
          <>
            <div className="flex flex-col gap-1.5">
              <label htmlFor={codeId} className="font-medium text-muted">
                Verification code
              </label>
              <OtpInput
                ref={codeInputRef}
                id={codeId}
                value={code}
                onChange={(val) => {
                  setCode(val);
                  if (error) setError(null);
                }}
                disabled={isPending}
                aria-invalid={error ? "true" : "false"}
              />
            </div>

            <Button
              variant="primary"
              type="submit"
              className="h-10"
              disabled={isPending || code.trim().length !== 6}
            >
              {isPending ? "Verifying..." : "Log in"}
            </Button>

            <div className="flex items-center justify-between text-xs text-muted">
              <button
                type="button"
                onClick={() => {
                  setStep(1);
                  setCode("");
                  setError(null);
                  setCooldown(0);
                }}
                className="cursor-pointer underline hover:text-fg"
                disabled={isPending}
              >
                Use a different email
              </button>
              <button
                type="button"
                onClick={handleResendOTP}
                className="cursor-pointer underline hover:text-fg disabled:cursor-not-allowed disabled:opacity-50 disabled:no-underline"
                disabled={isPending || cooldown > 0}
              >
                {formatCooldownLabel(cooldown)}
              </button>
            </div>
          </>
        )}
      </form>
    </main>
  );
}
