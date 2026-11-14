import { Link, Outlet, useLocation } from "react-router";

import { apps } from "~/apps";
import { Avatar } from "~/shared/components/Avatar";
import { Icon, type IconName } from "~/shared/components/Icon";
import { Popover } from "~/shared/components/Popover";
import { type ThemePref, useTheme } from "~/shared/theme";

const navButton =
  "inline-flex h-9 items-center gap-2 rounded-md px-2.5 text-navbar-fg hover:bg-white/15 aria-expanded:bg-white/15";

export default function ShellLayout() {
  const { pathname } = useLocation();
  const current = apps.find((a) => pathname.startsWith(a.path));

  return (
    <div className="flex h-dvh flex-col">
      <header className="flex h-12 shrink-0 items-center gap-1 bg-navbar px-2 text-navbar-fg">
        <AppSwitcher />
        <Link
          to={current?.path ?? "/"}
          className="rounded-md px-2 py-1 text-base font-semibold hover:bg-white/15"
        >
          {current?.name ?? "Opus"}
        </Link>
        <div className="ml-auto">
          <UserMenu />
        </div>
      </header>
      <main className="flex min-h-0 flex-1 flex-col">
        <Outlet />
      </main>
    </div>
  );
}

function AppSwitcher() {
  return (
    <Popover
      className="w-80 p-3"
      renderTrigger={(props) => (
        <button
          type="button"
          aria-label="Apps"
          className={`${navButton} px-2`}
          {...props}
        >
          <Icon name="grid" size={20} />
        </button>
      )}
    >
      {(close) => (
        <ul className="grid grid-cols-3 gap-2">
          {apps.map((app) => (
            <li key={app.id}>
              {app.enabled ? (
                <Link
                  to={app.path}
                  onClick={close}
                  className="flex flex-col items-center gap-1.5 rounded-lg p-2 hover:bg-surface-2"
                >
                  <AppTile icon={app.icon} />
                  <span className="font-medium">{app.name}</span>
                </Link>
              ) : (
                <div
                  aria-disabled="true"
                  className="flex flex-col items-center gap-1.5 rounded-lg p-2 opacity-50"
                >
                  <AppTile icon={app.icon} />
                  <span className="font-medium">{app.name}</span>
                  <span className="-mt-1 text-xs text-muted">Soon</span>
                </div>
              )}
            </li>
          ))}
        </ul>
      )}
    </Popover>
  );
}

function AppTile({ icon }: { icon: IconName }) {
  return (
    <span className="inline-flex size-12 items-center justify-center rounded-xl bg-primary-soft text-primary-ink">
      <Icon name={icon} size={24} />
    </span>
  );
}

const THEMES: { id: ThemePref; label: string; icon: IconName }[] = [
  { id: "light", label: "Light", icon: "sun" },
  { id: "dark", label: "Dark", icon: "moon" },
  { id: "system", label: "System", icon: "monitor" },
];

function UserMenu() {
  const { pref, setPref } = useTheme();

  return (
    <Popover
      align="right"
      className="w-60 p-2"
      renderTrigger={(props) => (
        <button
          type="button"
          aria-label="User menu"
          className={navButton}
          {...props}
        >
          <Avatar name="You" size={26} />
          <Icon name="chevronDown" size={14} />
        </button>
      )}
    >
      <p className="px-2 pt-1 pb-1.5 text-xs font-semibold tracking-wide text-muted uppercase">
        Theme
      </p>
      <fieldset
        aria-label="Theme"
        className="m-0 mb-2 grid min-w-0 grid-cols-3 gap-1 border-0 p-0"
      >
        {THEMES.map((t) => (
          <button
            key={t.id}
            type="button"
            aria-pressed={pref === t.id}
            onClick={() => setPref(t.id)}
            className="flex flex-col items-center gap-1 rounded-md border border-transparent px-2 py-2 text-xs hover:bg-surface-2 aria-pressed:border-primary aria-pressed:bg-primary-soft aria-pressed:text-primary-ink"
          >
            <Icon name={t.icon} size={18} />
            {t.label}
          </button>
        ))}
      </fieldset>
      <Link
        to="/login"
        className="flex items-center gap-2 rounded-md px-2 py-2 hover:bg-surface-2"
      >
        <Icon name="logout" size={16} />
        Log out
      </Link>
    </Popover>
  );
}
