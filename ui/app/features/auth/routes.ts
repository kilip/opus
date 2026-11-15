import { type RouteConfigEntry, route } from "@react-router/dev/routes";

export const authRoutes: RouteConfigEntry[] = [
	route("login", "features/auth/pages/LoginPage.tsx"),
	route("onboarding", "features/auth/pages/OnboardingPage.tsx"),
];
