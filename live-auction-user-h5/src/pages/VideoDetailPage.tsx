import { useEffect, useRef, useState, type TouchEvent } from 'react';
import { DouyinCommentSheet } from '../shared/ui/DouyinCommentSheet';
import { DouyinLoading } from '../shared/ui/DouyinLoading';

type RawAuthor = {
  uid?: string;
  short_id?: string;
  unique_id?: string;
  nickname?: string;
  avatar_168x168?: { url_list?: string[] };
  avatar_300x300?: { url_list?: string[] };
  avatar_thumb?: { url_list?: string[] };
  cover_url?: Array<{ url_list?: string[] }>;
};

type RawFeedVideo = {
  aweme_id?: string;
  desc?: string;
  author?: RawAuthor;
  music?: {
    title?: string;
    author?: string;
    cover_thumb?: { url_list?: string[] };
    cover_medium?: { url_list?: string[] };
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

type VideoSheet = 'comments' | 'share' | null;

const FEED_SOURCE = '/data/douyin-feed.json';
const USER_VIDEO_SOURCE = '/data/user-video-list';
const DOUYIN_IMAGE_BASE = 'https://liveauction.tos-cn-beijing.volces.com/douyin-h5/images/';
const SWIPE_DISTANCE = 42;
const DY_HEART_PATH = 'M453.036 88.712C493.774 30.664 560.66 0 634.251 0C774.403 0 890.972 121.59 890.972 266.137C890.972 266.155 890.972 266.173 890.972 266.191C890.981 266.171 890.991 266.151 891 266.131C891 270.252 890.93 273.372 890.878 275.668C890.806 278.819 890.77 280.416 891 280.916C890.469 311.319 885.222 336.438 875.899 369.628C870.609 375.588 865.694 386.812 860.798 399.199C853.074 411.197 850.151 417.009 845.697 428.77C841.152 436.399 836.324 444.063 831.245 451.744C793.796 508.544 743.708 565.074 693.667 615.252C615.336 694.277 535.544 760.058 500.832 788.675C491.247 796.576 485.1 801.644 483.368 803.375C471.067 815.678 458.765 815.986 446.463 815.994C446.139 815.998 445.814 816 445.486 816C420.25 816 407.632 803.381 395.014 790.763C394.051 789.8 391.601 787.779 387.858 784.783C349.625 756.6 263.586 687.786 182.742 604.783C121.066 542.02 61.622 470.007 29.092 399.588C16.474 374.351 0.731 314.264 0 280.922C0.269 280.655 0.227 279.049 0.144 275.844C0.083 273.498 0 270.297 0 266.137C0 121.524 116.502 0 256.721 0C330.179 0 397.131 30.664 453.036 88.712z';

function isSignedExpiringAsset(url?: string) {
  return Boolean(url && (/[?&]x-expires=/i.test(url) || /^https?:\/\/[^/]*-sign\.douyinpic\.com/i.test(url)));
}

function firstStableUrl(urlList?: string[]) {
  return Array.isArray(urlList) ? urlList.find((url) => Boolean(url) && !isSignedExpiringAsset(url)) || '' : '';
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

function isVideoPlaybackUrl(url: string) {
  return !/\.mp3(?:$|\?)|\/ies-music\//i.test(url);
}

function avatarUrl(author?: RawAuthor) {
  return normalizeRemoteAsset(
    firstStableUrl(author?.avatar_300x300?.url_list)
      || firstStableUrl(author?.avatar_168x168?.url_list)
      || firstStableUrl(author?.avatar_thumb?.url_list)
      || firstStableUrl(author?.cover_url?.[0]?.url_list),
  );
}

function coverUrl(video?: RawFeedVideo) {
  return normalizeRemoteAsset(firstStableUrl(video?.video?.cover?.url_list));
}

function videoUrl(video?: RawFeedVideo) {
  return normalizeRemoteVideoUrl((video?.video?.play_addr?.url_list || []).find(isVideoPlaybackUrl));
}

function videoUrls(video?: RawFeedVideo) {
  return (video?.video?.play_addr?.url_list || [])
    .filter(isVideoPlaybackUrl)
    .map(normalizeRemoteVideoUrl)
    .filter(Boolean);
}

function douyinId(video?: RawFeedVideo) {
  const author = video?.author;
  return author?.unique_id || author?.short_id || author?.uid || `dy_${video?.aweme_id?.slice(-8) || 'user'}`;
}

function formatCount(value?: number, fallback = '0') {
  if (typeof value !== 'number' || !Number.isFinite(value)) return fallback;
  if (value >= 100000000) return `${(value / 100000000).toFixed(value >= 1000000000 ? 0 : 1)}亿`;
  if (value >= 10000) return `${(value / 10000).toFixed(value >= 100000 ? 0 : 1)}万`;
  return value.toLocaleString('zh-CN');
}

function musicText(video?: RawFeedVideo) {
  if (!video?.music?.title) return '原创音乐';
  return `${video.music.title}${video.music.author ? ` · ${video.music.author}` : ''}`;
}

function musicCoverUrl(video?: RawFeedVideo) {
  return normalizeRemoteAsset(
    firstStableUrl(video?.author?.avatar_168x168?.url_list)
      || firstStableUrl(video?.author?.avatar_300x300?.url_list)
      || firstStableUrl(video?.author?.avatar_thumb?.url_list)
      || firstStableUrl(video?.music?.cover_thumb?.url_list)
      || firstStableUrl(video?.music?.cover_medium?.url_list),
  );
}

function SearchMiniIcon() {
  return (
    <svg className="icon" viewBox="0 0 512 512" aria-hidden="true">
      <path fill="currentColor" d="M456.69 421.39 362.6 327.3a173.8 173.8 0 0 0 34.84-104.58C397.44 126.38 319.06 48 222.72 48S48 126.38 48 222.72s78.38 174.72 174.72 174.72A173.8 173.8 0 0 0 327.3 362.6l94.09 94.09a25 25 0 0 0 35.3-35.3M97.92 222.72a124.8 124.8 0 1 1 124.8 124.8 124.95 124.95 0 0 1-124.8-124.8" />
    </svg>
  );
}

function BackMiniIcon() {
  return (
    <svg className="back" viewBox="0 0 48 48" aria-hidden="true">
      <path fill="none" stroke="currentColor" strokeLinecap="round" strokeLinejoin="round" strokeWidth="4" d="M31 36 19 24l12-12" />
    </svg>
  );
}

function chooseMediaFit(video?: RawFeedVideo): 'cover' | 'contain' {
  const width = video?.video?.width;
  const height = video?.video?.height;
  if (!width || !height) return 'cover';
  const frameAspect = window.innerWidth / Math.max(1, window.innerHeight - 58);
  const videoAspect = width / height;
  const cropRatio = videoAspect > frameAspect
    ? 1 - (frameAspect / videoAspect)
    : 1 - (videoAspect / frameAspect);
  return cropRatio <= 0.16 ? 'cover' : 'contain';
}

function goBack() {
  if (window.history.length > 1) {
    window.history.back();
    return;
  }
  window.location.assign('/home');
}

async function fetchRows(source: string) {
  const response = await fetch(source);
  if (!response.ok) throw new Error(`load failed: ${response.status}`);
  const rows = await response.json() as RawFeedVideo[];
  return Array.isArray(rows) ? rows : [];
}

async function loadDetailRows(awemeId: string, requestedAuthorId: string) {
  const feedRows = await fetchRows(FEED_SOURCE);
  const selectedFromFeed = feedRows.find((video) => video.aweme_id === awemeId);
  const authorId = requestedAuthorId || douyinId(selectedFromFeed);
  let rows: RawFeedVideo[] = [];

  if (authorId) {
    try {
      rows = await fetchRows(`${USER_VIDEO_SOURCE}/user-${encodeURIComponent(authorId)}.json`);
    } catch {
      rows = [];
    }
  }

  const selectedFromAuthor = rows.find((video) => video.aweme_id === awemeId);
  const selected = selectedFromAuthor || selectedFromFeed || rows[0] || feedRows[0];
  const fallbackRows = selected
    ? feedRows.filter((video) => douyinId(video) === douyinId(selected))
    : feedRows;
  const sourceRows = rows.length ? rows : fallbackRows.length ? fallbackRows : feedRows;
  const baseAuthor = selected?.author || rows.find((video) => video.author)?.author;
  const hydratedRows: RawFeedVideo[] = sourceRows.map((video) => ({
    ...video,
    author: video.author || baseAuthor,
  }));
  if (selected && !hydratedRows.some((video) => video.aweme_id === selected.aweme_id)) {
    hydratedRows.unshift(selected);
  }
  const initialIndex = Math.max(0, hydratedRows.findIndex((video) => video.aweme_id === awemeId));
  return { rows: hydratedRows, initialIndex };
}

function DetailIcon({ name }: { name: 'heart' | 'comment' | 'star' | 'share' }) {
  if (name === 'heart') {
    return <svg viewBox="0 0 891 816" aria-hidden="true"><path d={DY_HEART_PATH} /></svg>;
  }
  if (name === 'comment') {
    return <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M21.25 8.18a9.8 9.8 0 0 0-2.16-3.25 10 10 0 0 0-14.15 0 9.8 9.8 0 0 0-2.17 3.25A10 10 0 0 0 2.01 12a9.7 9.7 0 0 0 .74 3.77l-.5 3.65a1.95 1.95 0 0 0 1.29 2.26c.297.098.613.122.92.07l3.65-.54a9.8 9.8 0 0 0 3.88.79 10 10 0 0 0 9.24-13.82zM7.73 13.61a1.61 1.61 0 1 1 .001-3.22 1.61 1.61 0 0 1 0 3.22m4.28 0a1.61 1.61 0 1 1 .001-3.22 1.61 1.61 0 0 1 0 3.22m4.28 0a1.61 1.61 0 1 1 .001-3.22 1.61 1.61 0 0 1 0 3.22" /></svg>;
  }
  if (name === 'star') {
    return <svg viewBox="0 0 24 24" aria-hidden="true"><path d="m12 17.27 4.15 2.51c.76.46 1.69-.22 1.49-1.08l-1.1-4.72 3.67-3.18c.67-.58.31-1.68-.57-1.75l-4.83-.41-1.89-4.46c-.34-.81-1.5-.81-1.84 0L9.19 8.63l-4.83.41c-.88.07-1.24 1.17-.57 1.75l3.67 3.18-1.1 4.72c-.2.86.73 1.54 1.49 1.08z" /></svg>;
  }
  if (name === 'share') {
    return <img className="dyVideoReplicaShareImage" src="/douyin-assets/icons/share-white-full.png" alt="" />;
  }
  return null;
}

function VideoSlide({
  item,
  active,
  liked,
  onToggleLike,
  collected,
  onToggleCollect,
  followed,
  onToggleFollow,
  onOpenComments,
  onOpenShare,
}: {
  item: RawFeedVideo;
  active: boolean;
  liked: boolean;
  onToggleLike: () => void;
  collected: boolean;
  onToggleCollect: () => void;
  followed: boolean;
  onToggleFollow: () => void;
  onOpenComments: () => void;
  onOpenShare: () => void;
}) {
  const videoRef = useRef<HTMLVideoElement | null>(null);
  const sources = videoUrls(item);
  const fallbackSource = videoUrl(item);
  const sourceList = sources.length ? sources : fallbackSource ? [fallbackSource] : [];
  const [sourceIndex, setSourceIndex] = useState(0);
  const [mediaFailed, setMediaFailed] = useState(false);
  const [buffering, setBuffering] = useState(false);
  const [avatarFailed, setAvatarFailed] = useState(false);
  const [discFailed, setDiscFailed] = useState(false);
  const source = sourceList[sourceIndex] || '';
  const attachedSource = active ? source : '';
  const poster = coverUrl(item);
  const authorName = item.author?.nickname || '抖音创作者';
  const avatar = avatarUrl(item.author) || poster;
  const musicCover = musicCoverUrl(item) || avatar || poster;

  useEffect(() => {
    const node = videoRef.current;
    if (!node) return;
    if (!active) {
      node.pause();
      node.removeAttribute('src');
      node.load();
      return;
    }
    if (mediaFailed || !source) return;
    if (node.readyState < HTMLMediaElement.HAVE_FUTURE_DATA) setBuffering(true);
    void node.play().catch(() => undefined);
  }, [active, mediaFailed, source]);

  return (
    <section className="dyVideoReplicaSlide">
      {!mediaFailed && attachedSource ? (
        <video
          ref={videoRef}
          key={attachedSource}
          src={attachedSource}
          poster={poster}
          autoPlay={active}
          muted
          loop
          playsInline
          preload={active ? 'auto' : 'metadata'}
          style={{ objectFit: chooseMediaFit(item) }}
          onLoadStart={() => setBuffering(true)}
          onCanPlay={() => setBuffering(false)}
          onPlaying={() => setBuffering(false)}
          onWaiting={() => active && setBuffering(true)}
          onStalled={() => active && setBuffering(true)}
          onError={() => {
            if (sourceIndex < sourceList.length - 1) {
              setSourceIndex((value) => value + 1);
              setBuffering(true);
              return;
            }
            setMediaFailed(true);
            setBuffering(false);
          }}
        />
      ) : poster ? (
        <img className="dyVideoReplicaMediaImage" src={poster} alt="" loading="lazy" referrerPolicy="no-referrer" style={{ objectFit: chooseMediaFit(item) }} />
      ) : (
        <div className="dyVideoReplicaMediaFallback">抖音素材</div>
      )}
      {active && buffering && !mediaFailed && source ? (
        <div className="dyVideoReplicaLoadingOverlay" aria-hidden="true">
          <DouyinLoading />
        </div>
      ) : null}
      <div className="dyVideoReplicaBackdrop" aria-hidden="true" />
      <aside className="dyVideoReplicaRail">
        <div className="dyVideoReplicaAvatarCtn">
          <a href={`/user?awemeId=${encodeURIComponent(item.aweme_id || '')}`} className="dyVideoReplicaAvatar" aria-label={`进入 ${authorName} 的主页`}>
            {avatar && !avatarFailed ? <img src={avatar} alt="" referrerPolicy="no-referrer" onError={() => setAvatarFailed(true)} /> : <span>{authorName.slice(0, 1)}</span>}
          </a>
          <span
            className={`dyVideoReplicaAvatarOptions${followed ? ' attention' : ''}`}
            role="button"
            tabIndex={0}
            aria-label={followed ? '已关注' : '关注'}
            onClick={onToggleFollow}
            onKeyDown={(event) => {
              if (event.key === 'Enter' || event.key === ' ') {
                event.preventDefault();
                onToggleFollow();
              }
            }}
          >
            <img className="no" src="/douyin-assets/icons/add-light.png" alt="" />
            <img className="yes" src="/douyin-assets/icons/ok-red.png" alt="" />
          </span>
        </div>
        <button type="button" className={liked ? 'active' : ''} aria-pressed={liked} onClick={onToggleLike}>
          <span className="dyVideoReplicaToolbarIcon"><DetailIcon name="heart" /></span>
          <span>{formatCount((item.statistics?.digg_count || 0) + (liked ? 1 : 0), '0')}</span>
        </button>
        <button type="button" onClick={onOpenComments}>
          <span className="dyVideoReplicaToolbarIcon"><DetailIcon name="comment" /></span>
          <span>{formatCount(item.statistics?.comment_count, '评论')}</span>
        </button>
        <button type="button" className={collected ? 'active collect' : 'collect'} aria-pressed={collected} onClick={onToggleCollect}>
          <span className="dyVideoReplicaToolbarIcon"><DetailIcon name="star" /></span>
          <span>{formatCount((item.statistics?.collect_count || 0) + (collected ? 1 : 0), '收藏')}</span>
        </button>
        <button type="button" onClick={onOpenShare}>
          <span className="dyVideoReplicaToolbarIcon"><DetailIcon name="share" /></span>
          <span>{formatCount(item.statistics?.share_count, '分享')}</span>
        </button>
        <a href="/home/music" className="dyVideoReplicaDisc">
          {musicCover && !discFailed ? <img src={musicCover} alt="" referrerPolicy="no-referrer" onError={() => setDiscFailed(true)} /> : <span>♪</span>}
        </a>
      </aside>
      <section className="dyVideoReplicaCopy">
        <b>@{authorName}</b>
        <p>{item.desc || '刷到一条真实抖音素材'}</p>
        <a href="/home/music">♪ {musicText(item)}</a>
      </section>
    </section>
  );
}

export function VideoDetailPage() {
  const [items, setItems] = useState<RawFeedVideo[]>([]);
  const [index, setIndex] = useState(0);
  const [loading, setLoading] = useState(true);
  const [sheet, setSheet] = useState<VideoSheet>(null);
  const [likedIds, setLikedIds] = useState<Set<string>>(() => new Set());
  const [collectedIds, setCollectedIds] = useState<Set<string>>(() => new Set());
  const [followedAuthorIds, setFollowedAuthorIds] = useState<Set<string>>(() => new Set());
  const touchY = useRef(0);

  useEffect(() => {
    let disposed = false;
    const params = new URLSearchParams(window.location.search);
    const awemeId = params.get('awemeId') || '';
    const authorId = params.get('authorId') || '';
    void loadDetailRows(awemeId, authorId)
      .then(({ rows, initialIndex }) => {
        if (disposed) return;
        setItems(rows);
        setIndex(initialIndex);
      })
      .catch(() => {
        if (!disposed) setItems([]);
      })
      .finally(() => {
        if (!disposed) setLoading(false);
      });
    return () => {
      disposed = true;
    };
  }, []);

  const activeItem = items[index];

  function handleTouchStart(event: TouchEvent) {
    if (sheet) return;
    touchY.current = event.touches[0]?.clientY ?? 0;
  }

  function handleTouchEnd(event: TouchEvent) {
    if (sheet || !items.length) return;
    const end = event.changedTouches[0]?.clientY ?? touchY.current;
    const delta = end - touchY.current;
    if (Math.abs(delta) < SWIPE_DISTANCE) return;
    setIndex((current) => {
      if (delta < 0) return Math.min(items.length - 1, current + 1);
      return Math.max(0, current - 1);
    });
  }

  function toggleLike(item: RawFeedVideo) {
    const id = item.aweme_id || item.desc || '';
    if (!id) return;
    setLikedIds((current) => {
      const next = new Set(current);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  }

  function toggleCollect(item: RawFeedVideo) {
    const id = item.aweme_id || item.desc || '';
    if (!id) return;
    setCollectedIds((current) => {
      const next = new Set(current);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  }

  function toggleFollow(item: RawFeedVideo) {
    const id = douyinId(item) || item.aweme_id || '';
    if (!id) return;
    setFollowedAuthorIds((current) => {
      const next = new Set(current);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  }

  return (
    <main className="mobileShell dyVideoReplicaPage" onTouchStart={handleTouchStart} onTouchEnd={handleTouchEnd}>
      <header className="dyVideoReplicaSearch search-wrapper">
        <button type="button" aria-label="返回" onClick={goBack}><BackMiniIcon /></button>
        <a href="/home/search" className="search">
          <div className="left"><SearchMiniIcon /><span>搜你想看的</span></div>
          <div className="right"><span className="gang">|</span><span className="txt">搜索</span></div>
        </a>
      </header>

      {loading ? (
        <section className="dyVideoReplicaLoading"><DouyinLoading /><span>正在加载作品...</span></section>
      ) : items.length ? (
        <section className="dyVideoReplicaTrack" style={{ transform: `translateY(-${index * 100}%)` }}>
          {items.map((item, itemIndex) => {
            const key = item.aweme_id || `${item.desc}-${itemIndex}`;
            return (
              <VideoSlide
                item={item}
                active={itemIndex === index}
                liked={likedIds.has(key)}
                onToggleLike={() => toggleLike(item)}
                collected={collectedIds.has(key)}
                onToggleCollect={() => toggleCollect(item)}
                followed={followedAuthorIds.has(douyinId(item))}
                onToggleFollow={() => toggleFollow(item)}
                onOpenComments={() => setSheet('comments')}
                onOpenShare={() => setSheet('share')}
                key={key}
              />
            );
          })}
        </section>
      ) : (
        <section className="dyVideoReplicaLoading">作品加载失败</section>
      )}

      <footer className="dyVideoReplicaFooter">
        <label><span>我</span><input placeholder="善语结善缘，恶言伤人心" /></label>
        <button type="button">▣</button>
        <button type="button">@</button>
        <button type="button">☺</button>
      </footer>

      {sheet === 'comments' ? (
        <DouyinCommentSheet videoId={activeItem?.aweme_id} onClose={() => setSheet(null)} />
      ) : null}

      {sheet === 'share' ? (
        <section className="dyVideoReplicaSheet" role="dialog" aria-modal="true">
          <header><b>分享到</b><button type="button" onClick={() => setSheet(null)}>×</button></header>
          <div className="dyVideoReplicaShareGrid">
            {['转发', '私信朋友', '微信', '朋友圈', '复制链接', '举报'].map((item) => (
              <a href={item === '私信朋友' ? '/message/share-to-friend' : item === '举报' ? '/home/report' : '/video-detail'} key={item}><span>{item.slice(0, 1)}</span>{item}</a>
            ))}
          </div>
        </section>
      ) : null}
    </main>
  );
}
