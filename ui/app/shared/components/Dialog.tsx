import {
  type KeyboardEvent,
  type MouseEvent,
  type ReactNode,
  useEffect,
  useRef,
} from "react";

const FOCUSABLE_SELECTOR = [
  "a[href]",
  "button:not([disabled])",
  "input:not([disabled])",
  "select:not([disabled])",
  "textarea:not([disabled])",
  '[tabindex]:not([tabindex="-1"])',
].join(", ");

export function getFocusableElements(container: HTMLElement): HTMLElement[] {
  return Array.from(
    container.querySelectorAll<HTMLElement>(FOCUSABLE_SELECTOR),
  ).filter((el) => el.getAttribute("aria-hidden") !== "true");
}

export interface DialogProps {
  open: boolean;
  onClose: () => void;
  titleId?: string;
  ariaLabel?: string;
  className?: string;
  children?: ReactNode;
}

export function Dialog({
  open,
  onClose,
  titleId,
  ariaLabel,
  className = "w-full max-w-lg",
  children,
}: DialogProps) {
  const dialogRef = useRef<HTMLDialogElement>(null);
  const openerRef = useRef<HTMLElement | null>(null);

  useEffect(() => {
    const dialog = dialogRef.current;
    if (!dialog) return;

    if (open) {
      if (document.activeElement instanceof HTMLElement) {
        openerRef.current = document.activeElement;
      }
      if (!dialog.open && typeof dialog.showModal === "function") {
        dialog.showModal();
      }
      const autoFocusTarget =
        dialog.querySelector<HTMLElement>("[data-autofocus]");
      const focusTarget = autoFocusTarget ?? getFocusableElements(dialog)[0];
      focusTarget?.focus();
    } else {
      if (dialog.open && typeof dialog.close === "function") {
        dialog.close();
      }
      openerRef.current?.focus();
      openerRef.current = null;
    }
  }, [open]);

  useEffect(() => {
    const dialog = dialogRef.current;
    if (!dialog) return;

    const handleCancel = (e: Event) => {
      e.preventDefault();
      onClose();
    };
    dialog.addEventListener("cancel", handleCancel);
    return () => {
      dialog.removeEventListener("cancel", handleCancel);
    };
  }, [onClose]);

  const handleBackdropClick = (e: MouseEvent<HTMLDialogElement>) => {
    if (e.target === dialogRef.current) {
      onClose();
    }
  };

  const handleKeyDown = (e: KeyboardEvent<HTMLDialogElement>) => {
    if (e.key !== "Tab") return;
    const dialog = dialogRef.current;
    if (!dialog) return;
    const focusables = getFocusableElements(dialog);
    if (focusables.length === 0) {
      e.preventDefault();
      return;
    }
    const first = focusables[0];
    const last = focusables[focusables.length - 1];
    if (e.shiftKey) {
      if (
        document.activeElement === first ||
        document.activeElement === dialog
      ) {
        e.preventDefault();
        last.focus();
      }
    } else if (document.activeElement === last) {
      e.preventDefault();
      first.focus();
    }
  };

  return (
    <dialog
      ref={dialogRef}
      aria-labelledby={titleId}
      aria-label={ariaLabel}
      onClick={handleBackdropClick}
      onKeyDown={handleKeyDown}
      className={`m-auto rounded-lg border border-line bg-surface p-0 text-fg shadow-pop backdrop:bg-black/45 ${className}`}
    >
      {open ? children : null}
    </dialog>
  );
}
