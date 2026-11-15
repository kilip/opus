/** Query key factory for auth and session queries. */
export const authKeys = {
	all: ["session"] as const,
	me: () => [...authKeys.all, "me"] as const,
	setupStatus: () => [...authKeys.all, "setup-status"] as const,
	providers: () => [...authKeys.all, "providers"] as const,
	accounts: () => [...authKeys.all, "accounts"] as const,
};
