import type { ReactNode } from "react";
import { Navigate, Outlet } from "react-router";

import { useSession, useSetupStatus } from "./session";

export function RequireAuth({ children }: { children?: ReactNode }) {
	const setupQuery = useSetupStatus();
	const sessionQuery = useSession();

	if (setupQuery.isLoading) {
		return null;
	}

	if (setupQuery.data?.needs_setup) {
		return <Navigate to="/onboarding" replace />;
	}

	if (sessionQuery.isLoading) {
		return null;
	}

	if (!sessionQuery.data || sessionQuery.isError) {
		return <Navigate to="/login" replace />;
	}

	return (children ?? <Outlet />) as ReactNode;
}

export default RequireAuth;
