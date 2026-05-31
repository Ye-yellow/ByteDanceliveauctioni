import { useEffect, useMemo, useRef, useState, type CSSProperties, type PointerEvent, type ReactNode, type TouchEvent } from 'react';
import { listPublicRooms } from '../features/auction/api/auctionApi';
import { resolveLivePlaylist } from '../features/live/hooks/useLivePlayer';
import type { Room } from '../shared/api/types';
import { useAuthSession } from '../shared/auth/useAuthSession';
import { navigateTo, SPA_NAVIGATE_EVENT } from '../shared/navigation';
import { DouyinCommentSheet } from '../shared/ui/DouyinCommentSheet';
import { DouyinTabBar, type DouyinTab } from '../shared/ui/DouyinTabBar';
import { DouyinLoading } from '../shared/ui/DouyinLoading';

type FeedKind = 'video' | 'live';
type HomeSheet = 'comment' | 'share' | 'more' | 'friend' | null;
type CommentTarget = { videoId: string; comments: string };

type FeedItem = {
  id: string;
  awemeId?: string;
  kind: FeedKind;
  author: string;
  avatarUrl?: string;
  title: string;
  music: string;
  location?: string;
  likes: string;
  comments: string;
  collects: string;
  shares: string;
  tone: 'rose' | 'cyan' | 'amber' | 'violet' | 'dark';
  videoUrl?: string;
  videoUrls?: string[];
  coverUrl?: string;
  sourceLabel?: string;
  liveLabel?: string;
  liveHref?: string;
  publisherHref?: string;
  mediaWidth?: number;
  mediaHeight?: number;
};

type DouyinVideoRecord = {
  aweme_id?: string;
  desc?: string;
  author?: {
    nickname?: string;
    avatar_300x300?: { url_list?: string[] };
    avatar_168x168?: { url_list?: string[] };
    avatar_thumb?: { url_list?: string[] };
    cover_url?: Array<{ url_list?: string[] }>;
  };
  music?: {
    title?: string;
    author?: string;
  };
  video?: {
    play_addr?: { url_list?: string[] };
    cover?: { url_list?: string[] };
    width?: number;
    height?: number;
  };
  statistics?: {
    digg_count?: number;
    comment_count?: number;
    collect_count?: number;
    share_count?: number;
  };
};

const LIVE_CHANNEL_INDEX = 3;
const CHANNELS = ['热点', '长视频', '关注', '直播', '推荐'];
const DOUYIN_VIDEO_SOURCE = '/data/douyin-feed.json';
const DOUYIN_IMAGE_BASE = 'https://liveauction.tos-cn-beijing.volces.com/douyin-h5/images/';
const DOUYIN_COMMENT_VIDEO_IDS = [
  '6686589698707590411',
  '6826943630775831812',
  '6882368275695586568',
  '6923214072347512068',
  '7000587983069957383',
  '7005490661592026405',
  '7110263965858549003',
  '7128686458763889956',
  '7161000281575148800',
  '7194815099381484860',
  '7260749400622894336',
  '7267478481213181238',
  '7270431418822446370',
  '7293100687989148943',
  '7295697246132227343',
  '7321200290739326262',
];
const SIDE_RECENTS = ['青柠汽水', '山海收藏家', '奶油小熊', '林间晚风', 'LiveAuction', 'Yexieer'];
const SIDE_FEATURES = ['我的钱包', '券包', '小程序', '观看历史', '内容偏好', '离线模式', '设置', '稍后再看'];
const SHARE_OPTIONS = ['转发', '私信朋友', '微信', '朋友圈', 'QQ', '复制链接', '举报', '不感兴趣'];

const FEED_ITEMS: FeedItem[] = [
  {
    id: 'feed-hot-1',
    awemeId: '7260749400622894336',
    kind: 'video',
    author: '我是香秀',
    title: '你说爱像云 要自在漂浮才美丽',
    music: '热门原声 · 我是香秀',
    location: '推荐',
    likes: '12.8w',
    comments: '2.2w',
    collects: '2.4w',
    shares: '1.2w',
    tone: 'rose',
    coverUrl: `${DOUYIN_IMAGE_BASE}jwWCPZVTIA4IKM-8WipLF.png`,
  },
  {
    id: 'feed-live-1',
    kind: 'live',
    author: '严选直播间',
    title: '今晚 8 点严选好物开箱，先看细节再进直播间',
    music: '直播中 · 严选专场',
    location: '深圳',
    likes: '8.6w',
    comments: '1.1w',
    collects: '9.8k',
    shares: '3.2k',
    tone: 'cyan',
    liveLabel: 'LIVE · 999w人气',
    liveHref: '/home/live',
  },
  {
    id: 'feed-hot-2',
    awemeId: '6686589698707590411',
    kind: 'video',
    author: '周子然JingYi',
    title: '门有点矮哟～',
    music: '热门原声 · 周子然JingYi',
    location: '推荐',
    likes: '9.7w',
    comments: '1.9w',
    collects: '1.8w',
    shares: '888',
    tone: 'amber',
    coverUrl: `${DOUYIN_IMAGE_BASE}_T0vQPZKXrNC6ulECmMes.png`,
  },
  {
    id: 'feed-hot-3',
    awemeId: '6826943630775831812',
    kind: 'video',
    author: '拍场观察员',
    title: '今天的严选好物先看一眼，近景、材质、成交节奏都刷到',
    music: 'LiveAuction 原声 · 备拍现场',
    likes: '9.2w',
    comments: '463',
    collects: '1.1w',
    shares: '1392',
    tone: 'violet',
  },
  {
    id: 'feed-live-2',
    kind: 'live',
    author: '广州表哥',
    title: '同城直播开场，边聊边看今日热门好物',
    music: '直播中 · 同城好物',
    location: '广州',
    likes: '6.1w',
    comments: '824',
    collects: '6400',
    shares: '2.1k',
    tone: 'dark',
    liveLabel: 'LIVE · 红包',
    liveHref: '/home/live',
  },
];

const VIDEO_FEED_ITEMS = FEED_ITEMS.filter((item) => item.kind === 'video');
const HORIZONTAL_SWIPE_DISTANCE = 44;
const VERTICAL_SWIPE_DISTANCE = 44;
const QUICK_SWIPE_MS = 280;
const CHANNEL_VIDEO_WINDOW = 32;
const HOME_RETURN_STATE_KEY = 'douyin-home-return-state';
const HOME_RETURN_STATE_TTL_MS = 5 * 60 * 1000;
const FEED_TONES: FeedItem['tone'][] = ['rose', 'cyan', 'amber', 'violet', 'dark'];
const DY_HEART_PATH = 'M453.036 88.712C493.774 30.664 560.66 0 634.251 0C774.403 0 890.972 121.59 890.972 266.137C890.972 266.155 890.972 266.173 890.972 266.191C890.981 266.171 890.991 266.151 891 266.131C891 270.252 890.93 273.372 890.878 275.668C890.806 278.819 890.77 280.416 891 280.916C890.469 311.319 885.222 336.438 875.899 369.628C870.609 375.588 865.694 386.812 860.798 399.199C853.074 411.197 850.151 417.009 845.697 428.77C841.152 436.399 836.324 444.063 831.245 451.744C793.796 508.544 743.708 565.074 693.667 615.252C615.336 694.277 535.544 760.058 500.832 788.675C491.247 796.576 485.1 801.644 483.368 803.375C471.067 815.678 458.765 815.986 446.463 815.994C446.139 815.998 445.814 816 445.486 816C420.25 816 407.632 803.381 395.014 790.763C394.051 789.8 391.601 787.779 387.858 784.783C349.625 756.6 263.586 687.786 182.742 604.783C121.066 542.02 61.622 470.007 29.092 399.588C16.474 374.351 0.731 314.264 0 280.922C0.269 280.655 0.227 279.049 0.144 275.844C0.083 273.498 0 270.297 0 266.137C0 121.524 116.502 0 256.721 0C330.179 0 397.131 30.664 453.036 88.712z';

type HomeReturnState = {
  baseIndex: number;
  channelIndex: number;
  itemIndexes: number[];
  at: number;
};

function clampIndex(value: number, length: number) {
  return Math.max(0, Math.min(length - 1, value));
}

function wrapIndex(value: number, length: number) {
  if (length <= 0) return 0;
  return ((value % length) + length) % length;
}

function isMeEntryPath() {
  return location.pathname === '/me' || location.pathname.startsWith('/m/profile') || new URLSearchParams(location.search).get('tab') === 'me';
}

function initialBaseIndex() {
  return isMeEntryPath() ? 2 : 1;
}

function readHomeReturnState(): HomeReturnState | null {
  try {
    const raw = sessionStorage.getItem(HOME_RETURN_STATE_KEY);
    if (!raw) return null;
    sessionStorage.removeItem(HOME_RETURN_STATE_KEY);
    const parsed = JSON.parse(raw) as Partial<HomeReturnState>;
    if (!parsed || Date.now() - Number(parsed.at || 0) > HOME_RETURN_STATE_TTL_MS) return null;
    if (!Array.isArray(parsed.itemIndexes)) return null;
    return {
      baseIndex: Number(parsed.baseIndex),
      channelIndex: Number(parsed.channelIndex),
      itemIndexes: parsed.itemIndexes.map((value) => Number(value) || 0),
      at: Number(parsed.at),
    };
  } catch {
    sessionStorage.removeItem(HOME_RETURN_STATE_KEY);
    return null;
  }
}

function writeHomeReturnState(state: Omit<HomeReturnState, 'at'>) {
  sessionStorage.setItem(HOME_RETURN_STATE_KEY, JSON.stringify({ ...state, at: Date.now() }));
}

function shuffleItems<T>(items: T[]) {
  const shuffled = items.slice();
  for (let index = shuffled.length - 1; index > 0; index -= 1) {
    const randomIndex = Math.floor(Math.random() * (index + 1));
    [shuffled[index], shuffled[randomIndex]] = [shuffled[randomIndex], shuffled[index]];
  }
  return shuffled;
}

function swipeDistance(distance: number, elapsed: number, baseDistance: number) {
  return Math.abs(distance) > (elapsed < QUICK_SWIPE_MS ? baseDistance * 0.72 : baseDistance);
}

function normalizeRemoteAsset(url?: string) {
  if (!url) return '';
  if (/^https?:\/\//i.test(url)) return url;
  if (url.startsWith('/douyin-assets/') || url.startsWith('/data/')) return url;
  if (url.startsWith('/images/')) return `${DOUYIN_IMAGE_BASE}${url.slice('/images/'.length)}`;
  if (url.startsWith('/')) return url;
  return `${DOUYIN_IMAGE_BASE}${url}`;
}

function normalizeRemoteVideoUrl(url?: string) {
  const assetUrl = normalizeRemoteAsset(url);
  if (!assetUrl) return '';
  if (import.meta.env.DEV && /^https:\/\/www\.douyin\.com\/aweme\/v1\/play\//.test(assetUrl)) {
    return `/__douyin_video_proxy?url=${encodeURIComponent(assetUrl)}`;
  }
  return assetUrl;
}

function firstUrl(urlList?: string[]) {
  return Array.isArray(urlList) ? urlList.find(Boolean) || '' : '';
}

function authorAvatarUrl(video: DouyinVideoRecord) {
  return normalizeRemoteAsset(
    firstUrl(video.author?.avatar_300x300?.url_list)
      || firstUrl(video.author?.avatar_168x168?.url_list)
      || firstUrl(video.author?.avatar_thumb?.url_list)
      || firstUrl(video.author?.cover_url?.[0]?.url_list)
      || firstUrl(video.video?.cover?.url_list),
  );
}

function isVideoPlaybackUrl(url: string) {
  return !/\.mp3(?:$|\?)|\/ies-music\//i.test(url);
}

function formatRemoteCount(value?: number, fallback = '0') {
  if (typeof value !== 'number' || !Number.isFinite(value)) return fallback;
  if (value >= 100000000) return `${(value / 100000000).toFixed(value >= 1000000000 ? 0 : 1)}亿`;
  if (value >= 10000) return `${(value / 10000).toFixed(value >= 100000 ? 0 : 1)}w`;
  return value.toLocaleString('zh-CN');
}

function publisherHref(item: FeedItem) {
  return item.publisherHref || `/user?awemeId=${encodeURIComponent(item.awemeId || item.id)}`;
}

function chooseVideoFitFromSize(
  mediaWidth?: number,
  mediaHeight?: number,
  frameWidth = window.innerWidth,
  frameHeight = window.innerHeight - 58,
): 'cover' | 'contain' {
  if (!mediaWidth || !mediaHeight || !frameWidth || !frameHeight) return 'cover';
  const frameAspect = frameWidth / Math.max(1, frameHeight);
  const videoAspect = mediaWidth / mediaHeight;
  const coverCropRatio = videoAspect > frameAspect
    ? 1 - (frameAspect / videoAspect)
    : 1 - (videoAspect / frameAspect);
  return coverCropRatio <= 0.12 ? 'cover' : 'contain';
}

function remoteVideoToFeedItem(video: DouyinVideoRecord, index: number): FeedItem | null {
  const videoUrls = (video.video?.play_addr?.url_list || [])
    .filter(isVideoPlaybackUrl)
    .map(normalizeRemoteVideoUrl)
    .filter(Boolean);
  if (!videoUrls.length) return null;
  const videoUrl = videoUrls[0];
  const coverUrl = normalizeRemoteAsset(video.video?.cover?.url_list?.[0]);
  const avatarUrl = authorAvatarUrl(video);
  const musicTitle = video.music?.title || '原创音乐';
  const musicAuthor = video.music?.author ? ` · ${video.music.author}` : '';
  return {
    id: `dy-${video.aweme_id || index}`,
    awemeId: video.aweme_id,
    kind: 'video',
    author: video.author?.nickname || `抖音用户${index + 1}`,
    avatarUrl,
    title: video.desc || '刷到一条真实抖音素材',
    music: `${musicTitle}${musicAuthor}`,
    location: index % 3 === 0 ? '推荐' : undefined,
    likes: formatRemoteCount(video.statistics?.digg_count, '8.8w'),
    comments: formatRemoteCount(video.statistics?.comment_count, '999'),
    collects: formatRemoteCount(video.statistics?.collect_count, '1.2w'),
    shares: formatRemoteCount(video.statistics?.share_count, '2.1k'),
    tone: FEED_TONES[index % FEED_TONES.length],
    videoUrl,
    videoUrls,
    coverUrl,
    publisherHref: `/user?awemeId=${encodeURIComponent(video.aweme_id || String(index))}`,
    sourceLabel: '抖音素材',
    mediaWidth: video.video?.width,
    mediaHeight: video.video?.height,
  };
}

async function fetchDouyinFeedItemsFrom(source: string) {
  const response = await fetch(source, source.startsWith('/') ? undefined : { mode: 'cors' });
  if (!response.ok) throw new Error(`load douyin feed failed: ${response.status}`);
  const rows = await response.json() as DouyinVideoRecord[];
  const items = rows
    .map(remoteVideoToFeedItem)
    .filter((item): item is FeedItem => Boolean(item));
  const commentVideoIdSet = new Set(DOUYIN_COMMENT_VIDEO_IDS);
  return [
    ...items.filter((item) => item.awemeId && commentVideoIdSet.has(item.awemeId)),
    ...items.filter((item) => !item.awemeId || !commentVideoIdSet.has(item.awemeId)),
  ];
}

async function fetchDouyinFeedItems() {
  return shuffleItems(await fetchDouyinFeedItemsFrom(DOUYIN_VIDEO_SOURCE));
}

function takeLoop<T>(items: T[], count: number, start = 0) {
  if (!items.length) return [];
  const size = Math.min(count, items.length);
  return Array.from({ length: size }, (_, index) => items[(start + index) % items.length]);
}

function liveRoomFeedItems(rooms: Room[]): FeedItem[] {
  if (!rooms.length) return [];
  const tones: FeedItem['tone'][] = ['cyan', 'dark', 'violet', 'rose'];
  return rooms.map((room, index) => {
    const name = room.name || `直播间${index + 1}`;
    return {
      id: `room-${room.id}`,
      kind: 'live',
      author: name,
      title: `${name} 正在直播，进入直播间看今日精选好物`,
      music: `直播中 · ${name}`,
      location: room.platform || 'LiveAuction',
      likes: `${index + 8}.${(index + 3) % 10}w`,
      comments: `${index + 1}.${(index + 2) % 10}w`,
      collects: `${index + 6}.${index}k`,
      shares: `${index + 2}.${index + 1}k`,
      tone: tones[index % tones.length] || 'cyan',
      liveLabel: `LIVE · ${name}`,
      liveHref: `/m/room/${encodeURIComponent(room.id)}`,
    };
  });
}

function mixFeedItems(liveItems: FeedItem[], videoItems: FeedItem[]) {
  const videos = takeLoop(videoItems, CHANNEL_VIDEO_WINDOW);
  const lives = liveItems;
  return [
    videos[0],
    lives[0],
    videos[1],
    videos[2],
    lives[1],
    ...videos.slice(3, CHANNEL_VIDEO_WINDOW),
  ].filter((item): item is FeedItem => Boolean(item));
}

function channelItems(channelIndex: number, liveItems: FeedItem[], videoItems: FeedItem[]) {
  const videos = videoItems.length ? videoItems : VIDEO_FEED_ITEMS;
  const lives = liveItems;
  const mixed = mixFeedItems(lives, videos);
  if (channelIndex === 0) return mixed.slice(2).concat(mixed.slice(0, 2));
  if (channelIndex === 1) return takeLoop(videos, CHANNEL_VIDEO_WINDOW, 5).concat(lives.slice(0, 1));
  if (channelIndex === 2) return mixed.slice().reverse();
  if (channelIndex === LIVE_CHANNEL_INDEX) return lives;
  return mixed;
}

function Avatar({ name, src }: { name: string; src?: string }) {
  return (
    <span className="dyHomeReplicaAvatar">
      {src ? <img src={src} alt="" loading="lazy" referrerPolicy="no-referrer" /> : name.slice(0, 1)}
    </span>
  );
}

function HomeIcon({ name }: { name: 'menu' | 'search' | 'heart' | 'comment' | 'star' | 'share' | 'music' }) {
  if (name === 'menu') {
    return (
      <svg viewBox="0 0 24 24" aria-hidden="true">
        <path d="M4 7h16M4 12h16M4 17h16" />
      </svg>
    );
  }
  if (name === 'search') {
    return (
      <svg viewBox="0 0 512 512" aria-hidden="true">
        <path d="M456.69 421.39 362.6 327.3a173.8 173.8 0 0 0 34.84-104.58C397.44 126.38 319.06 48 222.72 48S48 126.38 48 222.72s78.38 174.72 174.72 174.72A173.8 173.8 0 0 0 327.3 362.6l94.09 94.09a25 25 0 0 0 35.3-35.3M97.92 222.72a124.8 124.8 0 1 1 124.8 124.8 124.95 124.95 0 0 1-124.8-124.8" />
      </svg>
    );
  }
  if (name === 'heart') {
    return (
      <svg viewBox="0 0 891 816" aria-hidden="true">
        <path d={DY_HEART_PATH} />
      </svg>
    );
  }
  if (name === 'comment') {
    return (
      <svg viewBox="0 0 24 24" aria-hidden="true">
        <path d="M21.25 8.18a9.8 9.8 0 0 0-2.16-3.25 10 10 0 0 0-14.15 0 9.8 9.8 0 0 0-2.17 3.25A10 10 0 0 0 2.01 12a9.7 9.7 0 0 0 .74 3.77l-.5 3.65a1.95 1.95 0 0 0 1.29 2.26c.297.098.613.122.92.07l3.65-.54a9.8 9.8 0 0 0 3.88.79 10 10 0 0 0 9.24-13.82zM7.73 13.61a1.61 1.61 0 1 1 .001-3.22 1.61 1.61 0 0 1 0 3.22m4.28 0a1.61 1.61 0 1 1 .001-3.22 1.61 1.61 0 0 1 0 3.22m4.28 0a1.61 1.61 0 1 1 .001-3.22 1.61 1.61 0 0 1 0 3.22" />
      </svg>
    );
  }
  if (name === 'star') {
    return (
      <svg viewBox="0 0 24 24" aria-hidden="true">
        <path d="m12 17.27 4.15 2.51c.76.46 1.69-.22 1.49-1.08l-1.1-4.72 3.67-3.18c.67-.58.31-1.68-.57-1.75l-4.83-.41-1.89-4.46c-.34-.81-1.5-.81-1.84 0L9.19 8.63l-4.83.41c-.88.07-1.24 1.17-.57 1.75l3.67 3.18-1.1 4.72c-.2.86.73 1.54 1.49 1.08z" />
      </svg>
    );
  }
  if (name === 'share') {
    return <img className="dyHomeReplicaShareImage" src="/douyin-assets/icons/share-white-full.png" alt="" />;
  }
  if (name === 'music') {
    return (
      <svg viewBox="0 0 24 24" aria-hidden="true">
        <path d="M9 18.2a3 3 0 1 1-2.4-2.94V6.1l10.7-2.3v10.9a3 3 0 1 1-1.9-2.8V7.2L8.5 8.68v7.6c.32.5.5 1.1.5 1.92Z" />
      </svg>
    );
  }
  return null;
}

function LikeBurstIcon() {
  return (
    <span className="dyHomeReplicaLikeBurst" aria-hidden="true">
      <i className="shockRing" />
      <i className="heartAura"><svg viewBox="0 0 891 816"><path d={DY_HEART_PATH} /></svg></i>
      <i className="heartCore"><svg viewBox="0 0 891 816"><path d={DY_HEART_PATH} /></svg></i>
      <i className="heartEcho echo0"><svg viewBox="0 0 891 816"><path d={DY_HEART_PATH} /></svg></i>
      <i className="heartEcho echo1"><svg viewBox="0 0 891 816"><path d={DY_HEART_PATH} /></svg></i>
      <i className="heartEcho echo2"><svg viewBox="0 0 891 816"><path d={DY_HEART_PATH} /></svg></i>
      <i className="heartEcho echo3"><svg viewBox="0 0 891 816"><path d={DY_HEART_PATH} /></svg></i>
      <i className="heartEcho echo4"><svg viewBox="0 0 891 816"><path d={DY_HEART_PATH} /></svg></i>
      <i className="heartEcho echo5"><svg viewBox="0 0 891 816"><path d={DY_HEART_PATH} /></svg></i>
      <i className="sparkDot spark0" />
      <i className="sparkDot spark1" />
      <i className="sparkDot spark2" />
      <i className="sparkDot spark3" />
      <i className="sparkDot spark4" />
      <i className="sparkDot spark5" />
    </span>
  );
}

function chooseVideoFit(video: HTMLVideoElement) {
  const slideRect = video.closest('.dyHomeReplicaFeedSlide')?.getBoundingClientRect();
  if (!slideRect || !video.videoWidth || !video.videoHeight) return 'contain';
  return chooseVideoFitFromSize(video.videoWidth, video.videoHeight, slideRect.width, Math.max(1, slideRect.height - 58));
}

function SideCard({ title, action, children }: { title: string; action?: string; children: ReactNode }) {
  return (
    <section className="dyHomeReplicaSideCard">
      <header>
        <b>{title}</b>
        {action ? <button type="button">{action} ›</button> : <span />}
      </header>
      {children}
    </section>
  );
}

function Sidebar({ onClose }: { onClose: () => void }) {
  return (
    <aside className="dyHomeReplicaSidebar">
      <header className="dyHomeReplicaSideHeader">
        <b>下午好</b>
        <a href="/scan"><span>▣</span>扫一扫</a>
      </header>
      <SideCard title="常用小程序" action="全部">
        <div className="dyHomeReplicaMiniGrid">
          {['今日头条', '西瓜视频'].map((item) => (
            <button type="button" key={item}>
              <span>{item.slice(0, 1)}</span>
              <b>{item}</b>
            </button>
          ))}
        </div>
      </SideCard>
      <SideCard title="最近常看" action="全部">
        <div className="dyHomeReplicaRecentGrid">
          {SIDE_RECENTS.map((item) => (
            <a href="/message/chat" key={item}>
              <Avatar name={item} />
              <b>{item}</b>
            </a>
          ))}
        </div>
      </SideCard>
      <SideCard title="常用功能">
        <div className="dyHomeReplicaFeatureGrid">
          {SIDE_FEATURES.map((item) => (
            <a href={item === '设置' ? '/me/right-menu/setting' : item === '观看历史' ? '/me/right-menu/look-history' : '/me'} key={item}>
              <span>{item.slice(0, 1)}</span>
              <b>{item}</b>
            </a>
          ))}
        </div>
      </SideCard>
      <button type="button" className="dyHomeReplicaSideMaskButton" onClick={onClose}>回到首页</button>
    </aside>
  );
}

function IndicatorHome({
  activeChannel,
  onOpenSidebar,
  onSetChannel,
}: {
  activeChannel: number;
  onOpenSidebar: () => void;
  onSetChannel: (index: number) => void;
}) {
  return (
    <header className="dyHomeReplicaIndicator">
      <button type="button" aria-label="打开侧边栏" onClick={onOpenSidebar}><HomeIcon name="menu" /></button>
      <nav aria-label="首页频道">
        {CHANNELS.map((channel, index) => (
          <button type="button" className={activeChannel === index ? 'active' : ''} onClick={() => onSetChannel(index)} key={channel}>
            {channel}
          </button>
        ))}
      </nav>
      <a href="/home/search" aria-label="搜索"><HomeIcon name="search" /></a>
    </header>
  );
}

function ActionRail({
  item,
  liked,
  onComment,
  onLike,
  onShare,
}: {
  item: FeedItem;
  liked: boolean;
  onComment: () => void;
  onLike: () => void;
  onShare: () => void;
}) {
  const [followed, setFollowed] = useState(false);
  const [collected, setCollected] = useState(false);

  return (
    <aside className="dyHomeReplicaActionRail">
      <div className="dyHomeReplicaAuthorCtn">
        <a href={publisherHref(item)} className="dyHomeReplicaAuthor" aria-label={`进入 ${item.author} 的主页`}>
          <Avatar name={item.author} src={item.avatarUrl} />
        </a>
        <span
          className={`dyHomeReplicaAuthorOptions${followed ? ' attention' : ''}`}
          role="button"
          tabIndex={0}
          aria-label={followed ? '已关注' : '关注'}
          onClick={() => setFollowed((value) => !value)}
          onKeyDown={(event) => {
            if (event.key === 'Enter' || event.key === ' ') {
              event.preventDefault();
              setFollowed((value) => !value);
            }
          }}
        >
          <img className="no" src="/douyin-assets/icons/add-light.png" alt="" />
          <img className="yes" src="/douyin-assets/icons/ok-red.png" alt="" />
        </span>
      </div>
      <button type="button" className={liked ? 'active' : ''} onClick={onLike}>
        <span className="dyHomeReplicaToolbarIcon"><HomeIcon name="heart" /></span>
        <small>{item.likes}</small>
      </button>
      <button type="button" onClick={onComment}>
        <span className="dyHomeReplicaToolbarIcon"><HomeIcon name="comment" /></span>
        <small>{item.comments}</small>
      </button>
      <button type="button" className={collected ? 'active collect' : 'collect'} onClick={() => setCollected((value) => !value)}>
        <span className="dyHomeReplicaToolbarIcon"><HomeIcon name="star" /></span>
        <small>{item.collects}</small>
      </button>
      <button type="button" onClick={onShare}>
        <span className="dyHomeReplicaToolbarIcon"><HomeIcon name="share" /></span>
        <small>{item.shares}</small>
      </button>
      <a href="/home/music" className="dyHomeReplicaMusicDisc"><HomeIcon name="music" /></a>
    </aside>
  );
}

function FeedSlide({
  item,
  active,
  shouldLoad,
  liked,
  paused,
  progressKey,
  onDoubleTapLike,
  onLike,
  onToggle,
  onEnterLive,
  onOpenSheet,
}: {
  item: FeedItem;
  active: boolean;
  shouldLoad: boolean;
  liked: boolean;
  paused: boolean;
  progressKey: string;
  onDoubleTapLike: () => void;
  onLike: () => void;
  onToggle: () => void;
  onEnterLive: (href: string) => void;
  onOpenSheet: (sheet: HomeSheet) => void;
}) {
  const [likeBurst, setLikeBurst] = useState(0);
  const [videoSourceIndex, setVideoSourceIndex] = useState(0);
  const [mediaFailed, setMediaFailed] = useState(false);
  const [buffering, setBuffering] = useState(false);
  const [playbackProgress, setPlaybackProgress] = useState({ src: '', value: 0 });
  const [videoFitState, setVideoFitState] = useState<{ src: string; fit: 'cover' | 'contain' }>({ src: '', fit: 'cover' });
  const videoRef = useRef<HTMLVideoElement | null>(null);
  const tapTimer = useRef<number | undefined>(undefined);
  const longPressTimer = useRef<number | undefined>(undefined);
  const longPressFired = useRef(false);
  const fallbackVideoSources = resolveLivePlaylist();
  const videoSources = (
    item.videoUrls?.length
      ? item.videoUrls
      : item.videoUrl
        ? [item.videoUrl]
        : fallbackVideoSources
  ).filter(Boolean);
  const videoSrc = videoSources[videoSourceIndex] || '';
  const sourceVideoFit = chooseVideoFitFromSize(item.mediaWidth, item.mediaHeight);
  const videoFit = videoFitState.src === videoSrc ? videoFitState.fit : sourceVideoFit;
  const shouldAttachVideo = active && shouldLoad;

  useEffect(() => {
    const video = videoRef.current;
    if (!video) return;
    if (!active || !shouldAttachVideo) {
      video.pause();
      if (!active) {
        try {
          video.currentTime = 0;
        } catch {
          // Ignore browsers that reject resetting an unloaded media element.
        }
      }
      video.removeAttribute('src');
      video.load();
      return;
    }
    if (mediaFailed || !videoSrc) return;
    if (active && !paused) {
      if (video.readyState < HTMLMediaElement.HAVE_FUTURE_DATA) setBuffering(true);
      void video.play().catch(() => undefined);
    } else {
      video.pause();
      if (!active) video.currentTime = 0;
    }
  }, [active, paused, mediaFailed, shouldAttachVideo, videoSrc]);

  useEffect(() => {
    return () => {
      if (tapTimer.current) window.clearTimeout(tapTimer.current);
      if (longPressTimer.current) window.clearTimeout(longPressTimer.current);
    };
  }, []);

  function clearLongPress() {
    if (longPressTimer.current) window.clearTimeout(longPressTimer.current);
    longPressTimer.current = undefined;
  }

  function handlePrimaryTap() {
    if (item.kind === 'live') {
      onEnterLive(item.liveHref || '/home/live');
      return;
    }
    if (longPressFired.current) {
      longPressFired.current = false;
      return;
    }
    if (tapTimer.current) {
      window.clearTimeout(tapTimer.current);
      tapTimer.current = undefined;
      onDoubleTapLike();
      setLikeBurst((value) => value + 1);
      return;
    }
    tapTimer.current = window.setTimeout(() => {
      tapTimer.current = undefined;
      onToggle();
    }, 210);
  }

  function handlePointerDown(event: PointerEvent) {
    if (event.pointerType === 'mouse' && event.button !== 0) return;
    longPressFired.current = false;
    clearLongPress();
    longPressTimer.current = window.setTimeout(() => {
      longPressFired.current = true;
      onOpenSheet('more');
    }, 520);
  }

  function handlePointerUp() {
    clearLongPress();
  }

  function updatePlaybackProgress(video: HTMLVideoElement) {
    const duration = video.duration;
    if (!Number.isFinite(duration) || duration <= 0) {
      setPlaybackProgress({ src: videoSrc, value: 0 });
      return;
    }
    setPlaybackProgress({ src: videoSrc, value: Math.max(0, Math.min(100, (video.currentTime / duration) * 100)) });
  }

  const visibleProgress = active && playbackProgress.src === videoSrc ? playbackProgress.value : 0;

  return (
    <section className={`dyHomeReplicaFeedSlide dyHomeReplicaTone-${item.tone} dyHomeReplicaFit-${videoFit}`}>
      {!mediaFailed && videoSrc ? (
        <video
          ref={videoRef}
          key={videoSrc}
          src={shouldAttachVideo ? videoSrc : undefined}
          poster={item.coverUrl}
          muted
          loop
          playsInline
          preload={shouldAttachVideo ? 'auto' : 'none'}
          onLoadStart={() => setBuffering(true)}
          onLoadedMetadata={(event) => {
            setVideoFitState({ src: videoSrc, fit: chooseVideoFit(event.currentTarget) });
            updatePlaybackProgress(event.currentTarget);
          }}
          onDurationChange={(event) => updatePlaybackProgress(event.currentTarget)}
          onTimeUpdate={(event) => updatePlaybackProgress(event.currentTarget)}
          onCanPlay={() => setBuffering(false)}
          onPlaying={() => setBuffering(false)}
          onWaiting={() => active && setBuffering(true)}
          onStalled={() => active && setBuffering(true)}
          onClick={handlePrimaryTap}
          onError={() => {
            if (videoSourceIndex < videoSources.length - 1) {
              setVideoSourceIndex((value) => value + 1);
              setBuffering(true);
              setPlaybackProgress({ src: videoSrc, value: 0 });
              return;
            }
            setMediaFailed(true);
            setBuffering(false);
            setPlaybackProgress({ src: videoSrc, value: 0 });
          }}
          onPointerDown={handlePointerDown}
          onPointerLeave={handlePointerUp}
          onPointerUp={handlePointerUp}
          onPointerCancel={handlePointerUp}
        />
      ) : null}
      <div
        className={`dyHomeReplicaVideoFallback${item.coverUrl ? ' hasCover' : ''}`}
        style={item.coverUrl ? { '--dy-cover': `url("${item.coverUrl}")` } as CSSProperties : undefined}
        aria-hidden="true"
        onClick={handlePrimaryTap}
        onPointerDown={handlePointerDown}
        onPointerLeave={handlePointerUp}
        onPointerUp={handlePointerUp}
        onPointerCancel={handlePointerUp}
      >
        <span>{item.sourceLabel || (item.kind === 'live' ? 'LIVE' : 'DOUYIN')}</span>
      </div>
      {active && shouldLoad && !paused && buffering && !mediaFailed ? (
        <div className="dyHomeReplicaVideoLoading" aria-hidden="true">
          <DouyinLoading />
        </div>
      ) : null}
      <div className="dyHomeReplicaProgress" aria-hidden="true">
        {active && videoSrc && !mediaFailed ? <i key={progressKey} style={{ width: `${visibleProgress}%` }} /> : null}
      </div>
      {likeBurst ? <LikeBurstIcon key={likeBurst} /> : null}
      {paused && active ? <button type="button" className="dyHomeReplicaPlayMark" onClick={onToggle}>▶</button> : null}
      {item.kind === 'live' ? (
        <a
          className="dyHomeReplicaLiveEnter"
          href={item.liveHref || '/home/live'}
          aria-label="进入直播间"
          onClick={(event) => {
            event.preventDefault();
            onEnterLive(item.liveHref || '/home/live');
          }}
        >
          点击进入直播间
        </a>
      ) : null}
      <section className="dyHomeReplicaCaption">
        {item.kind === 'live' ? <span className="dyHomeReplicaLiveTag">直播中</span> : null}
        <a className="dyHomeReplicaCaptionAuthor" href={publisherHref(item)}>@{item.author}</a>
        <p>{item.title}</p>
      </section>
      {item.kind === 'video' ? (
        <ActionRail
          item={item}
          liked={liked}
          onLike={onLike}
          onComment={() => onOpenSheet('comment')}
          onShare={() => onOpenSheet('share')}
        />
      ) : null}
    </section>
  );
}

const ME_TABS = ['作品', '私密', '喜欢', '收藏'] as const;
type MeTab = (typeof ME_TABS)[number];

function UserPanel() {
  const { user } = useAuthSession();
  const [activeTab, setActiveTab] = useState<MeTab>('作品');
  const isLoggedIn = Boolean(user);
  const displayName = user?.nickname?.trim() || user?.username?.trim() || '未登录';
  const douyinId = user?.username?.trim() || (isLoggedIn ? user?.id || '未设置' : '未设置');
  const avatarText = isLoggedIn ? displayName.slice(0, 1) : '';

  return (
    <aside className="dyHomeReplicaUserPanel">
      <header className="dyHomeReplicaMeFloat float">
        <a className="dyHomeReplicaMeEdit left" href={isLoggedIn ? '/me/edit' : '/login?next=/home'}>
          <svg className="iconify iconify--ri" width="1em" height="1em" viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M7.243 17.997H3v-4.243L14.435 2.319a1 1 0 0 1 1.414 0l2.829 2.828a1 1 0 0 1 0 1.415zm-4.243 2h18v2H3z" /></svg>
          <span>编辑资料</span>
        </a>
        <div className="dyHomeReplicaMeFloatActions right">
          <button type="button" className="dyHomeReplicaMeFloatItem item" aria-label="常用互动">
            <svg className="finger iconify iconify--fluent-emoji-high-contrast" width="1em" height="1em" viewBox="0 0 32 32" aria-hidden="true"><path fill="currentColor" d="M15.86 31c-6.5 0-10.876-5.269-10.876-13.109a3.42 3.42 0 0 1 1.176-2.843a3.3 3.3 0 0 1 1.854-.679c.151-1.585.76-3.231 2.944-3.39c.355-.026.711 0 1.058.08V4.531A3.53 3.53 0 0 1 15.531 1a3.457 3.457 0 0 1 3.453 3.531v5.61q.479-.055.956.008a3.53 3.53 0 0 1 2.344 1.435a2.9 2.9 0 0 1 1.5-.269a3.216 3.216 0 0 1 3.187 3.31C26.969 24.879 22.815 31 15.86 31M8.016 16.373a1.5 1.5 0 0 0-.614.243a1.59 1.59 0 0 0-.418 1.275C6.984 24.535 10.551 29 15.86 29c8.221 0 9.109-10.053 9.109-14.375c0-.8-.452-1.245-1.345-1.315a.86.86 0 0 0-.77.306a.98.98 0 0 1-.963.488a1 1 0 0 1-.845-.672c0-.005-.433-1.18-1.361-1.3a1.55 1.55 0 0 0-.7.036v1.113a1 1 0 1 1-2 0v-1.717l-.001-.015V4.531A1.453 1.453 0 0 0 15.531 3a1.54 1.54 0 0 0-1.515 1.531v8.2a1 1 0 0 1-1.762.648a1.53 1.53 0 0 0-1.15-.408c-.444.033-.937.069-1.088 1.453v2.092a1 1 0 1 1-2 0z" /></svg>
          </button>
          <a className="dyHomeReplicaMeFloatItem item" href="/people/find-acquaintance" aria-label="朋友">
            <svg className="iconify iconify--eva" width="1em" height="1em" viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M9 11a4 4 0 1 0-4-4a4 4 0 0 0 4 4m8 2a3 3 0 1 0-3-3a3 3 0 0 0 3 3m0 1a5 5 0 0 0-3.06 1.05A7 7 0 0 0 2 20a1 1 0 0 0 2 0a5 5 0 0 1 10 0a1 1 0 0 0 2 0a6.9 6.9 0 0 0-.86-3.35A3 3 0 0 1 20 19a1 1 0 0 0 2 0a5 5 0 0 0-5-5" /></svg>
          </a>
          <a className="dyHomeReplicaMeFloatItem item" href="/home/search" aria-label="搜索">
            <svg className="iconify iconify--ic" width="1em" height="1em" viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M15.5 14h-.79l-.28-.27a6.5 6.5 0 0 0 1.48-5.34c-.47-2.78-2.79-5-5.59-5.34a6.505 6.505 0 0 0-7.27 7.27c.34 2.8 2.56 5.12 5.34 5.59a6.5 6.5 0 0 0 5.34-1.48l.27.28v.79l4.25 4.25c.41.41 1.08.41 1.49 0s.41-1.08 0-1.49zM9.5 14C7.01 14 5 11.99 5 9.5S7.01 5 9.5 5S14 7.01 14 9.5S11.99 14 9.5 14" /></svg>
          </a>
          <a className="dyHomeReplicaMeFloatItem item" href="/me/right-menu/setting" aria-label="菜单">
            <svg className="iconify iconify--ic" width="1em" height="1em" viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M4 18h16c.55 0 1-.45 1-1s-.45-1-1-1H4c-.55 0-1 .45-1 1s.45 1 1 1m0-5h16c.55 0 1-.45 1-1s-.45-1-1-1H4c-.55 0-1 .45-1 1s.45 1 1 1M3 7c0 .55.45 1 1 1h16c.55 0 1-.45 1-1s-.45-1-1-1H4c-.55 0-1 .45-1 1" /></svg>
          </a>
        </div>
      </header>

      <section className="dyHomeReplicaMeProfile">
        <header className={`dyHomeReplicaMeCover${isLoggedIn ? '' : ' isAnonymous'}`}>
          <div className="dyHomeReplicaMeIdentity">
            <span className={`dyHomeReplicaMeAvatar${isLoggedIn ? '' : ' isEmpty'}`}>{avatarText}</span>
            <div>
              <h2>{displayName}</h2>
              <p>
                抖音号：{douyinId}
                {isLoggedIn ? <img src="/douyin-assets/me/qrcode-gray.png" alt="" /> : null}
              </p>
            </div>
          </div>
        </header>

        <section className="dyHomeReplicaMeDetail">
          <div className="dyHomeReplicaMeHead">
            <nav className="dyHomeReplicaMeHeat" aria-label="账号数据">
            {['0 获赞', '0 朋友', '0 关注', '0 粉丝'].map((item) => {
              const [value, label] = item.split(' ');
                return <span className="dyHomeReplicaMeHeatText" key={item}><b>{value}</b><small>{label}</small></span>;
            })}
            </nav>
            <a className="dyHomeReplicaMePrimaryButton" href={isLoggedIn ? '/people/find-acquaintance' : '/login?next=/home'}>
            {isLoggedIn ? '添加朋友' : '登录'}
            </a>
          </div>

          <div className="dyHomeReplicaMeSignature">
            <div>{isLoggedIn ? '点击添加介绍，让大家认识你...' : '登录后完善资料，让大家认识你...'}</div>
          </div>
          <div className="dyHomeReplicaMeMore">
            <span className="dyHomeReplicaMeMetaItem">{isLoggedIn ? '暂无资料' : '未登录'}</span>
          </div>

          <nav className="dyHomeReplicaMeOther" aria-label="个人功能">
            <a href="/shop">
              <svg viewBox="0 0 24 24" aria-hidden="true">
                <g fill="none">
                  <path fill="currentColor" d="M3 2.25a.75.75 0 0 0 0 1.5zM5 3l.748-.058A.75.75 0 0 0 5 2.25zm16 3l.745.083A.75.75 0 0 0 21 5.25zM5.23 6l-.747.058zm13.109 9.119l.053.748zm-10.355.74l-.053-.749zM3 3.75h2v-1.5H3zm5.037 12.856l10.355-.74l-.107-1.495l-10.354.74zm12.892-3.179l.816-7.344l-1.49-.166l-.816 7.345zM4.252 3.057l.231 3l1.496-.115l-.231-3zm.231 3l.617 8.017l1.495-.115l-.616-8.017zM21 5.25H5.23v1.5H21zm-2.608 10.617a2.75 2.75 0 0 0 2.537-2.44l-1.49-.165a1.25 1.25 0 0 1-1.154 1.109zM7.931 15.11a1.25 1.25 0 0 1-1.336-1.15l-1.495.114a2.75 2.75 0 0 0 2.937 2.532z" />
                  <path stroke="currentColor" strokeLinejoin="round" strokeWidth="2.25" d="M8.5 20.5h.01v.01H8.5zm9 0h.01v.01h-.01z" />
                </g>
              </svg>
              <span>抖音商城</span>
            </a>
            <a href="/home/music">
              <svg viewBox="0 0 24 24" aria-hidden="true">
                <g fill="none" stroke="currentColor" strokeLinecap="round" strokeWidth="1.5">
                  <circle cx="6" cy="18" r="3" strokeLinejoin="round" />
                  <path strokeLinejoin="round" d="M9 18V5" />
                  <path d="M21 3L9 5m12 2L9 9" />
                  <circle cx="18" cy="16" r="3" strokeLinejoin="round" />
                  <path strokeLinejoin="round" d="M21 16V3" />
                </g>
              </svg>
              <span>我的音乐</span>
            </a>
            <a href="/message">
              <svg viewBox="0 0 14 14" aria-hidden="true">
                <g fill="none" stroke="currentColor" strokeLinecap="round" strokeLinejoin="round">
                  <path d="M9.25 5a4.25 4.25 0 0 1 3.54 6.6l.71 1.9l-2.39-.43A4.25 4.25 0 1 1 9.25 5" />
                  <path d="M9.86 2.51A5.24 5.24 0 0 0 .5 5.75a5.2 5.2 0 0 0 .88 2.91L.5 11l2.12-.38" />
                </g>
              </svg>
              <span>我的群聊</span>
            </a>
            <a href="/me/right-menu/setting">
              <svg viewBox="0 0 24 24" aria-hidden="true">
                <g fill="none">
                  <path fill="currentColor" d="M3 2.25a.75.75 0 0 0 0 1.5zM5 3l.748-.058A.75.75 0 0 0 5 2.25zm16 3l.745.083A.75.75 0 0 0 21 5.25zM5.23 6l-.747.058zm13.109 9.119l.053.748zm-10.355.74l-.053-.749zM3 3.75h2v-1.5H3zm5.037 12.856l10.355-.74l-.107-1.495l-10.354.74zm12.892-3.179l.816-7.344l-1.49-.166l-.816 7.345zM4.252 3.057l.231 3l1.496-.115l-.231-3zm.231 3l.617 8.017l1.495-.115l-.616-8.017zM21 5.25H5.23v1.5H21zm-2.608 10.617a2.75 2.75 0 0 0 2.537-2.44l-1.49-.165a1.25 1.25 0 0 1-1.154 1.109zM7.931 15.11a1.25 1.25 0 0 1-1.336-1.15l-1.495.114a2.75 2.75 0 0 0 2.937 2.532z" />
                  <path stroke="currentColor" strokeLinejoin="round" strokeWidth="2.25" d="M8.5 20.5h.01v.01H8.5zm9 0h.01v.01h-.01z" />
                </g>
              </svg>
              <span>查看更多</span>
            </a>
          </nav>
        </section>
      </section>

      <nav className="dyHomeReplicaMeTabs" aria-label="作品分类">
        {ME_TABS.map((tab) => (
          <button type="button" className={activeTab === tab ? 'active' : ''} onClick={() => setActiveTab(tab)} key={tab}>{tab}</button>
        ))}
      </nav>

      <section className="dyHomeReplicaMeEmpty">
        {activeTab === '作品' ? null : <img src="/douyin-assets/me/lock-gray.png" alt="" />}
        <p>{activeTab === '作品' ? '还没有发布作品' : activeTab === '私密' ? '只有你能看到设为私密的作品和日常' : activeTab === '喜欢' ? '只有你能看到自己的喜欢列表' : '只有你能看到自己的收藏列表'}</p>
        <small>暂时没有更多了</small>
      </section>

    </aside>
  );
}

function SheetFrame({
  title,
  label,
  className = '',
  onClose,
  children,
  footer,
}: {
  title: string;
  label: string;
  className?: string;
  onClose: () => void;
  children: ReactNode;
  footer?: ReactNode;
}) {
  const sheetTouchStart = useRef({ x: 0, y: 0 });

  function handleSheetTouchStart(event: TouchEvent) {
    event.stopPropagation();
    const touch = event.touches[0];
    sheetTouchStart.current = { x: touch?.clientX ?? 0, y: touch?.clientY ?? 0 };
  }

  function handleSheetTouchEnd(event: TouchEvent) {
    event.stopPropagation();
    const touch = event.changedTouches[0];
    const dx = (touch?.clientX ?? sheetTouchStart.current.x) - sheetTouchStart.current.x;
    const dy = (touch?.clientY ?? sheetTouchStart.current.y) - sheetTouchStart.current.y;
    if (dy > 54 && Math.abs(dy) > Math.abs(dx) * 1.2) onClose();
  }

  return (
    <section
      className={`dyHomeReplicaSheet ${className}`.trim()}
      role="dialog"
      aria-modal="true"
      aria-label={label}
      onTouchStart={handleSheetTouchStart}
      onTouchEnd={handleSheetTouchEnd}
    >
      <span className="dyHomeReplicaSheetHandle" aria-hidden="true" />
      <header><b>{title}</b><button type="button" onClick={onClose}>×</button></header>
      {children}
      {footer}
    </section>
  );
}

function ShareSheet({ onClose, onFriend }: { onClose: () => void; onFriend: () => void }) {
  return (
    <SheetFrame title="分享到" label="分享" className="dyHomeReplicaShareSheet" onClose={onClose}>
      <div>
        {SHARE_OPTIONS.map((item) => (
          <button type="button" onClick={item === '私信朋友' ? onFriend : item === '举报' ? () => window.location.assign('/home/report') : undefined} key={item}>
            <span>{item.slice(0, 1)}</span>
            <b>{item}</b>
          </button>
        ))}
      </div>
    </SheetFrame>
  );
}

function MoreSheet({ onClose }: { onClose: () => void }) {
  return (
    <SheetFrame title="更多操作" label="更多" className="dyHomeReplicaMoreSheet" onClose={onClose}>
      <div>
        {['倍速播放', '自动连播', '清屏', '不感兴趣', '内容偏好', '投诉'].map((item) => <button type="button" key={item}>{item}</button>)}
      </div>
    </SheetFrame>
  );
}

function FriendSheet({ onClose }: { onClose: () => void }) {
  return (
    <SheetFrame
      title="私信给"
      label="分享给朋友"
      className="dyHomeReplicaFriendSheet"
      onClose={onClose}
      footer={<footer><button type="button">发送</button></footer>}
    >
      <label><span>⌕</span><input placeholder="搜索" /></label>
      <div>
        {SIDE_RECENTS.map((name) => (
          <button type="button" key={name}>
            <Avatar name={name} />
            <span><b>{name}</b><small>最近互动</small></span>
            <i />
          </button>
        ))}
      </div>
    </SheetFrame>
  );
}

export function HomePage() {
  const restoredHomeState = useMemo(() => (isMeEntryPath() ? null : readHomeReturnState()), []);
  const [baseIndex, setBaseIndex] = useState(() => clampIndex(restoredHomeState?.baseIndex ?? initialBaseIndex(), 3));
  const [channelIndex, setChannelIndex] = useState(() => clampIndex(restoredHomeState?.channelIndex ?? 4, CHANNELS.length));
  const [itemIndexes, setItemIndexes] = useState(() => {
    const saved = restoredHomeState?.itemIndexes || [];
    return CHANNELS.map((_, index) => Math.max(0, saved[index] || 0));
  });
  const [pausedId, setPausedId] = useState('');
  const [likedIds, setLikedIds] = useState<Set<string>>(() => new Set());
  const [sheet, setSheet] = useState<HomeSheet>(null);
  const [commentTarget, setCommentTarget] = useState<CommentTarget | null>(null);
  const [refreshing, setRefreshing] = useState(false);
  const [publicRooms, setPublicRooms] = useState<Room[]>([]);
  const [remoteVideoItems, setRemoteVideoItems] = useState<FeedItem[]>([]);
  const touchStart = useRef({ x: 0, y: 0, at: 0 });
  const fallbackVideoItems = useMemo(() => shuffleItems(VIDEO_FEED_ITEMS), []);
  const videoItems = remoteVideoItems.length ? remoteVideoItems : fallbackVideoItems;
  const liveItems = useMemo(() => liveRoomFeedItems(publicRooms), [publicRooms]);
  const channelFeeds = useMemo(() => CHANNELS.map((_, index) => channelItems(index, liveItems, videoItems)), [liveItems, videoItems]);
  const items = channelFeeds[channelIndex] || FEED_ITEMS;
  const itemIndex = clampIndex(itemIndexes[channelIndex] || 0, items.length);

  useEffect(() => {
    let disposed = false;
    void listPublicRooms().then((rooms) => {
      if (!disposed) setPublicRooms(rooms);
    }).catch(() => {
      if (!disposed) setPublicRooms([]);
    });
    void fetchDouyinFeedItems().then((items) => {
      if (!disposed) setRemoteVideoItems(items);
    }).catch(() => {
      if (!disposed) setRemoteVideoItems([]);
    });
    return () => {
      disposed = true;
    };
  }, []);

  useEffect(() => {
    const syncEntryPath = () => {
      if (isMeEntryPath()) setBaseIndex(2);
      else if (location.pathname === '/' || location.pathname === '/home') setBaseIndex(1);
    };
    window.addEventListener('popstate', syncEntryPath);
    window.addEventListener(SPA_NAVIGATE_EVENT, syncEntryPath);
    syncEntryPath();
    return () => {
      window.removeEventListener('popstate', syncEntryPath);
      window.removeEventListener(SPA_NAVIGATE_EVENT, syncEntryPath);
    };
  }, []);

  function setCurrentItemIndex(next: number | ((current: number) => number)) {
    setItemIndexes((current) => {
      const copy = current.slice();
      const value = typeof next === 'function' ? next(copy[channelIndex] || 0) : next;
      copy[channelIndex] = wrapIndex(value, items.length);
      return copy;
    });
  }

  function handleTouchStart(event: TouchEvent) {
    const touch = event.touches[0];
    touchStart.current = { x: touch?.clientX ?? 0, y: touch?.clientY ?? 0, at: Date.now() };
  }

  function handleTouchEnd(event: TouchEvent) {
    if (sheet) return;
    const touch = event.changedTouches[0];
    const dx = (touch?.clientX ?? touchStart.current.x) - touchStart.current.x;
    const dy = (touch?.clientY ?? touchStart.current.y) - touchStart.current.y;
    const elapsed = Date.now() - touchStart.current.at;
    if (swipeDistance(dx, elapsed, HORIZONTAL_SWIPE_DISTANCE) && Math.abs(dx) > Math.abs(dy) * 1.22) {
      if (baseIndex !== 1) {
        setBaseIndex((current) => clampIndex(current + (dx < 0 ? 1 : -1), 3));
        return;
      }
      if (dx < 0) {
        if (channelIndex < CHANNELS.length - 1) setChannelIndex((current) => current + 1);
        else setBaseIndex(2);
      } else if (channelIndex > 0) {
        setChannelIndex((current) => current - 1);
      } else {
        setBaseIndex(0);
      }
      setPausedId('');
      return;
    }
    if (baseIndex === 1 && swipeDistance(dy, elapsed, VERTICAL_SWIPE_DISTANCE) && Math.abs(dy) > Math.abs(dx) * 1.08) {
      if (dy > 72 && itemIndex === 0) {
        setRefreshing(true);
        void Promise.allSettled([listPublicRooms(), fetchDouyinFeedItems()])
          .then(([roomsResult, videosResult]) => {
            if (roomsResult.status === 'fulfilled') setPublicRooms(roomsResult.value);
            if (videosResult.status === 'fulfilled') setRemoteVideoItems(videosResult.value);
          })
          .finally(() => window.setTimeout(() => setRefreshing(false), 450));
        return;
      }
      setCurrentItemIndex((current) => current + (dy < 0 ? 1 : -1));
      setPausedId('');
    }
  }

  function selectChannel(index: number) {
    setChannelIndex(index);
    setPausedId('');
  }

  function toggleLike(itemId: string) {
    setLikedIds((current) => {
      const next = new Set(current);
      if (next.has(itemId)) next.delete(itemId);
      else next.add(itemId);
      return next;
    });
  }

  function likeItem(itemId: string) {
    setLikedIds((current) => {
      if (current.has(itemId)) return current;
      const next = new Set(current);
      next.add(itemId);
      return next;
    });
  }

  function openSheetForItem(nextSheet: HomeSheet, item: FeedItem) {
    if (nextSheet === 'comment') {
      setCommentTarget({ videoId: item.awemeId || item.id, comments: item.comments });
    }
    setSheet(nextSheet);
  }

  function enterLiveRoom(href: string) {
    writeHomeReturnState({ baseIndex: 1, channelIndex, itemIndexes });
    navigateTo(href);
  }

  function handleFooterTab(tab: Exclude<DouyinTab, 'publish'>, href: string) {
    if (tab === 'home') {
      setBaseIndex(1);
      navigateTo('/home');
      return;
    }
    if (tab === 'me') {
      setBaseIndex(2);
      navigateTo('/me');
      return;
    }
    navigateTo(href);
  }

  const activeItem = items[itemIndex] || items[0];
  const horizontalOffset = baseIndex * (100 / 3);
  const shellClassName = [
    'mobileShell dyHomeReplicaShell',
    sheet ? 'isSheetOpen' : '',
    sheet === 'comment' ? 'isCommentSheetOpen' : '',
  ].filter(Boolean).join(' ');

  return (
    <main className={shellClassName} onTouchStart={handleTouchStart} onTouchEnd={handleTouchEnd}>
      <section className="dyHomeReplicaHorizontal" style={{ transform: `translateX(-${horizontalOffset}%)` }}>
        <Sidebar onClose={() => setBaseIndex(1)} />
        <section className="dyHomeReplicaMain">
          <IndicatorHome
            activeChannel={channelIndex}
            onOpenSidebar={() => setBaseIndex(0)}
            onSetChannel={selectChannel}
          />
          <div className={`dyHomeReplicaRefreshNotice${refreshing ? ' active' : ''}`}>{refreshing ? '刷新中' : '下拉刷新内容'}</div>
          <section className="dyHomeReplicaChannelTrack" style={{ transform: `translateX(-${channelIndex * 100}%)` }}>
            {channelFeeds.map((feed, feedIndex) => {
              const paneItemIndex = clampIndex(itemIndexes[feedIndex] || 0, feed.length);
              return (
                <section className="dyHomeReplicaChannelPane" key={CHANNELS[feedIndex]}>
                  {feed.length ? (
                    <section className="dyHomeReplicaVertical" style={{ transform: `translateY(-${paneItemIndex * 100}%)` }}>
                      {feed.map((item, index) => (
                        <FeedSlide
                          item={item}
                          active={baseIndex === 1 && feedIndex === channelIndex && index === itemIndex}
                          shouldLoad={baseIndex === 1 && feedIndex === channelIndex && Math.abs(index - paneItemIndex) <= 2}
                          liked={likedIds.has(item.id)}
                          paused={pausedId === item.id}
                          progressKey={`${feedIndex}-${index}-${item.id}-${pausedId === item.id ? 'paused' : 'playing'}`}
                          onDoubleTapLike={() => likeItem(item.id)}
                          onLike={() => toggleLike(item.id)}
                          onToggle={() => setPausedId((current) => (current === item.id ? '' : item.id))}
                          onEnterLive={enterLiveRoom}
                          onOpenSheet={(nextSheet) => openSheetForItem(nextSheet, item)}
                          key={`${feedIndex}-${index}-${item.id}`}
                        />
                      ))}
                    </section>
                  ) : feedIndex === LIVE_CHANNEL_INDEX ? (
                    <section className="dyHomeReplicaLiveEmpty">
                      <b>目前暂无直播</b>
                      <span>有商家主账号开拍后，直播间会出现在这里。</span>
                      <button
                        type="button"
                        onClick={() => {
                          setRefreshing(true);
                          void listPublicRooms()
                            .then((rooms) => setPublicRooms(rooms))
                            .catch(() => setPublicRooms([]))
                            .finally(() => window.setTimeout(() => setRefreshing(false), 300));
                        }}
                      >
                        刷新
                      </button>
                    </section>
                  ) : null}
                </section>
              );
            })}
          </section>
        </section>
        <UserPanel />
      </section>
      <DouyinTabBar active={baseIndex === 2 ? 'me' : 'home'} onTab={handleFooterTab} />
      {sheet === 'comment' && (commentTarget || activeItem) ? (
        <DouyinCommentSheet
          videoId={commentTarget?.videoId || activeItem.awemeId || activeItem.id}
          onClose={() => setSheet(null)}
        />
      ) : null}
      {sheet === 'share' && activeItem ? <ShareSheet onClose={() => setSheet(null)} onFriend={() => setSheet('friend')} /> : null}
      {sheet === 'more' ? <MoreSheet onClose={() => setSheet(null)} /> : null}
      {sheet === 'friend' ? <FriendSheet onClose={() => setSheet(null)} /> : null}
    </main>
  );
}
