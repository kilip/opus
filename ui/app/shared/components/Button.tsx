import type { ButtonHTMLAttributes } from "react";

export type ButtonVariant = "primary" | "secondary" | "ghost";

const base =
  "inline-flex h-8 items-center justify-center gap-1.5 rounded-md px-3 text-sm font-medium whitespace-nowrap transition-colors disabled:cursor-not-allowed disabled:opacity-50";

const variants: Record<ButtonVariant, string> = {
  primary: "bg-primary text-primary-fg hover:bg-primary-hover",
  secondary: "border border-line-strong bg-surface text-fg hover:bg-surface-2",
  ghost: "text-fg hover:bg-surface-2",
};

/** Class string so links (`<Link>`) can look like buttons too. */
export function buttonClass(variant: ButtonVariant = "secondary", extra = "") {
  return `${base} ${variants[variant]} ${extra}`.trim();
}

export function Button({
  variant = "secondary",
  className = "",
  type,
  ...props
}: ButtonHTMLAttributes<HTMLButtonElement> & { variant?: ButtonVariant }) {
  return (
    <button
      type={type ?? "button"}
      className={buttonClass(variant, className)}
      {...props}
    />
  );
}
