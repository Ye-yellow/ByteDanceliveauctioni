import { useEffect, useMemo, useState } from 'react';
import './author-profile-replica.css';

type RawAuthor = {
  uid?: string;
  short_id?: string;
  unique_id?: string;
  nickname?: string;
  signature?: string;
  total_favorited?: number;
  following_count?: number;
  mplatform_followers_count?: number;
  aweme_count?: number;
  gender?: number;
  user_age?: number;
  ip_location?: string;
  province?: string;
  city?: string;
  avatar_168x168?: { url_list?: string[] };
  avatar_300x300?: { url_list?: string[] };
  avatar_thumb?: { url_list?: string[] };
  cover_url?: Array<{ url_list?: string[] }>;
};

type RawFeedVideo = {
  aweme_id?: string;
  desc?: string;
  fixture_category?: string;
  author?: RawAuthor;
  video?: {
    play_addr?: { url_list?: string[] };
    cover?: { url_list?: string[] };
  };
  statistics?: {
    digg_count?: number;
    comment_count?: number;
    collect_count?: number;
    share_count?: number;
  };
};

const FEED_SOURCE = '/data/douyin-feed.json';
const USER_VIDEO_SOURCE = '/data/user-video-list';
const DATA_FETCH_OPTIONS: RequestInit = { cache: 'no-store' };
const CATEGORY_LABELS: Record<string, string> = {
  animals: '萌宠',
  auction: '直播拍卖',
  beauty: '穿搭',
  city: '城市生活',
  culture: '国风',
  food: '美食',
  lifestyle: '生活',
  nature: '自然',
  sport: '运动健身',
  tech: '科技',
  travel: '旅行',
};

function firstUrl(urlList?: string[]) {
  return Array.isArray(urlList) ? urlList.find(Boolean) || '' : '';
}

function avatarUrl(author?: RawAuthor) {
  return firstUrl(author?.avatar_300x300?.url_list) || firstUrl(author?.avatar_168x168?.url_list) || firstUrl(author?.avatar_thumb?.url_list);
}

function authorAvatarUrl(video?: RawFeedVideo) {
  return avatarUrl(video?.author) || authorCoverUrl(video) || posterCoverUrl(video);
}

function authorCoverUrl(video?: RawFeedVideo) {
  return firstUrl(video?.author?.cover_url?.[0]?.url_list) || firstUrl(video?.video?.cover?.url_list);
}

function posterCoverUrl(video?: RawFeedVideo) {
  return firstUrl(video?.video?.cover?.url_list) || firstUrl(video?.author?.cover_url?.[0]?.url_list);
}

function formatCount(value?: number) {
  if (!value || !Number.isFinite(value)) return '0';
  if (value >= 100000000) return `${(value / 100000000).toFixed(value >= 1000000000 ? 0 : 1)}亿`;
  if (value >= 10000) return `${(value / 10000).toFixed(value >= 100000 ? 0 : 1)}万`;
  return value.toLocaleString('zh-CN');
}

function authorKey(video?: RawFeedVideo) {
  const author = video?.author;
  return author?.uid || author?.unique_id || author?.short_id || `${video?.fixture_category || 'author'}:${author?.nickname || video?.aweme_id || ''}`;
}

function douyinId(video?: RawFeedVideo) {
  const author = video?.author;
  return author?.unique_id || author?.short_id || author?.uid || `dy_${video?.aweme_id?.slice(-8) || 'user'}`;
}

function categoryLabel(category?: string) {
  return category ? CATEGORY_LABELS[category] || category : '';
}

function normalizeVideosByAuthor(videos: RawFeedVideo[], current?: RawFeedVideo) {
  const key = authorKey(current);
  if (!key) return [];
  return videos.filter((video) => authorKey(video) === key);
}

function backToHome() {
  if (history.length > 1) history.back();
  else location.assign('/home');
}

function BackIcon() {
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true">
      <path fill="currentColor" d="M15.7 4.7a1 1 0 0 1 0 1.4L9.8 12l5.9 5.9a1 1 0 1 1-1.4 1.4l-6.6-6.6a1 1 0 0 1 0-1.4l6.6-6.6a1 1 0 0 1 1.4 0Z" />
    </svg>
  );
}

function SearchIcon() {
  return (
    <svg viewBox="0 0 512 512" aria-hidden="true">
      <path fill="currentColor" d="M456.69 421.39 362.6 327.3a173.8 173.8 0 0 0 34.84-104.58C397.44 126.38 319.06 48 222.72 48S48 126.38 48 222.72s78.38 174.72 174.72 174.72A173.8 173.8 0 0 0 327.3 362.6l94.09 94.09a25 25 0 0 0 35.3-35.3ZM97.92 222.72a124.8 124.8 0 1 1 124.8 124.8 124.95 124.95 0 0 1-124.8-124.8Z" />
    </svg>
  );
}

function MoreIcon() {
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true">
      <path fill="currentColor" d="M5 12a2 2 0 1 0 .01 0M12 12a2 2 0 1 0 .01 0M19 12a2 2 0 1 0 .01 0" />
    </svg>
  );
}

function DownArrowIcon() {
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true">
      <path fill="currentColor" d="M11.178 19.569a.998.998 0 0 0 1.644 0l9-13A.999.999 0 0 0 21 5H3a1.002 1.002 0 0 0-.822 1.569z" />
    </svg>
  );
}

function PosterGrid({ videos }: { videos: RawFeedVideo[] }) {
  return (
    <section className="dyAuthorVideos" aria-label="作品列表">
      {videos.map((video) => {
        const poster = posterCoverUrl(video);
        const href = `/video-detail?awemeId=${encodeURIComponent(video.aweme_id || '')}&authorId=${encodeURIComponent(douyinId(video))}`;
        return (
          <a href={href} key={video.aweme_id || video.desc}>
            {poster ? <img src={poster} alt="" loading="lazy" referrerPolicy="no-referrer" /> : <span />}
            <small>{formatCount(video.statistics?.digg_count)}</small>
          </a>
        );
      })}
    </section>
  );
}

export function AuthorProfilePage() {
  const [videos, setVideos] = useState<RawFeedVideo[]>([]);
  const [authorVideos, setAuthorVideos] = useState<RawFeedVideo[]>([]);
  const [loading, setLoading] = useState(true);
  const [followed, setFollowed] = useState(false);
  const [showRecommend, setShowRecommend] = useState(false);
  const [navFixed, setNavFixed] = useState(false);
  const awemeId = new URLSearchParams(location.search).get('awemeId') || '';

  useEffect(() => {
    let disposed = false;
    void fetch(FEED_SOURCE, DATA_FETCH_OPTIONS).then((response) => response.json() as Promise<RawFeedVideo[]>).then((rows) => {
      if (!disposed) setVideos(rows);
    }).catch(() => {
      if (!disposed) setVideos([]);
    }).finally(() => {
      if (!disposed) setLoading(false);
    });
    return () => {
      disposed = true;
    };
  }, []);

  const currentVideo = useMemo(() => (
    videos.find((video) => video.aweme_id === awemeId) || videos[0]
  ), [awemeId, videos]);
  const author = currentVideo?.author;
  const authorId = douyinId(currentVideo);
  const works = useMemo(() => normalizeVideosByAuthor(videos, currentVideo), [currentVideo, videos]);
  const posterVideos = authorVideos.length ? authorVideos : works.length ? works : videos.slice(0, 12);
  const name = author?.nickname || '抖音创作者';
  const signature = author?.signature || currentVideo?.desc || '记录生活里的精彩瞬间';
  const areaText = [author?.ip_location, author?.province, author?.city].filter(Boolean).join(' · ');
  const currentAuthorKey = authorKey(currentVideo);
  const currentCategory = categoryLabel(currentVideo?.fixture_category);
  const statItems = [
    { label: '获赞', value: formatCount(author?.total_favorited || currentVideo?.statistics?.digg_count) },
    { label: '关注', value: formatCount(author?.following_count) },
    { label: '粉丝', value: formatCount(author?.mplatform_followers_count) },
  ];
  const recommendedAuthors = useMemo(() => {
    const seen = new Set<string>();
    return videos.filter((video) => {
      const key = authorKey(video);
      if (!key || key === currentAuthorKey || seen.has(key)) return false;
      seen.add(key);
      return true;
    }).slice(0, 4);
  }, [currentAuthorKey, videos]);

  useEffect(() => {
    if (!authorId) return;
    let disposed = false;
    void fetch(`${USER_VIDEO_SOURCE}/user-${encodeURIComponent(authorId)}.json`, DATA_FETCH_OPTIONS)
      .then((response) => {
        if (!response.ok) throw new Error(`load author videos failed: ${response.status}`);
        return response.json() as Promise<RawFeedVideo[]>;
      })
      .then((rows) => {
        if (!disposed) setAuthorVideos(Array.isArray(rows) ? rows : []);
      })
      .catch(() => {
        if (!disposed) setAuthorVideos([]);
      });
    return () => {
      disposed = true;
    };
  }, [authorId]);

  return (
    <main
      className="mobileShell dyAuthorProfile"
      onScroll={(event) => setNavFixed(event.currentTarget.scrollTop > 158)}
    >
      <nav className={navFixed ? 'dyAuthorFloat fixed' : 'dyAuthorFloat'} aria-label="发布者主页工具栏">
        <div className="dyAuthorFloatLeft">
          <button type="button" className="dyAuthorNavIcon" aria-label="返回" onClick={backToHome}><BackIcon /></button>
          {navFixed ? (
          <button type="button" className="dyAuthorFloatUser" onClick={() => setFollowed((value) => !value)}>
              {authorAvatarUrl(currentVideo) ? <img src={authorAvatarUrl(currentVideo)} alt="" referrerPolicy="no-referrer" /> : null}
              <span>{followed ? '私信' : '关注'}</span>
            </button>
          ) : null}
        </div>
        <div className="dyAuthorFloatActions">
          {!navFixed && followed ? <a href="/me/request-update" className="dyAuthorRequest">求更新</a> : null}
          <a href="/home/search" className="dyAuthorNavIcon" aria-label="搜索"><SearchIcon /></a>
          <button type="button" className="dyAuthorNavIcon" aria-label="更多"><MoreIcon /></button>
        </div>
      </nav>

      {loading ? <section className="dyAuthorLoading">正在加载主页...</section> : (
        <>
          <header className="dyAuthorCover">
            {authorCoverUrl(currentVideo) ? <img src={authorCoverUrl(currentVideo)} alt="" referrerPolicy="no-referrer" /> : null}
            <div className="dyAuthorShade" />
            <section className="dyAuthorIdentity">
              <span className="dyAuthorAvatar">
                {authorAvatarUrl(currentVideo) ? <img src={authorAvatarUrl(currentVideo)} alt="" referrerPolicy="no-referrer" /> : name.slice(0, 1)}
              </span>
              <div>
                <h1>{name}</h1>
                <p>抖音号：{douyinId(currentVideo)} <button type="button" aria-label="复制抖音号">▣</button></p>
              </div>
            </section>
          </header>

          <section className="dyAuthorBody">
            <nav className="dyAuthorStats" aria-label="账号数据">
              {statItems.map((item) => (
                <span key={item.label}><b>{item.value}</b><small>{item.label}</small></span>
              ))}
            </nav>

            <p className="dyAuthorSignature">{signature}</p>
            <div className="dyAuthorTags">
              {typeof author?.user_age === 'number' && author.user_age > 0 ? <span>{author.user_age}岁</span> : null}
              {areaText ? <span>{areaText}</span> : null}
              {currentCategory ? <span>{currentCategory}</span> : null}
            </div>

            <section className="dyAuthorCards" aria-label="发布者卡片">
              <a href="/home/music"><b>TA 的音乐</b><small>视频原声和收藏</small></a>
              <a href="/shop"><b>商品橱窗</b><small>精选好物与同款</small></a>
            </section>

            <div className="dyAuthorActions">
              <button type="button" className={followed ? 'followed' : ''} onClick={() => setFollowed((value) => !value)}>
                {followed ? '已关注' : '+ 关注'}
              </button>
              <a href="/message/chat">私信</a>
              <button
                type="button"
                className={showRecommend ? 'dyAuthorRecommendToggle expanded' : 'dyAuthorRecommendToggle'}
                aria-label={showRecommend ? '收起推荐' : '展开推荐'}
                onClick={() => setShowRecommend((value) => !value)}
              >
                <DownArrowIcon />
              </button>
            </div>

            <section className={showRecommend ? 'dyAuthorRecommend' : 'dyAuthorRecommend hidden'} aria-label="你可能感兴趣">
              <header><span>你可能感兴趣</span><i>ⓘ</i></header>
              <div>
                {recommendedAuthors.map((video) => (
                  <a href={`/user?awemeId=${encodeURIComponent(video.aweme_id || '')}`} key={video.aweme_id}>
                    <span>{authorAvatarUrl(video) ? <img src={authorAvatarUrl(video)} alt="" referrerPolicy="no-referrer" /> : (video.author?.nickname || '抖').slice(0, 1)}</span>
                    <b>{video.author?.nickname || '抖音创作者'}</b>
                    <small>可能感兴趣的人</small>
                  </a>
                ))}
              </div>
            </section>

            <nav className="dyAuthorTabs" aria-label="作品分类">
              <button type="button" className="active">作品 {authorVideos.length || author?.aweme_count || works.length}</button>
              <button type="button">喜欢</button>
            </nav>
            <PosterGrid videos={posterVideos} />
          </section>
        </>
      )}
    </main>
  );
}
