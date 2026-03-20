import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import { VitePWA } from 'vite-plugin-pwa'

export default defineConfig({
  plugins: [
    react(),
    VitePWA({
      registerType: 'autoUpdate',
      manifest: {
        name: 'INDIS Verifier Terminal',
        short_name: 'INDIS Verifier',
        theme_color: '#111111',
        background_color: '#111111',
        display: 'fullscreen',
        orientation: 'portrait',
        icons: [
          { src: '/icons/icon-192.png', sizes: '192x192', type: 'image/png' },
          { src: '/icons/icon-512.png', sizes: '512x512', type: 'image/png' },
        ],
      },
      workbox: {
        globPatterns: ['**/*.{js,css,html,ico,png,svg,woff2}'],
        runtimeCaching: [
          {
            // Cache revocation list for offline ZK proof validation (PRD FR-006: 72h)
            urlPattern: /\/v1\/credential\/revocations/i,
            handler: 'StaleWhileRevalidate',
            options: {
              cacheName: 'revocation-cache',
              expiration: { maxEntries: 1, maxAgeSeconds: 72 * 60 * 60 },
            },
          },
        ],
      },
    }),
  ],
  server: {
    proxy: {
      '/v1': 'http://localhost:8080',
    },
  },
})
