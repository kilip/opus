import type { ReactNode } from "react";
import { Link } from "react-router";

export type Crumb = { label: string; to?: string };

export function Breadcrumbs({ items }: { items: Crumb[] }) {
  return (
    <nav aria-label="Breadcrumb" className="min-w-0">
      <ol className="flex items-center gap-1.5 text-base">
        {items.map((item, i) => {
          const last = i === items.length - 1;
          return (
            <li
              key={item.to ?? item.label}
              className="flex min-w-0 items-center gap-1.5"
            >
              {item.to && !last ? (
                <Link
                  to={item.to}
                  className="truncate text-primary hover:underline"
                >
                  {item.label}
                </Link>
              ) : (
                <span
                  aria-current={last ? "page" : undefined}
                  className="truncate font-semibold"
                >
                  {item.label}
                </span>
              )}
              {!last && (
                <span aria-hidden="true" className="text-muted">
                  /
                </span>
              )}
            </li>
          );
        })}
      </ol>
    </nav>
  );
}

/**
 * Control panel: actions + breadcrumb on the left,
 * search / view switcher on the right.
 */
export function ControlPanel({
  breadcrumbs,
  actions,
  right,
}: {
  breadcrumbs: Crumb[];
  actions?: ReactNode;
  right?: ReactNode;
}) {
  return (
    <div className="flex flex-wrap items-center gap-x-4 gap-y-2 border-b border-line bg-surface px-4 py-2">
      <div className="flex min-w-0 items-center gap-3">
        {actions}
        <Breadcrumbs items={breadcrumbs} />
      </div>
      {right && (
        <div className="ml-auto flex flex-wrap items-center gap-3">{right}</div>
      )}
    </div>
  );
}
