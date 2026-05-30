import { useMemo, useState, type CSSProperties, type UIEvent } from 'react';

type MusicRankTab = 'hot' | 'rising' | 'original';

type MusicTrack = {
  id: string;
  name: string;
  author: string;
  duration: string;
  useCount: string;
  colorA: string;
  colorB: string;
};

const RANK_TABS: Array<{ key: MusicRankTab; label: string }> = [
  { key: 'hot', label: '热歌榜' },
  { key: 'rising', label: '飙升榜' },
  { key: 'original', label: '原创榜' },
];

const BASE_TRACKS: MusicTrack[] = [
  { id: 'song-1', name: '龙卷风', author: '周杰伦', duration: '01:39', useCount: '3744.1w', colorA: '#d2b48b', colorB: '#342414' },
  { id: 'song-2', name: '爱在西元前', author: '周杰伦', duration: '01:00', useCount: '3744.1w', colorA: '#7c9bb8', colorB: '#142231' },
  { id: 'song-3', name: '蜗牛', author: '周杰伦', duration: '01:00', useCount: '3744.1w', colorA: '#9db66c', colorB: '#25331a' },
  { id: 'song-4', name: '半岛铁盒', author: '周杰伦', duration: '01:00', useCount: '3744.1w', colorA: '#b66f82', colorB: '#331821' },
  { id: 'song-5', name: '轨迹', author: '周杰伦', duration: '01:00', useCount: '3744.1w', colorA: '#806cc6', colorB: '#201936' },
  { id: 'song-6', name: '七里香', author: '周杰伦', duration: '01:00', useCount: '3744.1w', colorA: '#c79552', colorB: '#352313' },
  { id: 'song-7', name: '发如雪', author: '周杰伦', duration: '01:00', useCount: '3744.1w', colorA: '#b99576', colorB: '#2d201b' },
  { id: 'song-8', name: '霍元甲', author: '周杰伦', duration: '01:00', useCount: '3744.1w', colorA: '#b25b45', colorB: '#321812' },
  { id: 'song-9', name: '千里之外', author: '周杰伦/费玉清', duration: '01:00', useCount: '3744.1w', colorA: '#7098a8', colorB: '#182930' },
  { id: 'song-10', name: '菊花台', author: '周杰伦', duration: '01:00', useCount: '3744.1w', colorA: '#d7bc75', colorB: '#392d16' },
  { id: 'song-11', name: '不能说的秘密', author: '周杰伦', duration: '01:00', useCount: '3744.1w', colorA: '#8a8ca4', colorB: '#20222c' },
  { id: 'song-12', name: '牛仔很忙', author: '周杰伦', duration: '01:00', useCount: '3744.1w', colorA: '#c88958', colorB: '#342114' },
];

function goBack() {
  if (window.history.length > 1) {
    window.history.back();
    return;
  }
  window.location.assign('/home/music');
}

function makeRankTracks(tab: MusicRankTab) {
  if (tab === 'rising') {
    return BASE_TRACKS.map((track, index) => ({
      ...track,
      id: `${track.id}-rising`,
      name: index % 2 === 0 ? `${track.name} DJ版` : `${track.name} 片段`,
      useCount: `${98 - index * 5}.8w`,
    }));
  }
  if (tab === 'original') {
    return BASE_TRACKS.map((track, index) => ({
      ...track,
      id: `${track.id}-original`,
      author: index % 2 === 0 ? '抖音原创音乐人' : '独立创作者',
      useCount: `${68 - index * 3}.2w`,
    }));
  }
  return BASE_TRACKS;
}

function MusicRankPage() {
  const [activeTab, setActiveTab] = useState<MusicRankTab>('hot');
  const [scrollTop, setScrollTop] = useState(0);
  const [playingId, setPlayingId] = useState<string | null>(null);
  const [collectedIds, setCollectedIds] = useState(() => new Set<string>());
  const tracks = useMemo(() => makeRankTracks(activeTab), [activeTab]);
  const fixedOpacity = Math.min(scrollTop / 120, 1);

  function handleScroll(event: UIEvent<HTMLElement>) {
    setScrollTop(event.currentTarget.scrollTop);
  }

  function toggleCollect(id: string) {
    setCollectedIds((current) => {
      const next = new Set(current);
      if (next.has(id)) {
        next.delete(id);
      } else {
        next.add(id);
      }
      return next;
    });
  }

  return (
    <main className="mobileShell dyMusicRankPage" aria-label="抖音音乐榜" onScroll={handleScroll}>
      <button type="button" className="dyMusicRankPageBack" aria-label="返回" onClick={goBack}>
        ‹
      </button>
      <header className="dyMusicRankPageFixedHeader" style={{ opacity: fixedOpacity }}>
        <h1>抖音音乐榜</h1>
      </header>

      <section className="dyMusicRankPageHero">
        <div>
          <b>抖音音乐榜</b>
          <span>DOUYIN MUSIC RANK</span>
        </div>
        <p>更新于：05.28</p>
      </section>

      <nav className="dyMusicRankPageTabs" aria-label="音乐榜类型">
        {RANK_TABS.map((tab) => (
          <button
            type="button"
            className={activeTab === tab.key ? 'active' : ''}
            onClick={() => {
              setActiveTab(tab.key);
              setPlayingId(null);
            }}
            key={tab.key}
          >
            {tab.label}
          </button>
        ))}
      </nav>

      <section className="dyMusicRankPageList">
        {tracks.map((track, index) => {
          const isCollected = collectedIds.has(track.id);
          const isPlaying = playingId === track.id;
          return (
            <article className="dyMusicRankPageItem" key={track.id}>
              <button
                type="button"
                className="dyMusicRankPageItemTop"
                onClick={() => setPlayingId((current) => (current === track.id ? null : track.id))}
              >
                <span className={`dyMusicRankPageRank${index < 3 ? ' top' : ''}`}>{index + 1}</span>
                <i
                  className="dyMusicRankPageCover"
                  style={{ '--dy-rank-cover-a': track.colorA, '--dy-rank-cover-b': track.colorB } as CSSProperties}
                >
                  {isPlaying ? 'Ⅱ' : '▶'}
                </i>
                <span className="dyMusicRankPageTrack">
                  <b>{track.name}</b>
                  <em>{track.author}</em>
                  <small>
                    {track.duration}
                    <i />
                    {track.useCount}人使用
                  </small>
                </span>
              </button>
              <div className="dyMusicRankPageOptions">
                <button
                  type="button"
                  className={isCollected ? 'active' : ''}
                  aria-label={isCollected ? '取消收藏' : '收藏'}
                  onClick={() => toggleCollect(track.id)}
                >
                  ★
                </button>
                <a href={`/home/music?name=${encodeURIComponent(track.name)}&author=${encodeURIComponent(track.author)}&use_count=${encodeURIComponent(track.useCount)}`} aria-label="进入音乐详情">
                  ⋯
                </a>
              </div>
              {isCollected ? (
                <div className="dyMusicRankPageArtist">
                  <span>{track.author.slice(0, 1)}</span>
                  <div>
                    <b>{track.author}</b>
                    <small>粉丝：83.4w</small>
                  </div>
                  <button type="button">关注</button>
                </div>
              ) : null}
            </article>
          );
        })}
      </section>
    </main>
  );
}

export default MusicRankPage;
