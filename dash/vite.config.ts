import path from 'node:path';
import babel from '@rolldown/plugin-babel';
import tailwindcss from '@tailwindcss/vite';
import { TanStackRouterVite } from '@tanstack/router-plugin/vite';
import react, { reactCompilerPreset } from '@vitejs/plugin-react';
import { VitePWA } from 'vite-plugin-pwa';
import { defineConfig } from 'vitest/config';

export default defineConfig({
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  plugins: [
    tailwindcss(),
    TanStackRouterVite(),
    react(),
    babel({ presets: [reactCompilerPreset()] }),
    VitePWA({
      registerType: 'autoUpdate',
      workbox: {
        globPatterns: ['**/*.{js,css,html,ico,png,svg,woff2}'],
        runtimeCaching: [
          {
            urlPattern: /^http:\/\/localhost:8080\/api\/.*/,
            handler: 'NetworkFirst',
            options: {
              cacheName: 'opus-api-cache',
              networkTimeoutSeconds: 5,
              expiration: { maxEntries: 200, maxAgeSeconds: 86400 },
              cacheableResponse: { statuses: [0, 200] },
            },
          },
          {
            urlPattern: /^http:\/\/localhost:8080\/api\/.*/,
            method: 'POST',
            handler: 'NetworkOnly',
            options: {
              backgroundSync: {
                name: 'opus-mutation-queue',
                options: { maxRetentionTime: 24 * 60 },
              },
            },
          },
        ],
      },
      manifest: {
        name: 'Opus Dash',
        short_name: 'Opus',
        description: 'Opus autonomous AI assistant dashboard',
        theme_color: '#f5f4f0',
        background_color: '#f5f4f0',
        display: 'standalone',
        scope: '/',
        start_url: '/',
      },
    }),
  ],
  test: {
    globals: true,
    environment: 'jsdom',
    setupFiles: './src/test/setup.ts',
    exclude: ['**/node_modules/**', '**/e2e/**', '**/dist/**'],
  },
});
