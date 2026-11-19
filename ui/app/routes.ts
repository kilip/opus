import { type RouteConfig, route } from "@react-router/dev/routes";

import { authRoutes, authSettingsRoutes } from "./features/auth/routes";
import { dashboardRoutes } from "./features/dashboard/routes";
import { nodesRoutes } from "./features/nodes/routes";
import { projectsRoutes } from "./features/projects/routes";
import { runsRoutes } from "./features/runs/routes";
import { shellRoutes } from "./features/shell/routes";
import { tasksRoutes } from "./features/tasks/routes";

/**
 * Route aggregator. Each feature owns its route definitions in
 * `features/<name>/routes.ts`; this file only composes them.
 *
 * To add a feature: create `features/<name>/routes.ts`, then add it below.
 */
export default [
  // Public routes (no application shell).
  ...authRoutes,

  // Everything inside the shell (navbar, app switcher).
  shellRoutes([
    ...dashboardRoutes,
    ...tasksRoutes,
    ...projectsRoutes,
    ...nodesRoutes,
    ...runsRoutes,
    ...authSettingsRoutes,
  ]),

  route("*", "shared/pages/NotFound.tsx"),
] satisfies RouteConfig;
