import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import { VitePWA } from 'vite-plugin-pwa'
import path from 'path'

export default defineConfig({
  plugins: [
    react(),
    VitePWA({
      registerType: 'autoUpdate',
      workbox: {
        // Cache revocation lists for up to 72 h (PRD requirement).
        runtimeCaching: [
          {
            urlPattern: /\/v1\/credentials\/revocation/,
            handler: 'StaleWhileRevalidate',
            options: {
              cacheName: 'revocation-cache',
              expiration: { maxAgeSeconds: 72 * 60 * 60 },
            },
          },
        ],
      },
      manifest: {
        name: 'INDIS — سامانه هویت دیجیتال',
        short_name: 'ایندیس',
        description: 'Iran National Digital Identity System — Citizen PWA',
        theme_color: '#1a56db',
        background_color: '#ffffff',
        display: 'standalone',
        orientation: 'portrait',
        dir: 'rtl',
        lang: 'fa',
        icons: [
          { src: '/icon-192.png', sizes: '192x192', type: 'image/png' },
          { src: '/icon-512.png', sizes: '512x512', type: 'image/png' },
        ],
      },
    }),
  ],
  resolve: {
    alias: { '@': path.resolve(__dirname, './src') },
  },
  server: {
    proxy: {
      '/v1': 'http://localhost:8080',
    },
  },
})
