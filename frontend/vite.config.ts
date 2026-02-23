import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  server: {
    port: 3000,
    proxy: {
      '/chat': {
        target: 'http://localhost:8080',
        changeOrigin: true
      },
      '/intent': {
        target: 'http://localhost:8080',
        changeOrigin: true
      },
      '/session': {
        target: 'http://localhost:8080',
        changeOrigin: true
      },
      '/ticket': {
        target: 'http://localhost:8080',
        changeOrigin: true
      },
      '/knowledge': {
        target: 'http://localhost:8080',
        changeOrigin: true
      }
    }
  }
})
