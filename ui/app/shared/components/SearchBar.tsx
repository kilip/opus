import { Icon } from "./Icon";
import { Popover } from "./Popover";

export type FilterDef = { id: string; label: string };

type SearchBarProps = {
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
  filters?: FilterDef[];
  active?: ReadonlySet<string>;
  onToggleFilter?: (id: string) => void;
};

/** Search input with removable facet chips and a filter dropdown. */
export function SearchBar({
  value,
  onChange,
  placeholder = "Search…",
  filters = [],
  active = new Set<string>(),
  onToggleFilter,
}: SearchBarProps) {
  const chips = filters.filter((f) => active.has(f.id));

  return (
    <div className="flex h-8 w-full items-center gap-1.5 rounded-md border border-line-strong bg-surface px-2 focus-within:border-primary focus-within:shadow-[0_0_0_3px_var(--primary-soft)] sm:w-96">
      <Icon name="search" size={16} className="shrink-0 text-muted" />
      {chips.map((chip) => (
        <span
          key={chip.id}
          className="inline-flex shrink-0 items-center gap-1 rounded bg-primary-soft py-0.5 pr-1 pl-2 text-xs font-medium text-primary-ink"
        >
          {chip.label}
          <button
            type="button"
            aria-label={`Remove filter ${chip.label}`}
            onClick={() => onToggleFilter?.(chip.id)}
            className="rounded p-0.5 hover:bg-primary hover:text-primary-fg"
          >
            <Icon name="x" size={12} />
          </button>
        </span>
      ))}
      <input
        type="search"
        aria-label="Search"
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder={chips.length ? "" : placeholder}
        className="min-w-0 flex-1 bg-transparent outline-none placeholder:text-muted [&::-webkit-search-cancel-button]:hidden"
      />
      {filters.length > 0 && (
        <Popover
          align="right"
          className="w-56 p-1"
          renderTrigger={(props) => (
            <button
              type="button"
              aria-label="Filters"
              className="rounded p-1 text-muted hover:bg-surface-2 hover:text-fg"
              {...props}
            >
              <Icon name="chevronDown" size={16} />
            </button>
          )}
        >
          <p className="px-2 pt-1.5 pb-1 text-xs font-semibold tracking-wide text-muted uppercase">
            Filters
          </p>
          {filters.map((f) => {
            const on = active.has(f.id);
            return (
              <button
                key={f.id}
                type="button"
                role="menuitemcheckbox"
                aria-checked={on}
                onClick={() => onToggleFilter?.(f.id)}
                className="flex w-full items-center gap-2 rounded px-2 py-1.5 text-left hover:bg-surface-2"
              >
                <span className="inline-flex w-4 justify-center text-primary">
                  {on && <Icon name="check" size={14} />}
                </span>
                {f.label}
              </button>
            );
          })}
        </Popover>
      )}
    </div>
  );
}
