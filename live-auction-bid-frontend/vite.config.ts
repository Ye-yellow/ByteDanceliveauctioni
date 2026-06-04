import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

export default defineConfig({
  plugins: [react()],
  build: {
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (id.includes('/node_modules/react') || id.includes('/node_modules/react-dom')) return 'react';
          if (id.includes('/node_modules/lucide-react')) return 'icons';
          return undefined;
        },
      },
    },
  },
  server: {
    host: '0.0.0.0',
    allowedHosts: true,
    proxy: {
      '/api': {
        target: process.env.VITE_DEV_PROXY_TARGET || 'http://127.0.0.1:18080',
        changeOrigin: true,
      },
      '/healthz': {
        target: process.env.VITE_DEV_PROXY_TARGET || 'http://127.0.0.1:18080',
        changeOrigin: true,
      },
      '/ws': {
        target: process.env.VITE_DEV_WS_PROXY_TARGET || 'ws://127.0.0.1:18080',
        ws: true,
        changeOrigin: true,
      },
    },
  },
});
