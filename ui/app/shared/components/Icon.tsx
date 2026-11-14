import type { SVGProps } from "react";

const paths = {
  grid: "M4 4h6v6H4z M14 4h6v6h-6z M4 14h6v6H4z M14 14h6v6h-6z",
  search: "M11 4a7 7 0 1 0 0 14a7 7 0 0 0 0-14z M20 20l-4-4",
  filter: "M3 5h18l-7 8v6l-4-2v-4z",
  kanban: "M4 4h5v16H4z M10 4h5v10h-5z M16 4h4v13h-4z",
  list: "M8 6h13 M8 12h13 M8 18h13 M3 6h.01 M3 12h.01 M3 18h.01",
  star: "M12 3l2.8 5.7 6.2.9-4.5 4.4 1.1 6.2L12 17.2 6.4 20.2l1.1-6.2L3 9.6l6.2-.9z",
  plus: "M12 5v14 M5 12h14",
  chevronDown: "M6 9l6 6 6-6",
  chevronRight: "M9 6l6 6-6 6",
  arrowRight: "M5 12h14 M13 6l6 6-6 6",
  sun: "M12 8a4 4 0 1 0 0 8a4 4 0 0 0 0-8z M12 2v2 M12 20v2 M4.9 4.9l1.4 1.4 M17.7 17.7l1.4 1.4 M2 12h2 M20 12h2 M4.9 19.1l1.4-1.4 M17.7 6.3l1.4-1.4",
  moon: "M20 14.5A8 8 0 0 1 9.5 4 8 8 0 1 0 20 14.5z",
  monitor: "M3 5h18v11H3z M8 20h8 M12 16v4",
  user: "M12 4a4 4 0 1 0 0 8a4 4 0 0 0 0-8z M4 20c0-4 4-6 8-6s8 2 8 6",
  logout: "M9 4H5v16h4 M16 8l4 4-4 4 M20 12H9",
  clock: "M12 3a9 9 0 1 0 0 18a9 9 0 0 0 0-18z M12 7v5l3 2",
  send: "M4 12l16-8-6 16-3-7z",
  note: "M5 4h14v16H5z M9 9h6 M9 13h6",
  check: "M5 12l5 5 9-10",
  x: "M6 6l12 12 M18 6L6 18",
  tasks: "M4 6l2 2 3-3 M12 7h8 M4 14l2 2 3-3 M12 15h8",
  bot: "M7 8h10a2 2 0 0 1 2 2v7a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2v-7a2 2 0 0 1 2-2z M12 4v4 M9 13h.01 M15 13h.01 M3 13v2 M21 13v2",
  activity: "M3 12h4l3-8 4 16 3-8h4",
  settings: "M4 7h10 M18 7h2 M4 17h2 M10 17h10 M16 5v4 M8 15v4",
} as const;

export type IconName = keyof typeof paths;

type IconProps = {
  name: IconName;
  size?: number;
  filled?: boolean;
} & Omit<SVGProps<SVGSVGElement>, "name">;

export function Icon({
  name,
  size = 18,
  filled = false,
  className,
  ...rest
}: IconProps) {
  return (
    <svg
      viewBox="0 0 24 24"
      width={size}
      height={size}
      fill={filled ? "currentColor" : "none"}
      stroke="currentColor"
      strokeWidth={1.8}
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
      focusable="false"
      className={className}
      {...rest}
    >
      <path d={paths[name]} />
    </svg>
  );
}
