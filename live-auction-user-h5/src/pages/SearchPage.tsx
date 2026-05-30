import { useMemo, useState } from 'react';

type SearchGuess = {
  name: string;
  mark?: 'new' | 'hot';
};

type SearchRankKey = 'hot' | 'live' | 'music' | 'brand';

type SearchRankRow = {
  title: string;
  meta: string;
  mark?: 'new' | 'hot' | 'pk' | 'redpack';
  href?: string;
};

const SEARCH_HISTORY = [
  '少年透明人',
  '今日直播热榜',
  '同城好物',
  '发如雪翻唱',
  '透明人挑战',
  '直播间成交',
  '新手机发布',
  '周杰伦音乐榜',
  '拍卖专场',
  '商城爆款',
];

const SEARCH_GUESSES: SearchGuess[] = [
  { name: '少年透明人' },
  { name: '花呗分批次接入征信' },
  { name: '新娘婚礼上跪求悔婚' },
  { name: '当你想换 iPhone 时', mark: 'new' },
  { name: 'Ling OS 灵犀系统' },
  { name: '桑塔纳 2026 款' },
  { name: '透明人' },
  { name: '恒大集团凌晨发公告', mark: 'hot' },
  { name: '日产 GT-R 新款', mark: 'new' },
  { name: '四川双一流大学名单' },
  { name: '一公司放假通知走红' },
  { name: '成都新全优教育倒闭' },
  { name: '当代女生社交现状' },
  { name: '直播竞拍规则' },
];

const HOT_RANKS: SearchRankRow[] = [
  { title: '专题：嘻嘻嘻哈哈瞄瞄嘻嘻嘻', meta: '专题' },
  { title: '国内手机厂商最大的软肋就是系统体验', meta: '999w', mark: 'hot' },
  { title: '大家的官网订单现在什么状态', meta: '999w' },
  { title: '库克不愧是供应链管理大师', meta: '999w' },
  { title: '找到了系统被怀疑窃听的可能原因', meta: '999w', mark: 'new' },
  { title: 'rebase 还是 merge？', meta: '999w' },
  { title: '十一出游西安，本地人能给些建议吗', meta: '999w', mark: 'hot' },
  { title: '为什么要抢购新手机呢？', meta: '999w' },
  { title: '百度输入法 VS 搜狗输入法', meta: '999w' },
  { title: '现在有推荐的同步盘么？', meta: '999w' },
];

const LIVE_RANKS: SearchRankRow[] = [
  { title: '毛三岁（收女徒弟）', meta: '999w 人气', mark: 'pk', href: '/home' },
  { title: '广州表哥', meta: '999w 人气', href: '/home' },
  { title: '一只扬儿', meta: '999w 人气', href: '/home' },
  { title: '沈酒', meta: '999w 人气', href: '/home' },
  { title: '客家婷子', meta: '999w 人气', mark: 'redpack', href: '/home' },
  { title: '三斤（9237）', meta: '999w 人气', href: '/home' },
  { title: '虎哥说车', meta: '999w 人气', href: '/home' },
  { title: '爆笑三江锅', meta: '999w 人气', href: '/home' },
  { title: '罗永浩', meta: '999w 人气', mark: 'redpack', href: '/home' },
];

const MUSIC_RANKS: SearchRankRow[] = [
  { title: '龙卷风', meta: '3744.1w', href: '/home/music-rank-list' },
  { title: '爱在西元前', meta: '3744.1w', href: '/home/music-rank-list' },
  { title: '蜗牛', meta: '3744.1w', href: '/home/music-rank-list' },
  { title: '半岛铁盒', meta: '3744.1w', href: '/home/music-rank-list' },
  { title: '轨迹', meta: '3744.1w', href: '/home/music-rank-list' },
  { title: '七里香', meta: '3744.1w', href: '/home/music-rank-list' },
  { title: '发如雪', meta: '3744.1w', href: '/home/music-rank-list' },
  { title: '霍元甲', meta: '3744.1w', href: '/home/music-rank-list' },
];

const BRAND_RANKS: Record<string, SearchRankRow[]> = {
  汽车: [
    { title: '五菱汽车', meta: '1395w' },
    { title: '宝马', meta: '1395w' },
    { title: '吉利汽车', meta: '1395w' },
    { title: '一汽大众-奥迪', meta: '1395w' },
    { title: '一汽-大众', meta: '1395w' },
  ],
  手机: [
    { title: '华为', meta: '1395w' },
    { title: '小米', meta: '1395w' },
    { title: 'vivo', meta: '1395w' },
    { title: 'oppo', meta: '1395w' },
    { title: '三星', meta: '1395w' },
  ],
  美妆: [
    { title: '巴黎欧莱雅', meta: '1395w' },
    { title: '兰蔻', meta: '1395w' },
    { title: '雅诗兰黛', meta: '1395w' },
    { title: '花西子', meta: '1395w' },
    { title: '完美日记', meta: '1395w' },
  ],
};

const RANK_TABS: Array<{ key: SearchRankKey; label: string }> = [
  { key: 'hot', label: '抖音热榜' },
  { key: 'live', label: '直播榜' },
  { key: 'music', label: '音乐榜' },
  { key: 'brand', label: '品牌榜' },
];

function goBack() {
  if (window.history.length > 1) {
    window.history.back();
    return;
  }
  window.location.assign('/home');
}

function SearchPage() {
  const [history, setHistory] = useState(SEARCH_HISTORY);
  const [expanded, setExpanded] = useState(false);
  const [guessRound, setGuessRound] = useState(0);
  const [activeRank, setActiveRank] = useState<SearchRankKey>('hot');
  const [activeBrand, setActiveBrand] = useState(Object.keys(BRAND_RANKS)[0]);

  const visibleHistory = expanded ? history : history.slice(0, 2);
  const guesses = useMemo(() => {
    const offset = (guessRound * 6) % SEARCH_GUESSES.length;
    return SEARCH_GUESSES.slice(offset).concat(SEARCH_GUESSES.slice(0, offset)).slice(0, 8);
  }, [guessRound]);
  const rankRows = useMemo(() => {
    if (activeRank === 'live') return LIVE_RANKS;
    if (activeRank === 'music') return MUSIC_RANKS;
    if (activeRank === 'brand') return BRAND_RANKS[activeBrand] || [];
    return HOT_RANKS;
  }, [activeBrand, activeRank]);

  return (
    <main className="mobileShell dySearchPage" aria-label="抖音搜索">
      <header className="dySearchPageHeader">
        <button type="button" aria-label="返回" onClick={goBack}>
          ‹
        </button>
        <label>
          <span aria-hidden="true">⌕</span>
          <input autoFocus placeholder="搜索用户名字/抖音号" />
        </label>
        <button type="button">搜索</button>
      </header>

      <div className="dySearchPageContent">
        <section className="dySearchPageHistory" aria-label="搜索历史">
          {visibleHistory.map((item) => (
            <div className="dySearchPageHistoryRow" key={item}>
              <button type="button" className="dySearchPageHistoryText">
                <span aria-hidden="true">◷</span>
                <b>{item}</b>
              </button>
              <button
                type="button"
                aria-label={`删除 ${item}`}
                onClick={() => setHistory((current) => current.filter((entry) => entry !== item))}
              >
                ×
              </button>
            </div>
          ))}
          {history.length > 2 ? (
            <button
              type="button"
              className="dySearchPageExpand"
              onClick={() => {
                if (expanded) {
                  setHistory([]);
                  setExpanded(false);
                  return;
                }
                setExpanded(true);
              }}
            >
              {expanded ? '清除全部搜索记录' : '展开全部'}
            </button>
          ) : null}
        </section>

        <section className="dySearchPageGuess" aria-label="猜你想搜">
          <header>
            <h2>猜你想搜</h2>
            <button type="button" onClick={() => setGuessRound((round) => round + 1)}>
              <span aria-hidden="true">↻</span>
              换一换
            </button>
          </header>
          <div className="dySearchPageGuessGrid">
            {guesses.map((item) => (
              <a href="/home/search" key={item.name}>
                <span>{item.name}</span>
                {item.mark ? <em>{item.mark === 'new' ? '新' : '热'}</em> : null}
              </a>
            ))}
          </div>
        </section>

        <section className="dySearchPageRank" aria-label="榜单">
          <nav aria-label="榜单类型">
            {RANK_TABS.map((tab) => (
              <button
                type="button"
                className={activeRank === tab.key ? 'active' : ''}
                onClick={() => setActiveRank(tab.key)}
                key={tab.key}
              >
                {tab.label}
              </button>
            ))}
          </nav>

          {activeRank === 'brand' ? (
            <div className="dySearchPageBrandTabs" aria-label="品牌分类">
              {Object.keys(BRAND_RANKS).map((brand) => (
                <button
                  type="button"
                  className={activeBrand === brand ? 'active' : ''}
                  onClick={() => setActiveBrand(brand)}
                  key={brand}
                >
                  {brand}
                </button>
              ))}
            </div>
          ) : null}

          <div className="dySearchPageRankList">
            {rankRows.map((item, index) => (
              <a href={item.href || '/home/search'} className="dySearchPageRankRow" key={item.title}>
                <span className={index < 3 ? 'top' : ''}>{index + 1}</span>
                <div>
                  <b>{item.title}</b>
                  {item.mark ? <em>{item.mark === 'new' ? '新' : item.mark === 'hot' ? '热' : item.mark === 'pk' ? 'PK' : '红包'}</em> : null}
                </div>
                <small>{item.meta}</small>
              </a>
            ))}
          </div>
          <a href={activeRank === 'music' ? '/home/music-rank-list' : '/home/search'} className="dySearchPageMore">
            查看完整{RANK_TABS.find((tab) => tab.key === activeRank)?.label} &gt;
          </a>
        </section>
      </div>
    </main>
  );
}

export default SearchPage;
