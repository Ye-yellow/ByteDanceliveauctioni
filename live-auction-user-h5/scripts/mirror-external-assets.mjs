import { mkdir, readdir, readFile, stat, writeFile } from 'node:fs/promises';
import { createHash } from 'node:crypto';
import path from 'node:path';

const ROOT = process.cwd();
const SCAN_ROOTS = ['src', 'public/data'];
const OUTPUT_DIR = path.join(ROOT, 'tmp/external-assets-mirror/files');
const MANIFEST_PATH = path.join(ROOT, 'tmp/external-assets-mirror/manifest.json');
const TOS_ROOT = 'https://liveauction.tos-cn-beijing.volces.com/douyin-h5';
const TOS_EXTERNAL_BASE = `${TOS_ROOT}/external`;
const TOS_IMAGE_BASE = `${TOS_ROOT}/images`;
const FALLBACK_AVATAR = `${TOS_IMAGE_BASE}/avatar-large-71158770-1s69laf.jpeg`;
const FALLBACK_IMAGE = `${TOS_IMAGE_BASE}/Rp7m-e4N5q9iIFik9-K1k.png`;

const SOURCE_EXTENSIONS = new Set(['.css', '.json', '.ts', '.tsx']);
const URL_PATTERN = /https?:\/\/[^\s"'<>\\]+/g;
const EXTERNAL_HOST_PATTERN = /(?:^|\/\/|[/?&.:@])(?:[^/?&\s"'<>\\]*\.)?(dy\.2study\.top|douyinpic\.com|douyin\.com|douyinstatic\.com|iesdouyin\.com|bytednsdoc\.com|byteimg\.com|ecombdimg\.com|ecombdstatic\.com)(?:[/?&:]|$)/i;
const DOWNLOAD_CONCURRENCY = 8;
const REQUEST_TIMEOUT_MS = 15000;

function sha1(value) {
  return createHash('sha1').update(value).digest('hex');
}

async function listFiles(entry) {
  const absolute = path.join(ROOT, entry);
  const info = await stat(absolute).catch(() => null);
  if (!info) return [];
  if (info.isFile()) return SOURCE_EXTENSIONS.has(path.extname(absolute)) ? [absolute] : [];

  const children = await readdir(absolute, { withFileTypes: true });
  const files = await Promise.all(children.map((child) => listFiles(path.join(entry, child.name))));
  return files.flat();
}

function cleanUrl(rawUrl) {
  return rawUrl.replace(/[),.;`]+$/g, '');
}

function isKnownExternalAsset(url) {
  if (url.includes('${')) return false;
  let parsed;
  try {
    parsed = new URL(url);
  } catch {
    return false;
  }
  const host = parsed.hostname.toLowerCase();
  if (host === 'dy.2study.top') return parsed.pathname.startsWith('/images/');
  return host.endsWith('douyinpic.com')
    || host.endsWith('bytednsdoc.com')
    || host.endsWith('byteimg.com')
    || host.endsWith('ecombdimg.com')
    || host.endsWith('ecombdstatic.com');
}

function existingTosImageUrl(url) {
  try {
    const parsed = new URL(url);
    if (parsed.hostname !== 'dy.2study.top' || !parsed.pathname.startsWith('/images/')) return '';
    return `${TOS_IMAGE_BASE}/${path.posix.basename(parsed.pathname)}`;
  } catch {
    return '';
  }
}

function fallbackUrl(url) {
  return /avatar|aweme-avatar|100x100|168x168|200x200|300x300/i.test(url) ? FALLBACK_AVATAR : FALLBACK_IMAGE;
}

function extensionFromContentType(contentType) {
  const normalized = contentType.split(';')[0].trim().toLowerCase();
  if (normalized === 'image/jpeg') return '.jpg';
  if (normalized === 'image/png') return '.png';
  if (normalized === 'image/webp') return '.webp';
  if (normalized === 'image/gif') return '.gif';
  if (normalized === 'image/avif') return '.avif';
  if (normalized === 'video/mp4') return '.mp4';
  return '';
}

function extensionFromUrl(url) {
  try {
    const parsed = new URL(url);
    const withoutTemplateSuffix = parsed.pathname.replace(/~tplv-[^/]+$/i, '');
    const ext = path.posix.extname(withoutTemplateSuffix).toLowerCase();
    if (['.avif', '.gif', '.jpg', '.jpeg', '.mp4', '.png', '.webp'].includes(ext)) {
      return ext === '.jpeg' ? '.jpg' : ext;
    }
  } catch {
    // ignore and fall back to content type
  }
  return '';
}

async function fetchAsset(url) {
  const existing = existingTosImageUrl(url);
  if (existing) {
    return { status: 'existing-tos', publicUrl: existing };
  }

  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), REQUEST_TIMEOUT_MS);
  try {
    const response = await fetch(url, {
      signal: controller.signal,
      headers: {
        'user-agent': 'Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 Mobile/15E148',
        referer: 'https://www.douyin.com/',
      },
    });
    if (!response.ok) {
      throw new Error(`HTTP ${response.status}`);
    }
    const bytes = Buffer.from(await response.arrayBuffer());
    if (bytes.length < 32) {
      throw new Error('empty asset');
    }
    const ext = extensionFromContentType(response.headers.get('content-type') || '') || extensionFromUrl(url) || '.bin';
    const filename = `external-${sha1(url).slice(0, 16)}${ext}`;
    await mkdir(OUTPUT_DIR, { recursive: true });
    await writeFile(path.join(OUTPUT_DIR, filename), bytes);
    return { status: 'downloaded', filename, publicUrl: `${TOS_EXTERNAL_BASE}/${filename}`, bytes: bytes.length };
  } finally {
    clearTimeout(timeout);
  }
}

async function runPool(items, worker) {
  let cursor = 0;
  const workers = Array.from({ length: Math.min(DOWNLOAD_CONCURRENCY, items.length) }, async () => {
    while (cursor < items.length) {
      const item = items[cursor];
      cursor += 1;
      await worker(item);
    }
  });
  await Promise.all(workers);
}

function externalVideoReplacement(node) {
  const url = node?.video?.play_addr?.url_list?.find((item) => typeof item === 'string' && item.startsWith(`${TOS_ROOT}/videos/`));
  return typeof url === 'string' ? url : '';
}

function normalizeJsonVideoSources(node) {
  if (!node || typeof node !== 'object') return false;
  let changed = false;

  if (!Array.isArray(node) && typeof node.source_video_url === 'string' && /^https?:\/\/www\.douyin\.com\/aweme\/v1\/play\//i.test(node.source_video_url)) {
    const replacement = externalVideoReplacement(node);
    if (replacement) {
      node.source_video_url = replacement;
    } else {
      delete node.source_video_url;
    }
    changed = true;
  }

  for (const value of Array.isArray(node) ? node : Object.values(node)) {
    if (normalizeJsonVideoSources(value)) changed = true;
  }
  return changed;
}

function scrubJsonExternalUrls(node) {
  if (!node || typeof node !== 'object') return false;
  let changed = false;

  const entries = Array.isArray(node) ? node.entries() : Object.entries(node);
  for (const [key, value] of entries) {
    if (typeof value === 'string' && EXTERNAL_HOST_PATTERN.test(value) && !value.startsWith(TOS_ROOT)) {
      node[key] = '';
      changed = true;
      continue;
    }

    if (scrubJsonExternalUrls(value)) changed = true;
  }

  return changed;
}

async function main() {
  const files = (await Promise.all(SCAN_ROOTS.map(listFiles))).flat();
  const contents = new Map();
  const urls = new Set();

  for (const file of files) {
    const content = await readFile(file, 'utf8');
    contents.set(file, content);
    for (const match of content.matchAll(URL_PATTERN)) {
      const url = cleanUrl(match[0]);
      if (isKnownExternalAsset(url)) urls.add(url);
    }
  }

  const mapping = new Map();
  const results = [];
  const urlList = Array.from(urls).sort();

  await runPool(urlList, async (url) => {
    try {
      const result = await fetchAsset(url);
      mapping.set(url, result.publicUrl);
      results.push({ url, ...result });
      console.log(`${result.status}: ${url}`);
    } catch (error) {
      const publicUrl = fallbackUrl(url);
      mapping.set(url, publicUrl);
      results.push({ url, status: 'fallback', publicUrl, error: error instanceof Error ? error.message : String(error) });
      console.warn(`fallback: ${url}`);
    }
  });

  let rewrittenFiles = 0;
  for (const [file, original] of contents.entries()) {
    let next = original;
    for (const [url, publicUrl] of mapping.entries()) {
      next = next.split(url).join(publicUrl);
    }
    if (path.extname(file) === '.json') {
      try {
        const parsed = JSON.parse(next);
        const changedVideoSources = normalizeJsonVideoSources(parsed);
        const changedExternalUrls = scrubJsonExternalUrls(parsed);
        if (changedVideoSources || changedExternalUrls) {
          next = `${JSON.stringify(parsed, null, 2)}\n`;
        }
      } catch {
        // Keep non-JSON-ish files unchanged.
      }
    }
    if (next !== original) {
      await writeFile(file, next);
      rewrittenFiles += 1;
    }
  }

  await mkdir(path.dirname(MANIFEST_PATH), { recursive: true });
  await writeFile(MANIFEST_PATH, `${JSON.stringify({
    generatedAt: new Date().toISOString(),
    outputDir: path.relative(ROOT, OUTPUT_DIR),
    total: results.length,
    downloaded: results.filter((item) => item.status === 'downloaded').length,
    existingTos: results.filter((item) => item.status === 'existing-tos').length,
    fallback: results.filter((item) => item.status === 'fallback').length,
    results,
  }, null, 2)}\n`);

  console.log(`Mirrored ${results.length} urls; rewritten files: ${rewrittenFiles}`);
  console.log(`Downloaded files: ${path.relative(ROOT, OUTPUT_DIR)}`);
}

await main();
