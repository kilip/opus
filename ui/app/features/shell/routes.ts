import {
  index,
  layout,
  type RouteConfigEntry,
  route,
} from "@react-router/dev/routes";

/** Wraps feature routes in the application shell and adds the fallbacks. */
export function shellRoutes(children: RouteConfigEntry[]): RouteConfigEntry {
  return layout("features/shell/ShellLayout.tsx", [
    index("features/shell/HomeRedirect.tsx"),
    ...children,
    route("*", "shared/pages/NotFound.tsx"),
  ]);
}
