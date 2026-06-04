import { useMemo, useState } from 'react';
import { consultBuyer } from '../features/auction/api/auctionApi';
import { formatMoney } from '../shared/lib/money';
import type { AIBuyerConsultReply } from '../shared/api/types';

type SearchGuess = {
  name: string;
  mark?: 'new' | 'hot';
};

type SearchRankKey = 'hot' | 'live' | 'deal' | 'category';

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
  { title: '老银鎏金手作镯', meta: '3.8w人看过', mark: 'hot' },
  { title: '冰种翡翠平安扣', meta: '2.6w人看过' },
  { title: '紫砂壶名家手作款', meta: '1.9w人看过', mark: 'new' },
  { title: '和田玉小籽料吊坠', meta: '1.6w人看过' },
  { title: '复古珐琅胸针套装', meta: '1.2w人看过' },
  { title: '朱砂醒狮手串', meta: '9800人看过', mark: 'hot' },
  { title: '宋式汝窑茶盏', meta: '8600人看过' },
  { title: '蜜蜡圆珠单圈', meta: '7300人看过' },
  { title: '孤品老相机收藏', meta: '6900人看过' },
  { title: '小叶紫檀无事牌', meta: '6200人看过' },
];

const LIVE_RANKS: SearchRankRow[] = [
  { title: '银饰手镯专场正在竞拍', meta: '竞拍中 · 12口', mark: 'hot', href: '/home' },
  { title: '翡翠小件今晚加拍', meta: '竞拍中 · 9口', href: '/home' },
  { title: '茶器孤品直播间', meta: '加时中 · 8口', mark: 'pk', href: '/home' },
  { title: '文玩手串低价开拍', meta: '竞拍中 · 6口', href: '/home' },
  { title: '复古饰品清仓场', meta: '待开拍 · 5件', mark: 'new', href: '/home' },
  { title: '玉石吊坠精选', meta: '竞拍中 · 5口', href: '/home' },
  { title: '老物件收藏小场', meta: '待开拍 · 4件', href: '/home' },
  { title: '茶盏茶宠组合场', meta: '竞拍中 · 3口', href: '/home' },
  { title: '礼物预算友好场', meta: '待开拍 · 3件', href: '/home' },
];

const DEAL_RANKS: SearchRankRow[] = [
  { title: '老银镯 15 分钟落槌', meta: '¥1280 成交' },
  { title: '蜜蜡单圈被秒拍', meta: '¥860 成交', mark: 'hot' },
  { title: '紫砂壶尾段反超', meta: '¥2380 成交' },
  { title: '和田玉吊坠加时成交', meta: '¥1680 成交', mark: 'new' },
  { title: '珐琅胸针组合拍出', meta: '¥520 成交' },
  { title: '汝窑茶盏两人争拍', meta: '¥760 成交' },
  { title: '朱砂手串成功落槌', meta: '¥390 成交' },
  { title: '老相机收藏场收官', meta: '¥1890 成交' },
];

const CATEGORY_RANKS: Record<string, SearchRankRow[]> = {
  饰品: [
    { title: '银饰手镯', meta: '热搜 1.3w' },
    { title: '复古胸针', meta: '热搜 9200' },
    { title: '珍珠耳饰', meta: '热搜 8800' },
    { title: '蜜蜡手串', meta: '热搜 7600' },
    { title: '朱砂手绳', meta: '热搜 6900' },
  ],
  文玩: [
    { title: '和田玉吊坠', meta: '热搜 1.1w' },
    { title: '紫砂壶', meta: '热搜 9800' },
    { title: '汝窑茶盏', meta: '热搜 8600' },
    { title: '小叶紫檀', meta: '热搜 7900' },
    { title: '老物件收藏', meta: '热搜 7200' },
  ],
  数码: [
    { title: '复古相机', meta: '热搜 9300' },
    { title: '机械键盘', meta: '热搜 8700' },
    { title: '黑胶唱机', meta: '热搜 7600' },
    { title: '掌机收藏', meta: '热搜 6800' },
    { title: '老镜头', meta: '热搜 6100' },
  ],
};

const RANK_TABS: Array<{ key: SearchRankKey; label: string }> = [
  { key: 'hot', label: '拍品热榜' },
  { key: 'live', label: '直播拍品' },
  { key: 'deal', label: '成交榜' },
  { key: 'category', label: '品类榜' },
];

function goBack() {
  if (window.history.length > 1) {
    window.history.back();
    return;
  }
  window.location.assign('/home');
}

function formatLotStatus(status: string) {
  switch (status) {
    case 'LOT_STATUS_QUEUED':
      return '待开拍';
    case 'LOT_STATUS_LIVE':
      return '竞拍中';
    case 'LOT_STATUS_EXTENDED':
      return '加时中';
    case 'LOT_STATUS_SOLD':
      return '已成交';
    case 'LOT_STATUS_PASSED':
      return '已流拍';
    case 'LOT_STATUS_CANCELLED':
      return '已取消';
    default:
      return '拍品';
  }
}

function SearchPage() {
  const [history, setHistory] = useState(SEARCH_HISTORY);
  const [expanded, setExpanded] = useState(false);
  const [guessRound, setGuessRound] = useState(0);
  const [activeRank, setActiveRank] = useState<SearchRankKey>('hot');
  const [activeCategory, setActiveCategory] = useState(Object.keys(CATEGORY_RANKS)[0]);
  const [query, setQuery] = useState('');
  const [aiQuery, setAIQuery] = useState('');
  const [aiReply, setAIReply] = useState<AIBuyerConsultReply | null>(null);
  const [aiLoading, setAILoading] = useState(false);
  const [aiError, setAIError] = useState('');

  const visibleHistory = expanded ? history : history.slice(0, 2);
  const guesses = useMemo(() => {
    const offset = (guessRound * 6) % SEARCH_GUESSES.length;
    return SEARCH_GUESSES.slice(offset).concat(SEARCH_GUESSES.slice(0, offset)).slice(0, 8);
  }, [guessRound]);
  const rankRows = useMemo(() => {
    if (activeRank === 'live') return LIVE_RANKS;
    if (activeRank === 'deal') return DEAL_RANKS;
    if (activeRank === 'category') return CATEGORY_RANKS[activeCategory] || [];
    return HOT_RANKS;
  }, [activeCategory, activeRank]);
  const runPlainSearch = () => {
    const text = query.trim();
    if (!text) return;
    setHistory((current) => [text, ...current.filter((item) => item !== text)].slice(0, 10));
  };
  const runAIConsult = async (nextQuery = aiQuery) => {
    const text = nextQuery.trim();
    if (!text) {
      setAIError('请输入想找的拍品、预算或用途');
      return;
    }
    setAILoading(true);
    setAIError('');
    setAIQuery(text);
    try {
      const reply = await consultBuyer({ query: text });
      setAIReply(reply);
      setHistory((current) => [text, ...current.filter((item) => item !== text)].slice(0, 10));
    } catch (error) {
      setAIError(error instanceof Error ? error.message : '找拍品服务暂时不可用');
    } finally {
      setAILoading(false);
    }
  };

  return (
    <main className="mobileShell dySearchPage" aria-label="抖音搜索">
      <header className="dySearchPageHeader">
        <button type="button" aria-label="返回" onClick={goBack}>
          ‹
        </button>
        <label>
          <span aria-hidden="true">⌕</span>
          <input
            autoFocus
            value={query}
            placeholder="描述你想找的拍品"
            onChange={(event) => setQuery(event.target.value)}
            onKeyDown={(event) => {
              if (event.key === 'Enter') runPlainSearch();
            }}
          />
        </label>
        <button type="button" onClick={runPlainSearch}>搜索</button>
      </header>

      <div className="dySearchPageContent">
        <section className="dySearchAI" aria-label="找好拍">
          <form
            className="dySearchAIBox"
            onSubmit={(event) => {
              event.preventDefault();
              void runAIConsult();
            }}
          >
            <span>找好拍</span>
            <label>
              <i aria-hidden="true">⌕</i>
              <input
                value={aiQuery}
                placeholder="输入预算、品类或用途"
                onChange={(event) => setAIQuery(event.target.value)}
              />
            </label>
            <button type="submit" disabled={aiLoading}>
              {aiLoading ? '查找中' : '找一下'}
            </button>
          </form>
          {aiError ? <p className="dySearchAIError">{aiError}</p> : null}
          {aiReply ? (
            <div className="dySearchAIReply">
              <p>{aiReply.answer}</p>
              {aiReply.fallbackUsed ? <small>为你推荐</small> : null}
              {aiReply.results.length ? (
                <div className="dySearchAIResults">
                  {aiReply.results.map((item) => (
                    <a href={item.href || `/m/room/${item.roomId}`} key={`${item.roomId}-${item.lotId}`} className="dySearchAIResultCard">
                      {item.imageUrl ? (
                        <img className="dySearchAIResultImage" src={item.imageUrl} alt={item.title} loading="lazy" />
                      ) : (
                        <span className="dySearchAIResultImage dySearchAIResultImageEmpty" aria-hidden="true">拍</span>
                      )}
                      <div className="dySearchAIResultInfo">
                        <span>{formatLotStatus(item.status)}</span>
                        <b>{item.title}</b>
                        <strong><em>起拍价</em>{formatMoney(item.currentPrice)}</strong>
                        <small>{item.reason}</small>
                      </div>
                    </a>
                  ))}
                </div>
              ) : (
                <div className="dySearchAIEmpty">暂时没有匹配到公开竞拍中的拍品，可以换个品类或预算试试。</div>
              )}
            </div>
          ) : (
            <div className="dySearchAIPrompts">
              {['翡翠手镯', '适合送礼的收藏品', '正在竞拍的拍品'].map((item) => (
                <button type="button" key={item} onClick={() => void runAIConsult(item)}>
                  {item}
                </button>
              ))}
            </div>
          )}
        </section>

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

          {activeRank === 'category' ? (
            <div className="dySearchPageBrandTabs" aria-label="品类分类">
              {Object.keys(CATEGORY_RANKS).map((category) => (
                <button
                  type="button"
                  className={activeCategory === category ? 'active' : ''}
                  onClick={() => setActiveCategory(category)}
                  key={category}
                >
                  {category}
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
          <a href="/home/search" className="dySearchPageMore">
            查看完整{RANK_TABS.find((tab) => tab.key === activeRank)?.label} &gt;
          </a>
        </section>
      </div>
    </main>
  );
}

export default SearchPage;
