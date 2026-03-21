import { defineConfig, loadEnv } from 'vite';
import react from '@vitejs/plugin-react';
import { VitePWA } from 'vite-plugin-pwa';

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '');
  const gatewayUrl = env.VITE_GATEWAY_URL ?? 'http://localhost:8080';
  return {
  plugins: [
    react(),
    VitePWA({
      registerType: 'autoUpdate',
      includeAssets: ['icons/*.svg', 'icons/*.png', 'robots.txt'],
      manifest: false, // using public/manifest.json
      workbox: {
        globPatterns: ['**/*.{js,css,html,ico,png,svg,woff2}'],
        runtimeCaching: [
          {
            // Cache credentials for offline presentation (PRD FR-006: 72h)
            urlPattern: /\/v1\/credential\/.*/i,
            handler: 'StaleWhileRevalidate',
            options: {
              cacheName: 'credential-cache',
              expiration: { maxEntries: 200, maxAgeSeconds: 72 * 60 * 60 },
            },
          },
          {
            // Cache revocation list for offline ZK proof validation (PRD FR-006: 72h)
            urlPattern: /\/v1\/credential\/revocations/i,
            handler: 'StaleWhileRevalidate',
            options: {
              cacheName: 'revocation-cache',
              expiration: { maxEntries: 1, maxAgeSeconds: 72 * 60 * 60 },
            },
          },
          {
            // Cache privacy history for offline viewing
            urlPattern: /\/v1\/privacy\/.*/i,
            handler: 'StaleWhileRevalidate',
            options: {
              cacheName: 'privacy-cache',
              expiration: { maxEntries: 50, maxAgeSeconds: 24 * 60 * 60 },
            },
          },
          {
            urlPattern: /^https:\/\/fonts\.googleapis\.com\/.*/i,
            handler: 'CacheFirst',
            options: {
              cacheName: 'google-fonts-cache',
              expiration: { maxEntries: 10, maxAgeSeconds: 365 * 24 * 60 * 60 },
            },
          },
        ],
      },
    }),
  ],
  resolve: {
    alias: { '@': '/src' },
  },
    server: {
      proxy: {
        // Gateway REST API (v1 prefix)
        '/v1': { target: gatewayUrl, changeOrigin: true },
        // Legacy /api prefix kept for backward compatibility
        '/api': { target: gatewayUrl, changeOrigin: true, rewrite: (p) => p.replace(/^\/api/, '') },
      },
    },
  };
});
