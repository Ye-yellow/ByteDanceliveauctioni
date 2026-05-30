import { useMemo, useState, type CSSProperties, type UIEvent } from 'react';

type MusicVideo = {
  id: string;
  title: string;
  author: string;
  colorA: string;
  colorB: string;
};

const DEFAULT_MUSIC = {
  name: '发如雪',
  author: '周杰伦',
  useCount: '3744.1w',
  colorA: '#b99a73',
  colorB: '#2f211c',
};

const MUSIC_POSTERS: MusicVideo[] = [
  { id: 'mv-1', title: '风就应该自由', author: '猫猫喜欢唱', colorA: '#e2bb83', colorB: '#46321f' },
  { id: 'mv-2', title: '暗色街景转场', author: '摄影日记', colorA: '#6d8bb8', colorB: '#101b2d' },
  { id: 'mv-3', title: '把雨拍成电影', author: '城市漫游', colorA: '#7a91a1', colorB: '#1b252a' },
  { id: 'mv-4', title: '旧唱片和新生活', author: '小声哼唱', colorA: '#b76f70', colorB: '#342022' },
  { id: 'mv-5', title: '夜晚便利店', author: '阿森', colorA: '#c9b463', colorB: '#292316' },
  { id: 'mv-6', title: '今天的拍场原声', author: 'LiveAuction', colorA: '#64b5a7', colorB: '#17342f' },
  { id: 'mv-7', title: '低速慢镜头', author: '慢热片段', colorA: '#c184b6', colorB: '#301e33' },
  { id: 'mv-8', title: '落日之后', author: '晚风', colorA: '#e17f5c', colorB: '#2f1d19' },
  { id: 'mv-9', title: '镜头前的小宇宙', author: '拍同款', colorA: '#8b98d9', colorB: '#1c2140' },
];

function goBack() {
  if (window.history.length > 1) {
    window.history.back();
    return;
  }
  window.location.assign('/home');
}

function readMusicFromUrl() {
  const params = new URLSearchParams(window.location.search);
  return {
    name: params.get('name') || DEFAULT_MUSIC.name,
    author: params.get('author') || DEFAULT_MUSIC.author,
    useCount: params.get('use_count') || DEFAULT_MUSIC.useCount,
    colorA: DEFAULT_MUSIC.colorA,
    colorB: DEFAULT_MUSIC.colorB,
  };
}

function MusicPage() {
  const music = useMemo(() => readMusicFromUrl(), []);
  const [fixed, setFixed] = useState(false);
  const [collected, setCollected] = useState(false);
  const [playing, setPlaying] = useState(false);
  const [sharing, setSharing] = useState(false);

  function handleScroll(event: UIEvent<HTMLElement>) {
    setFixed(event.currentTarget.scrollTop > 132);
  }

  return (
    <main className="mobileShell dyMusicPage" aria-label="抖音音乐详情">
      <header className={`dyMusicPageHeader${fixed ? ' isFixed' : ''}`}>
        <button type="button" aria-label="返回" onClick={goBack}>
          ‹
        </button>
        <div className="dyMusicPageHeaderTitle" aria-hidden={!fixed}>
          {fixed ? music.name : ''}
        </div>
        <div className="dyMusicPageHeaderActions">
          {!fixed ? <a href="/home/music-rank-list">抖音音乐榜</a> : null}
          {fixed ? (
            <button
              type="button"
              aria-label={collected ? '取消收藏' : '收藏'}
              className={collected ? 'active' : ''}
              onClick={() => setCollected((value) => !value)}
            >
              ★
            </button>
          ) : null}
          <button type="button" aria-label="分享" onClick={() => setSharing(true)}>
            ↗
          </button>
        </div>
      </header>

      <section className="dyMusicPageScroll" onScroll={handleScroll}>
        <section className="dyMusicPageHero">
          <button
            type="button"
            className="dyMusicPageCover"
            style={{ '--dy-music-cover-a': music.colorA, '--dy-music-cover-b': music.colorB } as CSSProperties}
            aria-label={playing ? '暂停音乐' : '播放音乐'}
            onClick={() => setPlaying((value) => !value)}
          >
            <span aria-hidden="true">{playing ? 'Ⅱ' : '▶'}</span>
          </button>
          <div className="dyMusicPageInfo">
            <h1>{music.name}</h1>
            <div>
              <p>{music.author}</p>
              <p>{music.useCount} 人使用</p>
            </div>
            <button
              type="button"
              className={collected ? 'active' : ''}
              onClick={() => setCollected((value) => !value)}
            >
              <span aria-hidden="true">★</span>
              {collected ? '已收藏' : '收藏'}
            </button>
          </div>
        </section>

        <section className="dyMusicPagePosterGrid" aria-label="使用该音乐的视频">
          {MUSIC_POSTERS.map((poster, index) => (
            <a href="/video-detail" className="dyMusicPagePoster" key={poster.id}>
              <span
                style={{ '--dy-poster-a': poster.colorA, '--dy-poster-b': poster.colorB } as CSSProperties}
                aria-hidden="true"
              >
                <i>{index + 1}</i>
              </span>
              <b>{poster.title}</b>
              <small>@{poster.author}</small>
            </a>
          ))}
        </section>
        <p className="dyMusicPageNoMore">暂时没有更多了</p>
      </section>

      <footer className="dyMusicPageActions">
        <a href="/publish">
          <span aria-hidden="true">♪</span>
          分享到日常
        </a>
        <a href="/publish" className="primary">
          <span aria-hidden="true">●</span>
          拍同款
        </a>
      </footer>

      {sharing ? (
        <section className="dyMusicPageShareMask" role="dialog" aria-modal="true" aria-label="分享音乐">
          <button type="button" aria-label="关闭分享" onClick={() => setSharing(false)} />
          <div className="dyMusicPageShareSheet">
            <header>
              <b>分享到</b>
              <button type="button" onClick={() => setSharing(false)}>
                ×
              </button>
            </header>
            <div>
              {['私信朋友', '微信', '朋友圈', 'QQ', '微博', '复制链接'].map((item) => (
                <button type="button" key={item}>
                  <span>{item.slice(0, 1)}</span>
                  {item}
                </button>
              ))}
            </div>
          </div>
        </section>
      ) : null}
    </main>
  );
}

export default MusicPage;
