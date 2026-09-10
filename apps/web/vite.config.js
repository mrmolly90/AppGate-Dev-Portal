import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [react()],
  server: {
    port: 3000,
    proxy: {
      '/v1': {
        target: process.env.DP_API_PROXY || 'http://localhost:8080',
        changeOrigin: true,
      },
      '/health': {
        target: process.env.DP_API_PROXY || 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
  build: {
    outDir: 'dist',
    sourcemap: false,
    rollupOptions: {
      onLog(level, log, handler) {
        // Silence benign third-party (zod) comment-annotation warnings.
        // These are informational only — the build output is unaffected.
        if (level === 'warn' && log.code === 'COMMENT_ANNOTATION') {
          return
        }
        handler(level, log)
      },
    },
  },
})