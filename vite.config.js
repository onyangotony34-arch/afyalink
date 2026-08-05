import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react(), tailwindcss()],
  server: {
    // Proxying /api keeps the browser on a single origin in development, so the
    // refresh cookie (HttpOnly, SameSite=Strict, scoped to /api/auth) is sent
    // with no CORS or cookie-partitioning special cases. In production the API
    // is a sibling host and the backend's CORS_ORIGINS allowlist governs.
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: false,
      },
    },
  },
})
