import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import path from 'path'

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  server: {
    port: 9273,
    proxy: {
      '/api': {
        target: 'http://localhost:9280',
        changeOrigin: true,
        ws: true, // WebSocket/SSE support
      },
    },
  },
  build: {
    outDir: 'dist',
    emptyOutDir: true,
  },
})
