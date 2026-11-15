import { type FormEvent, useEffect, useId, useRef, useState } from "react";
import { Navigate, useNavigate } from "react-router";

import { ApiError } from "~/shared/api/client";
import { Button } from "~/shared/components/Button";
import { useCompleteSetup, useRequestSetupOTP } from "../api/actions";
import { useSession, useSetupStatus } from "../api/queries";
import { OtpInput } from "../components/OtpInput";

export const meta = () => [{ title: "Welcome · Opus" }];

export default function OnboardingPage() {
	const navigate = useNavigate();
	const emailId = useId();
	const codeId = useId();
	const codeInputRef = useRef<HTMLInputElement>(null);

	const [step, setStep] = useState<1 | 2>(1);
	const [email, setEmail] = useState("");
	const [code, setCode] = useState("");
	const [error, setError] = useState<string | null>(null);

	const { data: setupStatus, isLoading: isSetupLoading } = useSetupStatus();
	const { data: user, isLoading: isSessionLoading } = useSession();

	const requestSetupOTP = useRequestSetupOTP();
	const completeSetup = useCompleteSetup();

	const isPending = requestSetupOTP.isPending || completeSetup.isPending;

	useEffect(() => {
		if (step === 2) {
			codeInputRef.current?.focus();
		}
	}, [step]);

	if (isSetupLoading || isSessionLoading) {
		return null;
	}

	if (user) {
		return <Navigate to="/tasks" replace />;
	}

	if (setupStatus && !setupStatus.needs_setup) {
		return <Navigate to="/login" replace />;
	}

	const handleRequestOTP = async (e: FormEvent) => {
		e.preventDefault();
		setError(null);
		const cleanEmail = email.trim();
		if (!cleanEmail) {
			return;
		}

		try {
			await requestSetupOTP.mutateAsync({ email: cleanEmail });
			setStep(2);
		} catch (err) {
			if (
				err instanceof ApiError &&
				(err.status === 409 || err.code === "SETUP_CLOSED")
			) {
				navigate("/login", { replace: true });
				return;
			}
			if (err instanceof ApiError) {
				setError(err.message);
			} else {
				setError("Failed to send verification code. Please try again.");
			}
		}
	};

	const handleCompleteSetup = async (e: FormEvent) => {
		e.preventDefault();
		setError(null);
		const cleanCode = code.trim();
		if (cleanCode.length !== 6) {
			return;
		}

		try {
			await completeSetup.mutateAsync({
				email: email.trim(),
				code: cleanCode,
			});
			navigate("/tasks", { replace: true });
		} catch (err) {
			if (
				err instanceof ApiError &&
				(err.status === 409 || err.code === "SETUP_CLOSED")
			) {
				navigate("/login", { replace: true });
				return;
			}
			if (err instanceof ApiError) {
				setError(err.message);
			} else {
				setError("Failed to complete setup. Please try again.");
			}
		}
	};

	return (
		<main className="flex min-h-dvh items-center justify-center bg-bg p-4">
			<form
				onSubmit={step === 1 ? handleRequestOTP : handleCompleteSetup}
				className="flex w-full max-w-sm flex-col gap-5 rounded-xl border border-line bg-surface p-8 shadow-pop"
			>
				<div className="flex flex-col items-center gap-3 text-center">
					<img src="/icon.svg" alt="" width={56} height={56} />
					<div>
						<h1 className="text-xl font-semibold">
							{step === 1 ? "Welcome to Opus" : "Confirm owner email"}
						</h1>
						<p className="text-sm text-muted">
							{step === 1
								? "Set up your owner account to get started"
								: `We sent a 6-digit code to ${email}`}
						</p>
					</div>
				</div>

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
						<div className="flex flex-col gap-1.5">
							<label htmlFor={emailId} className="font-medium text-muted">
								Owner email
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
							{isPending ? "Sending code..." : "Continue"}
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
							{isPending ? "Setting up..." : "Complete setup"}
						</Button>

						<div className="flex items-center justify-between text-xs text-muted">
							<button
								type="button"
								onClick={() => {
									setStep(1);
									setCode("");
									setError(null);
								}}
								className="cursor-pointer underline hover:text-fg"
								disabled={isPending}
							>
								Use different email
							</button>
							<button
								type="button"
								onClick={async () => {
									setError(null);
									try {
										await requestSetupOTP.mutateAsync({ email: email.trim() });
									} catch (err) {
										if (
											err instanceof ApiError &&
											(err.status === 409 || err.code === "SETUP_CLOSED")
										) {
											navigate("/login", { replace: true });
											return;
										}
										if (err instanceof ApiError) {
											setError(err.message);
										} else {
											setError("Failed to resend code.");
										}
									}
								}}
								className="cursor-pointer underline hover:text-fg"
								disabled={isPending}
							>
								Resend code
							</button>
						</div>
					</>
				)}
			</form>
		</main>
	);
}
