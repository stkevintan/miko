import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

// Enable top-level await in dependencies like @novnc/novnc by targeting modern output.
export default defineConfig({
  plugins: [vue()],
  server: {
    port: 5173,
    strictPort: true,
  },
  build: {
    target: 'esnext',
  },
})
