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
const dsn =
  runtimeConfig.SENTRY_DSN ||
  "https://957993bc03a49ec153c832a5f86ba579@o4507164991684608.ingest.us.sentry.io/4512431358935040";

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

startTransition(() => {
  hydrateRoot(
    document,
    <StrictMode>
      <HydratedRouter onError={Sentry.sentryOnError} />
    </StrictMode>,
  );
});
