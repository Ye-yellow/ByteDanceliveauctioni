import { readdir, readFile, stat } from 'node:fs/promises';
import path from 'node:path';

const ROOT = process.cwd();
const SCAN_ROOTS = ['src', 'public/data'];
const FORBIDDEN_PATTERNS = [
  /(?:https?:\/\/)?dy\.2study\.top/i,
  /(?:https?:\/\/)?[^"'\s<>()]*douyinpic\.com/i,
  /(?:https?:\/\/)?www\.douyin\.com\/aweme\/v1\/play\//i,
  /(?:https?:\/\/)?[^"'\s<>()]*douyinstatic\.com/i,
  /(?:https?:\/\/)?[^"'\s<>()]*iesdouyin\.com/i,
  /(?:https?:\/\/)?[^"'\s<>()]*bytednsdoc\.com/i,
  /(?:https?:\/\/)?[^"'\s<>()]*byteimg\.com/i,
  /(?:https?:\/\/)?[^"'\s<>()]*ecombd(?:img|static)\.com/i,
];

const SOURCE_EXTENSIONS = new Set(['.css', '.json', '.ts', '.tsx']);

async function listFiles(entry) {
  const absolute = path.join(ROOT, entry);
  const info = await stat(absolute).catch(() => null);
  if (!info) return [];
  if (info.isFile()) return SOURCE_EXTENSIONS.has(path.extname(absolute)) ? [absolute] : [];

  const children = await readdir(absolute, { withFileTypes: true });
  const files = await Promise.all(children.map((child) => listFiles(path.join(entry, child.name))));
  return files.flat();
}

const files = (await Promise.all(SCAN_ROOTS.map(listFiles))).flat();
const findings = [];

for (const file of files) {
  const content = await readFile(file, 'utf8');
  const lines = content.split('\n');
  lines.forEach((line, index) => {
    if (FORBIDDEN_PATTERNS.some((pattern) => pattern.test(line))) {
      findings.push({
        file: path.relative(ROOT, file),
        line: index + 1,
        text: line.trim().slice(0, 220),
      });
    }
  });
}

if (findings.length) {
  console.error(`External Douyin/material assets found (${findings.length}). Mirror them to TOS before building.`);
  for (const finding of findings.slice(0, 40)) {
    console.error(`${finding.file}:${finding.line} ${finding.text}`);
  }
  if (findings.length > 40) {
    console.error(`...and ${findings.length - 40} more`);
  }
  process.exit(1);
}

console.log('External asset check passed.');
