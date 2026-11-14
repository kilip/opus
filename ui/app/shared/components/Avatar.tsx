import { Icon } from "./Icon";

const HUES = [18, 28, 38, 8, 48, 350, 24];

function hash(text: string) {
  let h = 0;
  for (const ch of text) h = (h * 31 + ch.charCodeAt(0)) >>> 0;
  return h;
}

/** Agents are named `<role>-agent` and get a bot glyph instead of initials. */
export function isAgent(name: string) {
  return name.endsWith("-agent");
}

export function Avatar({ name, size = 24 }: { name: string; size?: number }) {
  const hue = HUES[hash(name) % HUES.length];
  const initials = name
    .split(/[\s-]+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((w) => w[0]?.toUpperCase())
    .join("");

  return (
    <span
      role="img"
      aria-label={name}
      title={name}
      style={{
        width: size,
        height: size,
        fontSize: size * 0.42,
        background: `hsl(${hue} 42% 38%)`,
      }}
      className="inline-flex shrink-0 items-center justify-center rounded-full font-semibold text-white"
    >
      {isAgent(name) ? (
        <Icon name="bot" size={Math.round(size * 0.62)} />
      ) : (
        initials
      )}
    </span>
  );
}
