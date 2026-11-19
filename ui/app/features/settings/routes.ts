import { type RouteConfigEntry, route } from "@react-router/dev/routes";

export const settingsRoutes: RouteConfigEntry[] = [
  route("settings", "features/settings/pages/SettingsPage.tsx"),
];
