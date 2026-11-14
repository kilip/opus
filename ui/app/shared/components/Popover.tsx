import {
  type ReactNode,
  useCallback,
  useEffect,
  useId,
  useRef,
  useState,
} from "react";

export type PopoverTriggerProps = {
  onClick: () => void;
  "aria-expanded": boolean;
  "aria-haspopup": "true";
  "aria-controls": string;
};

type PopoverProps = {
  renderTrigger: (props: PopoverTriggerProps) => ReactNode;
  children: ReactNode | ((close: () => void) => ReactNode);
  align?: "left" | "right";
  className?: string;
};

/** Minimal dropdown: closes on outside click and Escape. No dependencies. */
export function Popover({
  renderTrigger,
  children,
  align = "left",
  className = "",
}: PopoverProps) {
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);
  const panelId = useId();
  const close = useCallback(() => setOpen(false), []);

  useEffect(() => {
    if (!open) return;
    const onPointerDown = (e: PointerEvent) => {
      if (!ref.current?.contains(e.target as Node)) close();
    };
    const onKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Escape") close();
    };
    document.addEventListener("pointerdown", onPointerDown);
    document.addEventListener("keydown", onKeyDown);
    return () => {
      document.removeEventListener("pointerdown", onPointerDown);
      document.removeEventListener("keydown", onKeyDown);
    };
  }, [open, close]);

  return (
    <div ref={ref} className="relative inline-flex">
      {renderTrigger({
        onClick: () => setOpen((o) => !o),
        "aria-expanded": open,
        "aria-haspopup": "true",
        "aria-controls": panelId,
      })}
      {open && (
        <div
          id={panelId}
          className={`absolute top-full z-40 mt-1 rounded-lg border border-line bg-surface text-fg shadow-pop ${
            align === "right" ? "right-0" : "left-0"
          } ${className}`}
        >
          {typeof children === "function" ? children(close) : children}
        </div>
      )}
    </div>
  );
}
