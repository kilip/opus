import { type RouteConfigEntry, route } from "@react-router/dev/routes";

export const authRoutes: RouteConfigEntry[] = [
  route("login", "features/auth/LoginPage.tsx"),
];
