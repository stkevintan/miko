import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import tailwindcss from '@tailwindcss/vite'

// Enable top-level await in dependencies like @novnc/novnc by targeting modern output.
export default defineConfig({
  plugins: [tailwindcss(), vue()],
  server: {
    port: 5173,
    strictPort: true,
    proxy: {
      '/api': {
        target: 'http://localhost:8082',
        changeOrigin: true,
      },
    },
  },
  build: {
    target: 'esnext',
  },
})
