import { type RouteConfigEntry, route } from "@react-router/dev/routes";

export const runsRoutes: RouteConfigEntry[] = [
  route("runs", "features/runs/pages/RunsPage.tsx"),
  route("runs/:id", "features/runs/pages/RunDetailPage.tsx"),
];
