import react from '@vitejs/plugin-react';
import { defineConfig } from 'vitest/config';

// In development the browser talks to Vite, which forwards /api to the backend.
// In Kubernetes the Ingress does the same job, so the frontend always uses the
// relative path /api and the browser sees a single origin (no CORS).
const apiTarget = process.env.VITE_DEV_API_TARGET ?? 'http://localhost:8080';

export default defineConfig({
  plugins: [react()],
  server: {
    proxy: { '/api': apiTarget },
  },
  test: {
    environment: 'jsdom',
    globals: true,
    setupFiles: './src/test/setup.ts',
    coverage: {
      provider: 'v8',
      include: ['src/**'],
      exclude: ['src/main.tsx', 'src/test/**', '**/*.test.*'],
      reporter: ['text', 'html'],
    },
  },
});
