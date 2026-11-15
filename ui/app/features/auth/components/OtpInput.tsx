import { forwardRef, useEffect, useImperativeHandle, useRef } from "react";

export interface OtpInputProps {
	id?: string;
	value: string;
	onChange: (value: string) => void;
	disabled?: boolean;
	autoFocus?: boolean;
	className?: string;
	"aria-describedby"?: string;
	"aria-invalid"?: boolean | "true" | "false";
}

// ponytail: single styled input with numeric tracking instead of 6 synced div cells for bulletproof accessibility, autofill, and paste
export const OtpInput = forwardRef<HTMLInputElement, OtpInputProps>(
	function OtpInput(
		{
			id,
			value,
			onChange,
			disabled = false,
			autoFocus = false,
			className = "",
			"aria-describedby": ariaDescribedby,
			"aria-invalid": ariaInvalid,
		},
		ref,
	) {
		const inputRef = useRef<HTMLInputElement>(null);

		useImperativeHandle(ref, () => inputRef.current as HTMLInputElement);

		useEffect(() => {
			if (autoFocus) {
				inputRef.current?.focus();
			}
		}, [autoFocus]);

		return (
			<input
				ref={inputRef}
				id={id}
				type="text"
				inputMode="numeric"
				autoComplete="one-time-code"
				pattern="[0-9]*"
				maxLength={6}
				value={value}
				onChange={(e) => {
					const next = e.target.value.replace(/\D/g, "").slice(0, 6);
					onChange(next);
				}}
				disabled={disabled}
				aria-describedby={ariaDescribedby}
				aria-invalid={ariaInvalid}
				placeholder="123456"
				className={`field h-12 text-center font-mono text-2xl font-semibold tracking-[0.5em] placeholder:font-sans placeholder:text-base placeholder:font-normal placeholder:tracking-normal placeholder:text-muted ${className}`}
			/>
		);
	},
);
