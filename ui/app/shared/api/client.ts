import createClient from "openapi-fetch";

import { resolveErrorMessage } from "./errors";
import type { paths } from "./schema";

/** Typed API client. Paths are relative to the Fiber `/api` base path. */
export const api = createClient<paths>({ baseUrl: "/api" });

export interface ApiMeta {
	total?: number;
	nextCursor?: string;
}

export interface ApiEnvelope<T> {
	data: T;
	meta?: ApiMeta;
	error?: string;
}

/** A non-2xx API response, or a 2xx response without a body. */
export class ApiError extends Error {
	readonly status: number;
	readonly code: string;

	constructor(status: number, code: string, message?: string) {
		super(message || code);
		this.name = "ApiError";
		this.status = status;
		this.code = code;
	}
}

export function isAuthOrSetupPath(urlOrPath: string): boolean {
	let path = urlOrPath;
	try {
		path = new URL(urlOrPath, "http://localhost").pathname;
	} catch {
		// ponytail: fallback if urlOrPath is not a full URL or valid path
	}
	if (path.startsWith("/api/")) {
		path = path.slice(4);
	} else if (path === "/api") {
		path = "/";
	}
	return (
		path === "/auth" ||
		path.startsWith("/auth/") ||
		path === "/setup" ||
		path.startsWith("/setup/")
	);
}

export type QueryCacheRemover = {
	removeQueries: (filters: { queryKey: readonly unknown[] }) => void;
};

export type UnauthorizedRedirectFn = (to: string) => void;

let activeQueryClient: QueryCacheRemover | null = null;

export function setActiveQueryClient(client: QueryCacheRemover | null) {
	activeQueryClient = client;
}

let redirectFn: UnauthorizedRedirectFn = (to: string) => {
	if (typeof window !== "undefined" && window.location.pathname !== to) {
		window.location.assign(to);
	}
};

export function setUnauthorizedRedirect(fn: UnauthorizedRedirectFn) {
	redirectFn = fn;
}

export function handleUnauthorized(url: string) {
	if (isAuthOrSetupPath(url)) {
		return;
	}
	activeQueryClient?.removeQueries({ queryKey: ["session"] });
	redirectFn("/login");
}

api.use({
	async onResponse({ request, response }) {
		if (response.status === 401) {
			handleUnauthorized(request.url);
		}
	},
});

function extractErrorCode(error: unknown, status: number): string {
	if (typeof error === "object" && error !== null) {
		if ("error" in error && typeof error.error === "string" && error.error) {
			return error.error;
		}
		if (
			"message" in error &&
			typeof error.message === "string" &&
			error.message
		) {
			return error.message;
		}
	}
	return status === 404 ? "RESOURCE_NOT_FOUND" : "REQUEST_FAILED";
}

/** Return the data payload of a successful envelope response, or throw an {@link ApiError}. */
export async function unwrap<T>(
	promiseOrResult:
		| Promise<{ data?: unknown; error?: unknown; response: Response }>
		| { data?: unknown; error?: unknown; response: Response },
	featureErrors?: Record<string, string>,
): Promise<T> {
	const { data, error, response } = await promiseOrResult;
	if (response.ok && data !== undefined) {
		if (typeof data === "object" && data !== null && "data" in data) {
			const payload: unknown = data.data;
			return payload as T;
		}
		return data as T;
	}
	const code = extractErrorCode(error, response.status);
	const message = resolveErrorMessage(code, featureErrors);
	throw new ApiError(response.status, code, message);
}

/** Return data and meta of a successful envelope response, or throw an {@link ApiError}. */
export async function unwrapMeta<T>(
	promiseOrResult:
		| Promise<{ data?: unknown; error?: unknown; response: Response }>
		| { data?: unknown; error?: unknown; response: Response },
	featureErrors?: Record<string, string>,
): Promise<{ data: T; meta?: ApiMeta }> {
	const { data, error, response } = await promiseOrResult;
	if (response.ok && data !== undefined) {
		if (typeof data === "object" && data !== null && "data" in data) {
			const payload: unknown = data.data;
			const metaVal: unknown = "meta" in data ? data.meta : undefined;
			return { data: payload as T, meta: metaVal as ApiMeta | undefined };
		}
		return { data: data as T };
	}
	const code = extractErrorCode(error, response.status);
	const message = resolveErrorMessage(code, featureErrors);
	throw new ApiError(response.status, code, message);
}
