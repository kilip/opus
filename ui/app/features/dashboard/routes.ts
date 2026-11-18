import { type RouteConfigEntry, route } from "@react-router/dev/routes";

export const dashboardRoutes: RouteConfigEntry[] = [
	route("dashboard", "features/dashboard/pages/DashboardPage.tsx"),
];
