import { defaultCache } from "@serwist/next/worker";
import { Serwist, NetworkOnly, StaleWhileRevalidate, CacheFirst } from "serwist";

declare const self: ServiceWorkerGlobalScope & {
  __SW_MANIFEST: (string | { url: string; revision: string | null })[];
};

const serwist = new Serwist({
  precacheEntries: self.__SW_MANIFEST,
  skipWaiting: true,
  clientsClaim: true,
  navigationPreload: true,
  runtimeCaching: [
    {
      matcher: /^\/api\/.*/,
      handler: new NetworkOnly(),
    },
    {
      matcher: /\.(?:js|css|woff2)$/,
      handler: new StaleWhileRevalidate(),
    },
    {
      matcher: /^\/offline$/,
      handler: new CacheFirst(),
    },
    ...defaultCache,
  ],
  fallbacks: {
    entries: [
      {
        url: "/offline",
        matcher: ({ request }) => request.mode === "navigate",
      },
    ],
  },
});

serwist.addEventListeners();
