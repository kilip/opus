import { Icon } from "./Icon";

/** 0–3 star priority widget. Read-only when `onChange` is omitted. */
export function Priority({
  value,
  onChange,
  size = 16,
}: {
  value: number;
  onChange?: (next: number) => void;
  size?: number;
}) {
  return (
    <fieldset
      aria-label={`Priority ${value} of 3`}
      className="m-0 inline-flex min-w-0 items-center gap-0.5 border-0 p-0"
    >
      {[1, 2, 3].map((n) => {
        const on = n <= value;
        const color = on ? "text-accent" : "text-line-strong";
        const icon = <Icon name="star" size={size} filled={on} />;

        if (!onChange) {
          return (
            <span key={n} className={color}>
              {icon}
            </span>
          );
        }
        return (
          <button
            key={n}
            type="button"
            aria-label={`Set priority to ${n}`}
            aria-pressed={n === value}
            onClick={() => onChange(value === n ? 0 : n)}
            className={`rounded p-0.5 ${color} hover:text-accent`}
          >
            {icon}
          </button>
        );
      })}
    </fieldset>
  );
}
