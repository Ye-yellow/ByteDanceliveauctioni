import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

export default defineConfig({
  plugins: [react()],
  server: {
    host: '0.0.0.0',
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
  build: {
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (!id.includes('node_modules')) return undefined;
          if (id.includes('/react/') || id.includes('/react-dom/')) return 'react-vendor';
          if (id.includes('/xgplayer') || id.includes('/xgplayer-hls-live')) return 'player-xg';
          if (id.includes('/hls.js')) return 'player-hls';
          return 'vendor';
        },
      },
    },
  },
});
