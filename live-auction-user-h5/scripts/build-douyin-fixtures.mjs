import { copyFile, mkdir, readFile, readdir, rm, writeFile } from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const scriptDir = path.dirname(fileURLToPath(import.meta.url));
const appRoot = path.resolve(scriptDir, '..');
const repoRoot = path.resolve(appRoot, '..');
const referenceRoot = path.join(repoRoot, 'tmp/douyin-reference/douyin-master/public');
const referenceVideoPath = path.join(referenceRoot, 'data/videos.json');
const referenceUserPath = path.join(referenceRoot, 'data/users.json');
const referenceUserVideoListDir = path.join(referenceRoot, 'data/user_video_list');
const referenceCommentDir = path.join(referenceRoot, 'data/comments');
const referenceImageDir = path.join(referenceRoot, 'images');
const publicDataDir = path.join(appRoot, 'public/data');
const publicCommentDir = path.join(publicDataDir, 'comments');
const publicUserVideoListDir = path.join(publicDataDir, 'user-video-list');
const publicImageDir = path.join(appRoot, 'public/douyin-assets/images');

const args = new Map(process.argv.slice(2).map((item) => {
  const [key, ...valueParts] = item.replace(/^--/, '').split('=');
  return [key, valueParts.join('=') || 'true'];
}));

const limit = Math.max(1, Number.parseInt(args.get('limit') || '96', 10));
const tosBase = String(args.get('tos-base') || '').replace(/\/+$/, '');
const assetBase = String(args.get('asset-base') || '').replace(/\/+$/, '');
const seed = String(args.get('seed') || 'live-auction-douyin-diverse');

const CATEGORY_RULES = [
  ['food', /美食|吃|饭|菜|餐|厨房|面|火锅|咖啡|甜品|蛋糕|小吃|料理|烧烤|早餐|午餐|晚餐|豌豆|牛肉|土豆|黄瓜|皮蛋|涮羊肉/],
  ['travel', /旅行|旅游|风景|城市|海|山|湖|街|公园|户外|天空|日出|日落|雪|秋|夏|春|冬|新疆|西藏|云南|北京|上海|广州|深圳|成都|重庆|杭州|巴黎|法国|日本|曼谷/],
  ['pet', /猫|狗|宠物|小狗|小猫|猫咪|狗狗|动物|熊猫|兔|鸟|猫步/],
  ['culture', /国风|汉服|文化|传统|非遗|书法|古风|东方|历史|博物馆|诗|音乐|唱|歌|乐器|钢琴|吉他|姜子牙|老祖宗/],
  ['life', /生活|日常|记录|vlog|上班|下班|家|朋友|孩子|妈妈|爸爸|校园|工作|装修|房间|面试|同事|朋友圈/],
  ['comedy', /搞笑|哈哈|笑|段子|整活|离谱|社死|尴尬|沙雕|表情包|梗|挑战|舔狗|傻子|屎/],
  ['sport', /运动|健身|篮球|足球|跑步|骑行|滑雪|滑板|游泳|拳击|羽毛球|乒乓|瑜伽|马甲线|瘦身|拉伸|训练|跳舞|舞蹈/],
  ['auto', /车|汽车|机车|电动|越野|四驱|驾驶|方向盘/],
  ['beauty', /美女|女神|小姐姐|姐姐|穿搭|妆|漂亮|可爱|颜值|裙|写真|氛围感|自拍|甜妹|泳池/],
];
const CATEGORY_ORDER = ['food', 'travel', 'pet', 'culture', 'life', 'comedy', 'sport', 'auto', 'other', 'beauty'];
const CATEGORY_AUTHORS = {
  food: ['食味记录', '记录家常与街头美味，认真吃每一口。'],
  travel: ['城市漫游', '在路上收集光影、街景和风。'],
  pet: ['萌宠日常', '把可爱的小瞬间都留下来。'],
  culture: ['国风拾光', '传统、音乐和日常美学记录。'],
  life: ['生活观察', '普通日子也值得被看见。'],
  comedy: ['快乐放映', '刷到这里，先笑一下。'],
  sport: ['训练日记', '运动、训练和一点点自律。'],
  auto: ['新车研究所', '看车、试驾、记录新鲜感。'],
  beauty: ['穿搭日记', '记录穿搭、状态和镜头里的自己。'],
  other: ['精选创作者', '随手记录有意思的内容。'],
};

function firstUrl(urlList) {
  return Array.isArray(urlList) ? urlList.find((url) => typeof url === 'string' && url) || '' : '';
}

function isVideoPlaybackUrl(url) {
  return typeof url === 'string' && !/\.mp3(?:$|\?)|\/ies-music\//i.test(url);
}

function localCoverUrl(fileName) {
  return fileName ? `/douyin-assets/images/${path.basename(fileName)}` : '';
}

function publicAssetUrl(fileName) {
  const basename = path.basename(fileName);
  return assetBase ? `${assetBase}/images/${basename}` : localCoverUrl(basename);
}

function tosVideoUrl(awemeId) {
  return tosBase ? `${tosBase}/videos/${awemeId}.mp4` : '';
}

function hashString(value) {
  let hash = 2166136261;
  for (let index = 0; index < value.length; index += 1) {
    hash ^= value.charCodeAt(index);
    hash = Math.imul(hash, 16777619);
  }
  return hash >>> 0;
}

function seededShuffle(items, bucketName) {
  const copy = items.slice();
  let state = hashString(`${seed}:${bucketName}`) || 1;
  for (let index = copy.length - 1; index > 0; index -= 1) {
    state = (Math.imul(state, 1664525) + 1013904223) >>> 0;
    const swapIndex = state % (index + 1);
    [copy[index], copy[swapIndex]] = [copy[swapIndex], copy[index]];
  }
  return copy;
}

function rawCommentId(raw, index) {
  return String(raw.comment_id || `${raw.aweme_id || 'comment'}-${raw.create_time || 0}-${index}`);
}

function classifyVideo(item) {
  const text = `${item.desc || ''} ${item.author?.nickname || ''} ${item.music?.title || ''}`;
  const matched = CATEGORY_RULES.find(([, pattern]) => pattern.test(text));
  return matched?.[0] || 'other';
}

function cloneJSON(value) {
  return value ? JSON.parse(JSON.stringify(value)) : value;
}

function authorIds(author = {}) {
  return [author.unique_id, author.short_id, author.uid].filter((id) => id !== undefined && id !== null && String(id));
}

function douyinIdForAuthor(author = {}) {
  return String(author.unique_id || author.short_id || author.uid || '').trim();
}

function normalizedSearchText(value) {
  return String(value || '').replace(/[^\p{L}\p{N}]+/gu, '').toLowerCase();
}

function authorMatchTokens(author = {}) {
  const tokens = new Set(authorIds(author).map(normalizedSearchText));
  const nickname = String(author.nickname || '');
  const compactNickname = normalizedSearchText(nickname);
  if (compactNickname) tokens.add(compactNickname);
  const chineseName = (nickname.match(/\p{Script=Han}+/gu) || []).join('');
  if (chineseName) {
    const compactChineseName = normalizedSearchText(chineseName);
    tokens.add(compactChineseName);
    if (compactChineseName.length >= 3) tokens.add(compactChineseName.slice(0, 3));
  }
  return [...tokens].filter((token) => token.length >= 3 && token !== '0');
}

function workVideoSearchText(work = {}) {
  const challengeText = Array.isArray(work.text_extra)
    ? work.text_extra.map((item) => item.hashtag_name || item.name || item.word).filter(Boolean).join(' ')
    : '';
  const searchWords = Array.isArray(work.suggest_words?.suggest_words)
    ? work.suggest_words.suggest_words.map((item) => item.word).filter(Boolean).join(' ')
    : '';
  return normalizedSearchText([
    work.author?.nickname,
    ...authorIds(work.author || {}),
    work.desc,
    work.music?.title,
    work.music?.author,
    work.music?.owner_nickname,
    work.share_info?.share_link_desc,
    challengeText,
    searchWords,
  ].filter(Boolean).join(' '));
}

function workVideoBelongsToAuthor(work, author) {
  const targetIds = new Set(authorIds(author).map((id) => String(id)));
  const workIds = authorIds(work.author || {}).map((id) => String(id));
  if (workIds.some((id) => targetIds.has(id))) return true;

  const workText = workVideoSearchText(work);
  return authorMatchTokens(author).some((token) => workText.includes(token));
}

function dedupeVideos(rows) {
  const seen = new Set();
  return rows.filter((row) => {
    const id = String(row.aweme_id || row.desc || '');
    if (!id || seen.has(id)) return false;
    seen.add(id);
    return true;
  });
}

function sourceImageExt(source) {
  try {
    const ext = path.extname(new URL(source).pathname);
    return ext && ext.length <= 8 ? ext : '.jpg';
  } catch {
    const ext = path.extname(source);
    return ext && ext.length <= 8 ? ext : '.jpg';
  }
}

function filenameForRemoteImage(source, prefix) {
  const safePrefix = prefix.replace(/[^a-z0-9_-]+/gi, '-').replace(/^-+|-+$/g, '').toLowerCase() || 'asset';
  return `${safePrefix}-${hashString(source).toString(36)}${sourceImageExt(source)}`;
}

async function downloadRemoteImage(source, prefix) {
  const fileName = filenameForRemoteImage(source, prefix);
  if (assetBase) return publicAssetUrl(fileName);
  const dest = path.join(publicImageDir, fileName);
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), 6000);
  try {
    const response = await fetch(source, {
      signal: controller.signal,
      headers: {
        'accept': 'image/avif,image/webp,image/apng,image/svg+xml,image/*,*/*;q=0.8',
        'user-agent': 'Mozilla/5.0 live-auction-h5-fixture-builder',
      },
    });
    if (!response.ok) return source;
    const buffer = Buffer.from(await response.arrayBuffer());
    await writeFile(dest, buffer);
    return publicAssetUrl(fileName);
  } catch {
    return source;
  } finally {
    clearTimeout(timeout);
  }
}

async function storeImageAsset(source, prefix) {
  if (!source || typeof source !== 'string') return '';
  if (/^https?:\/\//i.test(source)) return downloadRemoteImage(source, prefix);
  const basename = path.basename(source);
  if (!basename) return '';
  const copied = await copyIfPresent(path.join(referenceImageDir, basename), path.join(publicImageDir, basename));
  return copied ? publicAssetUrl(basename) : source;
}

async function storeImageList(urlList, prefix) {
  const mapped = [];
  for (const source of Array.isArray(urlList) ? urlList : []) {
    const url = await storeImageAsset(source, prefix);
    if (url) mapped.push(url);
  }
  return mapped;
}

function userVideoListFileNameForId(id, fileNames) {
  if (!id) return '';
  return fileNames.find((file) => file === `user-${id}.json`) || fileNames.find((file) => file === `user-${id}.md`) || '';
}

function userVideoListFileNameForAuthor(author, fileNames) {
  for (const id of authorIds(author)) {
    const file = userVideoListFileNameForId(String(id), fileNames);
    if (file) return file;
  }
  return '';
}

function selectDiverseVideos(rows, maxCount, requiredIds = new Set()) {
  const buckets = new Map(CATEGORY_ORDER.map((category) => [category, []]));
  for (const item of rows) {
    const category = classifyVideo(item);
    if (!buckets.has(category)) buckets.set(category, []);
    buckets.get(category).push(item);
  }
  const shuffledBuckets = new Map([...buckets.entries()].map(([category, items]) => [category, seededShuffle(items, category)]));
  const selected = [];
  const seen = new Set();
  while (selected.length < maxCount) {
    const before = selected.length;
    for (const category of CATEGORY_ORDER) {
      const next = shuffledBuckets.get(category)?.shift();
      if (!next) continue;
      const awemeId = String(next.aweme_id || '');
      if (!awemeId || seen.has(awemeId)) continue;
      selected.push({ ...next, fixture_category: category });
      seen.add(awemeId);
      if (selected.length >= maxCount) break;
    }
    if (selected.length === before) break;
  }
  const selectedIds = new Set(selected.map((item) => String(item.aweme_id || '')));
  const missingRequired = rows
    .filter((item) => requiredIds.has(String(item.aweme_id || '')) && !selectedIds.has(String(item.aweme_id || '')))
    .map((item) => ({ ...item, fixture_category: classifyVideo(item) }));
  for (const item of missingRequired) {
    const replaceIndex = [...selected].reverse().findIndex((candidate) => !requiredIds.has(String(candidate.aweme_id || '')));
    if (replaceIndex < 0) break;
    selected[selected.length - 1 - replaceIndex] = item;
  }
  return selected;
}

function authorForVideo(item, index, usersById, studyUsers) {
  const matchedAuthor = item.author?.nickname
    ? authorIds(item.author).map((id) => usersById.get(String(id))).find(Boolean)
    : null;
  const matchedStudyAuthor = studyUsers.find((user) => workVideoBelongsToAuthor(item, user));
  if (matchedAuthor || matchedStudyAuthor || item.author?.nickname) return cloneJSON(matchedAuthor || matchedStudyAuthor || item.author);
  const category = item.fixture_category || classifyVideo(item);
  const [nickname, signature] = CATEGORY_AUTHORS[category] || CATEGORY_AUTHORS.other;
  const awemeId = String(item.aweme_id || hashString(item.desc || nickname));
  return {
    avatar_168x168: { url_list: [] },
    avatar_300x300: { url_list: [] },
    aweme_count: 36 + (hashString(`${awemeId}:works`) % 280),
    card_entries: [],
    city: '',
    cover_url: [{ url_list: firstUrl(item.video?.cover?.url_list) ? [firstUrl(item.video?.cover?.url_list)] : [] }],
    follow_status: 0,
    following_count: 10 + (hashString(`${awemeId}:following`) % 480),
    gender: 0,
    ip_location: '',
    mplatform_followers_count: 2000 + (hashString(`${awemeId}:fans`) % 880000),
    nickname,
    province: '',
    signature,
    total_favorited: Number(item.statistics?.digg_count || 0) + 5000 + (hashString(`${awemeId}:likes`) % 2000000),
    uid: `fixture_${category}`,
    unique_id: `dy_${category}_${String(hashString(category)).slice(0, 6)}`,
    user_age: -1,
    white_cover_url: [],
  };
}

async function normalizeAuthorForVideo(author, item) {
  const normalized = cloneJSON(author || {});
  const awemeId = String(item.aweme_id || hashString(item.desc || normalized.nickname || 'author'));
  const category = item.fixture_category || classifyVideo(item);
  const [fallbackName, fallbackSignature] = CATEGORY_AUTHORS[category] || CATEGORY_AUTHORS.other;
  normalized.nickname ||= fallbackName;
  normalized.signature ||= fallbackSignature;
  normalized.aweme_count ||= 36 + (hashString(`${awemeId}:works`) % 280);
  normalized.following_count ||= 10 + (hashString(`${awemeId}:following`) % 480);
  normalized.mplatform_followers_count ||= 2000 + (hashString(`${awemeId}:fans`) % 880000);
  normalized.total_favorited ||= Number(item.statistics?.digg_count || 0) + 5000 + (hashString(`${awemeId}:likes`) % 2000000);
  normalized.uid ||= `fixture_${category}_${hashString(awemeId)}`;
  normalized.unique_id ||= normalized.short_id || `dy_${category}_${String(hashString(`${category}:${awemeId}`)).slice(0, 6)}`;
  normalized.user_age ??= -1;
  normalized.gender ??= 0;
  normalized.card_entries ||= [];
  normalized.white_cover_url ||= [];

  const fallbackCover = firstUrl(item.video?.cover?.url_list);
  if (!Array.isArray(normalized.cover_url) || !normalized.cover_url.length) {
    normalized.cover_url = [{ url_list: fallbackCover ? [fallbackCover] : [] }];
  }
  normalized.avatar_168x168 = {
    ...(normalized.avatar_168x168 || {}),
    url_list: await storeImageList(normalized.avatar_168x168?.url_list || normalized.avatar_thumb?.url_list, `avatar-${douyinIdForAuthor(normalized) || awemeId}`),
  };
  normalized.avatar_300x300 = {
    ...(normalized.avatar_300x300 || {}),
    url_list: await storeImageList(normalized.avatar_300x300?.url_list || normalized.avatar_168x168?.url_list, `avatar-large-${douyinIdForAuthor(normalized) || awemeId}`),
  };
  normalized.avatar_thumb = {
    ...(normalized.avatar_thumb || {}),
    url_list: await storeImageList(normalized.avatar_thumb?.url_list || normalized.avatar_168x168?.url_list, `avatar-thumb-${douyinIdForAuthor(normalized) || awemeId}`),
  };
  normalized.cover_url = await Promise.all(normalized.cover_url.map(async (cover, index) => ({
    ...cover,
    url_list: await storeImageList(cover?.url_list?.length ? cover.url_list : fallbackCover ? [fallbackCover] : [], `author-cover-${douyinIdForAuthor(normalized) || awemeId}-${index}`),
  })));
  return normalized;
}

async function loadUserVideoRows(author, userVideoListFileNames) {
  const file = userVideoListFileNameForAuthor(author, userVideoListFileNames);
  if (!file) return [];
  try {
    return JSON.parse(await readFile(path.join(referenceUserVideoListDir, file), 'utf8'));
  } catch {
    return [];
  }
}

async function normalizeWorkVideo(work, author, index) {
  const awemeId = String(work.aweme_id || `${douyinIdForAuthor(author)}-${index}`);
  const sourceVideoUrl = firstUrl((work.video?.play_addr?.url_list || []).filter(isVideoPlaybackUrl));
  const coverFile = firstUrl(work.video?.cover?.url_list);
  const coverUrl = coverFile ? await storeImageAsset(coverFile, `work-cover-${douyinIdForAuthor(author) || awemeId}-${index}`) : '';
  return {
    ...work,
    aweme_id: awemeId,
    author,
    source_video_url: sourceVideoUrl,
    video: {
      ...(work.video || {}),
      play_addr: { ...(work.video?.play_addr || {}), url_list: tosVideoUrl(awemeId) || sourceVideoUrl ? [tosVideoUrl(awemeId) || sourceVideoUrl] : [] },
      cover: { ...(work.video?.cover || {}), url_list: coverUrl ? [coverUrl] : [] },
    },
  };
}

async function copyIfPresent(src, dest) {
  try {
    await copyFile(src, dest);
    return true;
  } catch {
    return false;
  }
}

function buildSeededCommentsForVideo(awemeId, sourceRows, maxCount = 40) {
  const shuffled = seededShuffle(sourceRows, `comments:${awemeId}`);
  return shuffled.slice(0, maxCount).map((row, index) => ({
    ...row,
    comment_id: `${awemeId}${String(index).padStart(3, '0')}${rawCommentId(row, index).slice(-4)}`,
    aweme_id: awemeId,
    user_digged: 0,
    is_author_digged: false,
    sub_comment_count: String(Math.max(0, Number.parseInt(String(row.sub_comment_count || '0'), 10) || 0)),
  }));
}

const videos = JSON.parse(await readFile(referenceVideoPath, 'utf8'));
const users = JSON.parse(await readFile(referenceUserPath, 'utf8'));
const userVideoListFileNames = (await readdir(referenceUserVideoListDir)).filter((file) => /^user-.+\.(json|md)$/.test(file));
const usersById = new Map();
for (const user of users) {
  for (const id of authorIds(user)) usersById.set(String(id), user);
}
const studyUsers = users.filter((user) => userVideoListFileNameForAuthor(user, userVideoListFileNames));
const commentFiles = (await readdir(referenceCommentDir)).filter((file) => /^video_id_\d+\.json$/.test(file));
const commentVideoIds = new Set(commentFiles.map((file) => file.replace(/^video_id_/, '').replace(/\.json$/, '')));
const playableVideos = videos.filter((item) => firstUrl((item.video?.play_addr?.url_list || []).filter(isVideoPlaybackUrl)));
const selectedVideos = selectDiverseVideos(playableVideos, limit, commentVideoIds);

await rm(publicCommentDir, { recursive: true, force: true });
await rm(publicUserVideoListDir, { recursive: true, force: true });
await rm(publicImageDir, { recursive: true, force: true });
await mkdir(publicDataDir, { recursive: true });
await mkdir(publicCommentDir, { recursive: true });
await mkdir(publicUserVideoListDir, { recursive: true });
await mkdir(publicImageDir, { recursive: true });

const commentRowsByVideoId = new Map();
const allCommentRows = [];
for (const file of commentFiles) {
  const videoId = file.replace(/^video_id_/, '').replace(/\.json$/, '');
  const rows = JSON.parse(await readFile(path.join(referenceCommentDir, file), 'utf8'));
  commentRowsByVideoId.set(videoId, rows);
  allCommentRows.push(...rows);
}

const manifest = [];
const report = [];
const usedAuthorsById = new Map();

for (const [index, item] of selectedVideos.entries()) {
  const awemeId = String(item.aweme_id || '');
  if (!awemeId) continue;
  const sourceVideoUrl = firstUrl((item.video?.play_addr?.url_list || []).filter(isVideoPlaybackUrl));
  const coverFile = firstUrl(item.video?.cover?.url_list);
  const coverUrl = coverFile ? await storeImageAsset(coverFile, `feed-cover-${awemeId}`) : '';
  const videoUrl = tosVideoUrl(awemeId) || sourceVideoUrl;
  const author = await normalizeAuthorForVideo(authorForVideo(item, index, usersById, studyUsers), item);
  const authorId = douyinIdForAuthor(author);
  if (authorId) usedAuthorsById.set(authorId, author);
  manifest.push({
    ...item,
    author,
    source_video_url: sourceVideoUrl,
    fixture_category: item.fixture_category || classifyVideo(item),
    has_local_comments: commentVideoIds.has(awemeId),
    video: {
      ...item.video,
      play_addr: { ...(item.video?.play_addr || {}), url_list: videoUrl ? [videoUrl] : [] },
      cover: { ...(item.video?.cover || {}), url_list: coverUrl ? [coverUrl] : [] },
    },
  });
  report.push({
    aweme_id: awemeId,
    author_id: authorId,
    author: author.nickname,
    desc: item.desc || '',
    category: item.fixture_category || classifyVideo(item),
    has_local_comments: commentVideoIds.has(awemeId),
    source_video_url: sourceVideoUrl,
    target_video_url: videoUrl,
    cover: coverUrl,
    avatar: firstUrl(author.avatar_168x168?.url_list),
  });
}

for (const item of manifest) {
  const awemeId = String(item.aweme_id || '');
  if (!awemeId) continue;
  const exactRows = commentRowsByVideoId.get(awemeId);
  const rows = exactRows?.length ? exactRows : buildSeededCommentsForVideo(awemeId, allCommentRows);
  await writeFile(path.join(publicCommentDir, `video_id_${awemeId}.json`), `${JSON.stringify(rows, null, 2)}\n`);
}

for (const [authorId, author] of usedAuthorsById.entries()) {
  const rows = await loadUserVideoRows(author, userVideoListFileNames);
  const ownManifestRows = manifest.filter((item) => douyinIdForAuthor(item.author) === authorId);
  const matchedRows = rows.filter((row) => workVideoBelongsToAuthor(row, author));
  const sourceRows = dedupeVideos([...(matchedRows.length ? matchedRows : []), ...ownManifestRows]);
  const normalizedRows = [];
  for (const [index, row] of sourceRows.entries()) {
    normalizedRows.push(await normalizeWorkVideo(row, author, index));
  }
  await writeFile(path.join(publicUserVideoListDir, `user-${authorId}.json`), `${JSON.stringify(normalizedRows, null, 2)}\n`);
}

await writeFile(path.join(publicDataDir, 'douyin-feed.json'), `${JSON.stringify(manifest, null, 2)}\n`);
await writeFile(path.join(publicDataDir, 'douyin-feed.import-report.json'), `${JSON.stringify(report, null, 2)}\n`);

console.log(`douyin fixtures built: ${manifest.length} videos, ${commentFiles.length} source comment files, ${usedAuthorsById.size} author work lists`);
console.log(JSON.stringify(manifest.reduce((acc, item) => {
  acc[item.fixture_category] = (acc[item.fixture_category] || 0) + 1;
  return acc;
}, {}), null, 2));
if (!tosBase) {
  console.log('video urls still point to source play urls; pass --tos-base=<public-prefix> after uploading mp4 files to TOS.');
}
if (!assetBase) {
  console.log('image urls still point to local /douyin-assets; pass --asset-base=<public-prefix> after uploading images to TOS.');
}
