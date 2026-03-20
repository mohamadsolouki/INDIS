import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import { VitePWA } from 'vite-plugin-pwa';

export default defineConfig({
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
            // Also cache local dev gateway responses for offline use
            urlPattern: /\/v1\/credential\/.*/i,
            handler: 'StaleWhileRevalidate',
            options: {
              cacheName: 'credential-cache',
              expiration: { maxEntries: 200, maxAgeSeconds: 72 * 60 * 60 },
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
      '/v1': { target: 'http://localhost:8080', changeOrigin: true },
      // Legacy /api prefix kept for backward compatibility
      '/api': { target: 'http://localhost:8080', changeOrigin: true, rewrite: (p) => p.replace(/^\/api/, '') },
    },
  },
});
