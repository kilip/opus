import { reactRouter } from "@react-router/dev/vite";
import tailwindcss from "@tailwindcss/vite";
import { defineConfig } from "vite";

export default defineConfig({
  plugins: [tailwindcss(), reactRouter()],
  resolve: {
    tsconfigPaths: true,
  },
  server: {
    host: true,
    allowedHosts: process.env.ALLOWED_HOSTS ? process.env.ALLOWED_HOSTS.split(',') : true,
    proxy: {
      // Forward any request starting with /api to your Fiber backend
      '/api': {
        target: 'http://127.0.0.1:7880', // Must match the port in internal/app/app.go. Use 127.0.0.1 instead of localhost to avoid IPv6 issues
        changeOrigin: true,
        // Uncomment below if your Fiber routes don't have the /api prefix:
        // rewrite: (path) => path.replace(/^\/api/, ''),
      },
    },
  }
});
