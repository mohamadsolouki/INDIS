import { defineConfig, loadEnv } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '')
  const gatewayUrl = env.VITE_GATEWAY_URL ?? 'http://localhost:8080'
  return {
    plugins: [react()],
    server: {
      proxy: {
        '/v1': gatewayUrl,
        '/graphql': gatewayUrl,
      },
    },
  }
})
