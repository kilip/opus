import * as Sentry from "@sentry/react-router";
import { StrictMode, startTransition } from "react";
import { hydrateRoot } from "react-dom/client";
import { HydratedRouter } from "react-router/dom";

declare global {
	interface Window {
		__CONFIG__?: {
			APP_VERSION?: string;
			SENTRY_DSN?: string;
		};
	}
}

const runtimeConfig =
	(typeof window !== "undefined" && window.__CONFIG__) || {};
const dsn = runtimeConfig.SENTRY_DSN;

if (dsn) {
	Sentry.init({
		dsn,
		release: runtimeConfig.APP_VERSION
			? `opus-ui@${runtimeConfig.APP_VERSION}`
			: undefined,
		integrations: [
			Sentry.reactRouterTracingIntegration(),
			Sentry.replayIntegration(),
		],
		tracesSampleRate: import.meta.env.PROD ? 0.1 : 1.0,
		tracePropagationTargets: [
			/^\//,
			/^https?:\/\/localhost(:\d+)?\/api/,
			/^https?:\/\/127\.0\.0\.1(:\d+)?\/api/,
		],
		replaysSessionSampleRate: 0.1,
		replaysOnErrorSampleRate: 1.0,
	});
}

startTransition(() => {
	hydrateRoot(
		document,
		<StrictMode>
			<HydratedRouter onError={dsn ? Sentry.sentryOnError : undefined} />
		</StrictMode>,
	);
});
