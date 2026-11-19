import { index, layout, type RouteConfigEntry } from "@react-router/dev/routes";

/** Wraps feature routes in the application shell. */
export function shellRoutes(children: RouteConfigEntry[]): RouteConfigEntry {
  return layout("shared/auth/RequireAuth.tsx", [
    layout("features/shell/ShellLayout.tsx", [
      index("features/shell/HomeRedirect.tsx"),
      ...children,
    ]),
  ]);
}
