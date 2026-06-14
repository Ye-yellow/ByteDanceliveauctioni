import { defineConfig } from 'vite';
import type { Plugin, ViteDevServer } from 'vite';
import react from '@vitejs/plugin-react';
import type { IncomingMessage, ServerResponse } from 'node:http';
import { createHash } from 'node:crypto';
import { createReadStream, createWriteStream, existsSync, mkdirSync, renameSync, statSync, unlinkSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { Readable } from 'node:stream';
import type { ReadableStream } from 'node:stream/web';

const DOUYIN_VIDEO_CACHE_DIR = join(tmpdir(), 'live-auction-douyin-video-cache');

async function resolveDouyinVideoLocation(target: string, req: IncomingMessage) {
  let lastError: unknown;
  for (let attempt = 0; attempt < 3; attempt += 1) {
    const controller = new AbortController();
    const timer = setTimeout(() => controller.abort(), 6000);
    try {
      const response = await fetch(target, {
        redirect: 'manual',
        signal: controller.signal,
        headers: {
          'user-agent': req.headers['user-agent'] || 'Mozilla/5.0',
          referer: 'https://www.douyin.com/',
          accept: '*/*',
        },
      });
      return response.headers.get('location') || '';
    } catch (error) {
      lastError = error;
    } finally {
      clearTimeout(timer);
    }
  }
  throw lastError;
}

async function fetchDouyinVideo(target: string, req: IncomingMessage) {
  let lastError: unknown;
  for (let attempt = 0; attempt < 3; attempt += 1) {
    const controller = new AbortController();
    const timer = setTimeout(() => controller.abort(), 12000);
    try {
      return await fetch(target, {
        signal: controller.signal,
        headers: {
          'user-agent': req.headers['user-agent'] || 'Mozilla/5.0',
          referer: 'https://www.douyin.com/',
          accept: '*/*',
          ...(req.headers.range ? { range: req.headers.range } : {}),
        },
      });
    } catch (error) {
      lastError = error;
    } finally {
      clearTimeout(timer);
    }
  }
  throw lastError;
}

function douyinVideoCachePath(target: string) {
  mkdirSync(DOUYIN_VIDEO_CACHE_DIR, { recursive: true });
  return join(DOUYIN_VIDEO_CACHE_DIR, `${createHash('sha1').update(target).digest('hex')}.mp4`);
}

function serveCachedVideo(req: IncomingMessage, res: ServerResponse, filePath: string) {
  const size = statSync(filePath).size;
  const range = req.headers.range;
  res.setHeader('accept-ranges', 'bytes');
  res.setHeader('content-type', 'video/mp4');
  res.setHeader('cache-control', 'public, max-age=3600');

  if (range) {
    const match = /^bytes=(\d*)-(\d*)$/.exec(range);
    const start = match?.[1] ? Number(match[1]) : 0;
    const end = match?.[2] ? Number(match[2]) : size - 1;
    if (Number.isFinite(start) && Number.isFinite(end) && start <= end && start < size) {
      const chunkEnd = Math.min(end, size - 1);
      res.statusCode = 206;
      res.setHeader('content-range', `bytes ${start}-${chunkEnd}/${size}`);
      res.setHeader('content-length', String(chunkEnd - start + 1));
      createReadStream(filePath, { start, end: chunkEnd }).pipe(res);
      return;
    }
  }

  res.statusCode = 200;
  res.setHeader('content-length', String(size));
  createReadStream(filePath).pipe(res);
}

function douyinVideoProxyPlugin(): Plugin {
  return {
    name: 'douyin-video-proxy',
    configureServer(server: ViteDevServer) {
      server.middlewares.use('/__douyin_video_proxy', async (req: IncomingMessage, res: ServerResponse) => {
        const requestUrl = new URL(req.url || '/', 'http://localhost');
        const target = requestUrl.searchParams.get('url');
        if (!target || !/^https:\/\/www\.douyin\.com\/aweme\/v1\/play\//.test(target)) {
          res.statusCode = 400;
          res.end('invalid video url');
          return;
        }

        try {
          const cachePath = douyinVideoCachePath(target);
          if (existsSync(cachePath) && statSync(cachePath).size > 0) {
            serveCachedVideo(req, res, cachePath);
            return;
          }

          const resolvedTarget = await resolveDouyinVideoLocation(target, req).catch(() => '') || target;
          const upstream = await fetchDouyinVideo(resolvedTarget, req);
          if (!upstream.ok) {
            res.statusCode = upstream.status || 502;
            res.end(`video upstream failed: ${upstream.status}`);
            return;
          }

          res.statusCode = upstream.status;
          for (const header of ['content-type', 'content-length', 'content-range', 'accept-ranges']) {
            const value = upstream.headers.get(header);
            if (value) res.setHeader(header, value);
          }
          res.setHeader('cache-control', 'no-store');

          if (!upstream.body) {
            res.end();
            return;
          }
          const stream = Readable.fromWeb(upstream.body as ReadableStream<Uint8Array>);
          if (!req.headers.range) {
            const cacheTempPath = `${cachePath}.${process.pid}.${Date.now()}.tmp`;
            const cacheWriter = createWriteStream(cacheTempPath);
            cacheWriter.on('finish', () => {
              try {
                if (existsSync(cacheTempPath) && statSync(cacheTempPath).size > 0) renameSync(cacheTempPath, cachePath);
                else if (existsSync(cacheTempPath)) unlinkSync(cacheTempPath);
              } catch {
                // The cache is an optimization; playback should not depend on it.
              }
            });
            stream.pipe(cacheWriter);
          }
          stream.pipe(res);
        } catch (error) {
          res.statusCode = 502;
          res.end(error instanceof Error ? error.message : 'video proxy failed');
        }
      });
    },
  };
}

export default defineConfig({
  plugins: [react(), douyinVideoProxyPlugin()],
  server: {
    host: '0.0.0.0',
    allowedHosts: true,
    watch: {
      usePolling: true,
      interval: 300,
    },
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
          return 'vendor';
        },
      },
    },
  },
});
