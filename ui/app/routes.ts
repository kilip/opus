import type { RouteConfig } from "@react-router/dev/routes";

import { authRoutes } from "./features/auth/routes";
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
  shellRoutes([...tasksRoutes]),
] satisfies RouteConfig;
