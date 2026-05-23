import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

export default defineConfig({
  plugins: [react()],
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
