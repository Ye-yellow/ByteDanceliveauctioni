import { useEffect, useMemo, useRef, useState } from 'react';
import { ChevronDown, ChevronLeft, Clock3, Flame, RefreshCw, Search, Sparkles, Target, X } from 'lucide-react';
import { consultBuyer, listBuyerSuggestions, listPublicRooms } from '../features/auction/api/auctionApi';
import { clearSearchAIState, readSearchAIStateForRestore, saveSearchAIStateForRoomReturn } from '../features/search/model/searchAIState';
import { formatMoney } from '../shared/lib/money';
import type { AIBuyerConsultReply, AIBuyerResult } from '../shared/api/types';

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

const SEARCH_HISTORY_STORAGE_KEY = 'live-auction-h5.search.history.v1';

const AI_PROMPTS = [
  '翡翠手镯',
  '玛瑙手镯',
  '玉石吊坠',
  '项链',
  '适合送礼的收藏品',
  '正在竞拍的拍品',
  '预算500以内',
  '预算1000以内',
  '低价开拍',
  '即将开拍',
  '新手可拍',
  '有实拍图',
  '送妈妈礼物',
  '百元低价',
  '今晚可拍',
  '翡翠玉石',
];

function randomPrompts(prompts: string[]) {
  return [...prompts].sort(() => Math.random() - 0.5).slice(0, 6);
}

function nextPromptBatch(prompts: string[], current: string[] = []) {
  const unique = Array.from(new Set(prompts.map((item) => item.trim()).filter(Boolean)));
  const fresh = unique.filter((item) => !current.includes(item));
  const picked = randomPrompts(fresh.length >= 6 ? fresh : unique);
  if (picked.length === current.length && picked.every((item, index) => item === current[index]) && unique.length > 6) {
    return unique.slice(6).concat(unique.slice(0, 6)).slice(0, 6);
  }
  return picked;
}

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
  clearSearchAIState();
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

function storageAvailable() {
  return typeof window !== 'undefined' && Boolean(window.localStorage) && Boolean(window.sessionStorage);
}

function readSearchHistory() {
  if (!storageAvailable()) return SEARCH_HISTORY;
  try {
    const parsed = JSON.parse(window.localStorage.getItem(SEARCH_HISTORY_STORAGE_KEY) || '[]');
    if (!Array.isArray(parsed)) return SEARCH_HISTORY;
    const values = parsed.filter((item): item is string => typeof item === 'string' && item.trim() !== '');
    return values.length ? values.slice(0, 10) : SEARCH_HISTORY;
  } catch {
    return SEARCH_HISTORY;
  }
}

function priceLabel(status: string) {
  return status === 'LOT_STATUS_LIVE' || status === 'LOT_STATUS_EXTENDED' ? '当前价' : '起拍价';
}

function reasonTags(reason: string) {
  const normalized = reason.replace(/^命中[:：]/, '命中：');
  return normalized
    .split(/[·、]/)
    .map((item) => item.trim())
    .filter(Boolean)
    .slice(0, 4);
}

function displayReason(item: AIBuyerResult) {
  if (/标题命中|命中[:：]/.test(item.reason)) {
    return `${formatLotStatus(item.status)}，可以进直播间看实物`;
  }
  return item.reason;
}

function displayTags(item: AIBuyerResult) {
  const tags: string[] = [];
  const status = formatLotStatus(item.status);
  if (status && status !== '拍品') tags.push(status);
  if (/手镯|吊坠|项链|首饰|翡翠|玛瑙|玉/.test(item.title)) tags.push('适合送礼');
  reasonTags(item.reason)
    .filter((tag) => !/标题命中|命中[:：]/.test(tag))
    .forEach((tag) => tags.push(tag));
  return Array.from(new Set(tags)).slice(0, 2);
}

function emptyCopy(query: string) {
  if (/预算|\d+\s*(元|块|以内|以下|内)/.test(query)) {
    return '暂时没找到这个预算内的公开拍品，可以放宽预算或看看低价开拍。';
  }
  if (query.includes('直播间')) {
    return '暂时没找到这个直播间的公开拍品，可以确认名称，或看看正在竞拍的拍品。';
  }
  if (/送礼|礼物|收藏/.test(query)) {
    return '暂时没找到特别匹配送礼场景的拍品，可以看看首饰、玉石或低价开拍。';
  }
  return '暂时没有匹配到公开竞拍中的拍品，可以换个品类或预算试试。';
}

function emptyPrompts(query: string) {
  if (/预算|\d+\s*(元|块|以内|以下|内)/.test(query)) return ['低价开拍', '预算1000以内', '正在竞拍的拍品'];
  if (query.includes('直播间')) return ['正在竞拍的拍品', '即将开拍', '翡翠玉石'];
  if (/送礼|礼物|收藏/.test(query)) return ['翡翠玉石', '手镯吊坠', '预算500以内'];
  return ['正在竞拍的拍品', '翡翠手镯', '适合送礼的收藏品'];
}

function compactQueryLabel(query: string) {
  const text = query.trim().replace(/\s+/g, '');
  if (!text) return '好拍';
  return text.length > 8 ? `${text.slice(0, 8)}...` : text;
}

function resultStatusPhrase(results: AIBuyerResult[]) {
  if (results.some((item) => item.status === 'LOT_STATUS_LIVE' || item.status === 'LOT_STATUS_EXTENDED')) {
    return '正在竞拍';
  }
  if (results.some((item) => item.status === 'LOT_STATUS_QUEUED')) {
    return '即将开拍';
  }
  return '可查看拍品';
}

function buyerReplyHeadline(query: string, reply: AIBuyerConsultReply) {
  if (!reply.results.length) return { lead: emptyCopy(query), highlight: '' };
  return {
    lead: `猜你想找${compactQueryLabel(query)}，当前有`,
    highlight: `${reply.results.length} 件${resultStatusPhrase(reply.results)}`,
  };
}

function SearchPage() {
  const restoredAIState = useMemo(() => readSearchAIStateForRestore(), []);
  const restoredScrollY = useRef(restoredAIState.scrollY);
  const restoredScrollDone = useRef(false);
  const searchRequestId = useRef(0);
  const [history, setHistory] = useState(readSearchHistory);
  const [expanded, setExpanded] = useState(false);
  const [guessRound, setGuessRound] = useState(0);
  const [activeRank, setActiveRank] = useState<SearchRankKey>('hot');
  const [activeCategory, setActiveCategory] = useState(Object.keys(CATEGORY_RANKS)[0]);
  const [query, setQuery] = useState('');
  const [aiQuery, setAIQuery] = useState(restoredAIState.query);
  const [aiReply, setAIReply] = useState<AIBuyerConsultReply | null>(restoredAIState.reply);
  const [aiLoading, setAILoading] = useState(false);
  const [aiError, setAIError] = useState('');
  const [aiPrompts, setAIPrompts] = useState(() => nextPromptBatch(AI_PROMPTS));
  const [aiPromptsRefreshing, setAIPromptsRefreshing] = useState(false);
  const [aiOpen, setAIOpen] = useState(() => Boolean(restoredAIState.query || restoredAIState.reply));
  const [aiResultsExpanded, setAIResultsExpanded] = useState(false);
  const [roomNames, setRoomNames] = useState<Record<string, string>>({});

  const visibleHistory = expanded ? history : history.slice(0, 2);
  useEffect(() => {
    if (!storageAvailable()) return;
    window.localStorage.setItem(SEARCH_HISTORY_STORAGE_KEY, JSON.stringify(history.slice(0, 10)));
  }, [history]);

  useEffect(() => {
    let disposed = false;
    void listPublicRooms()
      .then((rooms) => {
        if (disposed) return;
        setRoomNames(Object.fromEntries(rooms.map((room) => [room.id, room.name])));
      })
      .catch(() => {});
    return () => {
      disposed = true;
    };
  }, []);

  useEffect(() => {
    let disposed = false;
    void listBuyerSuggestions(6)
      .then((reply) => {
        if (disposed) return;
        const next = Array.from(new Set(
          (reply.suggestions || [])
            .map((item) => item.text.trim())
            .filter(Boolean),
        )).slice(0, 6);
        if (next.length) setAIPrompts((current) => nextPromptBatch([...next, ...AI_PROMPTS], current));
      })
      .catch(() => {});
    return () => {
      disposed = true;
    };
  }, []);

  const refreshAIPrompts = async () => {
    setAIPromptsRefreshing(true);
    try {
      const reply = await listBuyerSuggestions(12);
      const next = Array.from(new Set(
        (reply.suggestions || [])
          .map((item) => item.text.trim())
          .filter(Boolean),
      ));
      setAIPrompts((current) => nextPromptBatch(next.length ? [...next, ...AI_PROMPTS, ...current] : [...AI_PROMPTS, ...current], current));
    } catch {
      setAIPrompts((current) => nextPromptBatch([...AI_PROMPTS, ...current], current));
    } finally {
      setAIPromptsRefreshing(false);
    }
  };

  useEffect(() => {
    if (restoredScrollDone.current || !aiReply || restoredScrollY.current <= 0) return;
    restoredScrollDone.current = true;
    window.requestAnimationFrame(() => window.scrollTo({ top: restoredScrollY.current }));
  }, [aiReply]);

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
  const updateHistory = (text: string) => {
    setHistory((current) => [text, ...current.filter((item) => item !== text)].slice(0, 10));
  };
  const runPlainSearch = () => {
    const text = query.trim();
    if (!text) return;
    updateHistory(text);
  };
  const runAIConsult = async (nextQuery = aiQuery) => {
    const text = nextQuery.trim();
    if (!text) {
      setAIError('请输入想找的拍品、预算或用途');
      return;
    }
    const requestId = searchRequestId.current + 1;
    searchRequestId.current = requestId;
    setAIOpen(true);
    setAILoading(true);
    setAIError('');
    setAIQuery(text);
    setAIReply(null);
    setAIResultsExpanded(false);
    try {
      const reply = await consultBuyer({ query: text });
      if (searchRequestId.current !== requestId) return;
      setAIReply(reply);
      updateHistory(text);
    } catch (error) {
      if (searchRequestId.current !== requestId) return;
      setAIError(error instanceof Error ? error.message : '找拍品服务暂时不可用');
    } finally {
      if (searchRequestId.current === requestId) setAILoading(false);
    }
  };
  const persistBeforeEnter = () => saveSearchAIStateForRoomReturn(aiQuery, aiReply, window.scrollY);
  const aiReplyHeadline = aiReply ? buyerReplyHeadline(aiQuery, aiReply) : null;
  const visibleAIResults = aiReply
    ? (aiResultsExpanded ? aiReply.results : aiReply.results.slice(0, 1))
    : [];
  const aiGateHint = aiReplyHeadline?.highlight
    ? `${aiReplyHeadline.lead} ${aiReplyHeadline.highlight}`
    : '智能帮你找正在拍的好物';

  return (
    <main className="mobileShell dySearchPage" aria-label="抖音搜索">
      <header className="dySearchPageHeader">
        <button type="button" aria-label="返回" onClick={goBack}>
          <ChevronLeft className="dySearchIcon" aria-hidden="true" strokeWidth={2.8} />
        </button>
        <label>
          <span aria-hidden="true"><Search className="dySearchIcon" strokeWidth={2.4} /></span>
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
        <section className={`dySearchAI ${aiOpen ? 'isOpen' : 'isCollapsed'}`} aria-label="找好拍" aria-busy={aiLoading}>
          <button
            type="button"
            className="dySearchAIGate"
            aria-expanded={aiOpen}
            onClick={() => setAIOpen((value) => !value)}
          >
            <span className="dySearchAIGateIcon" aria-hidden="true"><Sparkles className="dySearchIcon" strokeWidth={2.6} /></span>
            <span className="dySearchAIGateText">
              <b>找好拍</b>
              <small>{aiGateHint}</small>
            </span>
            <ChevronDown className="dySearchAIGateChevron dySearchIcon" aria-hidden="true" strokeWidth={2.6} />
          </button>

          <div className="dySearchAIPanelShell" aria-hidden={!aiOpen}>
            <div className="dySearchAIPanelInner">
              <form
                className="dySearchAIBox"
                onSubmit={(event) => {
                  event.preventDefault();
                  void runAIConsult();
                }}
              >
                <div className="dySearchAIInputPill">
                  <input
                    value={aiQuery}
                    aria-label="找好拍条件"
                    placeholder="预算 / 品类 / 用途 / 直播间"
                    onChange={(event) => setAIQuery(event.target.value)}
                    tabIndex={aiOpen ? 0 : -1}
                  />
                  <button type="submit" disabled={aiLoading} tabIndex={aiOpen ? 0 : -1}>
                    {aiLoading ? '查找中' : '找一下'}
                  </button>
                </div>
              </form>
              {aiLoading ? (
                <div className="dySearchAILoading">正在从公开直播间里帮你找好拍...</div>
              ) : null}
              {!aiLoading && aiError ? (
                <div className="dySearchAIErrorState">
                  <p className="dySearchAIError">{aiError}</p>
                  <button type="button" onClick={() => void runAIConsult(aiQuery)} tabIndex={aiOpen ? 0 : -1}>重新找拍</button>
                </div>
              ) : null}
              {!aiLoading && !aiError && aiReply && aiReplyHeadline ? (
                <div className="dySearchAIReply">
                  <div className="dySearchAIReplySummary">
                    <span aria-hidden="true"><Target className="dySearchIcon" strokeWidth={2.8} /></span>
                    <p>
                      {aiReplyHeadline.lead}
                      {' '}
                      {aiReplyHeadline.highlight ? <b>{aiReplyHeadline.highlight}</b> : null}
                    </p>
                  </div>
                  {aiReply.fallbackUsed ? <small>已基于当前公开拍品为你推荐</small> : null}
                  {aiReply.results.length ? (
                    <>
                      <div className="dySearchAIResults">
                        {visibleAIResults.map((item) => (
                          <SearchAIResultCard
                            item={item}
                            key={`${item.roomId}-${item.lotId}`}
                            roomName={roomNames[item.roomId]}
                            onEnter={persistBeforeEnter}
                          />
                        ))}
                      </div>
                      <button
                        type="button"
                        className="dySearchAIExpand"
                        disabled={aiReply.results.length <= 1}
                        onClick={() => setAIResultsExpanded((value) => !value)}
                        tabIndex={aiOpen ? 0 : -1}
                      >
                        {aiResultsExpanded ? '收起' : '展开全部'}
                        <ChevronDown className="dySearchIcon" aria-hidden="true" strokeWidth={2.6} />
                      </button>
                    </>
                  ) : (
                    <div className="dySearchAIEmptyState">
                      <p className="dySearchAIEmpty">{emptyCopy(aiQuery)}</p>
                      <div className="dySearchAIPrompts compact">
                        {emptyPrompts(aiQuery).map((item) => (
                          <button type="button" key={item} onClick={() => void runAIConsult(item)} tabIndex={aiOpen ? 0 : -1}>
                            {item}
                          </button>
                        ))}
                      </div>
                    </div>
                  )}
                </div>
              ) : null}
              {!aiLoading && !aiError && !aiReply ? (
                <div className="dySearchAIPromptPanel">
                  <div className="dySearchAIPromptHeader">
                    <span>为你精选</span>
                    <button type="button" onClick={() => void refreshAIPrompts()} disabled={aiPromptsRefreshing} tabIndex={aiOpen ? 0 : -1}>
                      <RefreshCw className="dySearchIcon" aria-hidden="true" strokeWidth={2.4} />
                      {aiPromptsRefreshing ? '换中' : '换一批'}
                    </button>
                  </div>
                  <div className="dySearchAIPrompts">
                    {aiPrompts.map((item) => (
                      <button type="button" key={item} onClick={() => void runAIConsult(item)} tabIndex={aiOpen ? 0 : -1}>
                        {item}
                      </button>
                    ))}
                  </div>
                </div>
              ) : null}
            </div>
          </div>
        </section>

        <section className="dySearchPageHistory" aria-label="搜索历史">
          {visibleHistory.map((item) => (
            <div className="dySearchPageHistoryRow" key={item}>
              <button type="button" className="dySearchPageHistoryText" onClick={() => void runAIConsult(item)}>
                <span aria-hidden="true"><Clock3 className="dySearchIcon" strokeWidth={2.3} /></span>
                <b>{item}</b>
              </button>
              <button
                type="button"
                aria-label={`删除 ${item}`}
                onClick={() => setHistory((current) => current.filter((entry) => entry !== item))}
              >
                <X className="dySearchIcon" aria-hidden="true" strokeWidth={2.3} />
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
            <h2><span aria-hidden="true"><Flame className="dySearchIcon" strokeWidth={2.4} /></span>猜你想搜</h2>
            <button type="button" onClick={() => setGuessRound((round) => round + 1)}>
              <RefreshCw className="dySearchIcon" aria-hidden="true" strokeWidth={2.4} />
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

function SearchAIResultCard({ item, roomName, onEnter }: { item: AIBuyerResult; roomName?: string; onEnter: () => void }) {
  const href = item.href || `/m/room/${item.roomId}`;
  const tags = displayTags(item);
  const [imageFailed, setImageFailed] = useState(false);
  const showImage = Boolean(item.imageUrl && !imageFailed);
  return (
    <article className="dySearchAIResultCard">
      <a href={href} className="dySearchAIResultThumb" onClick={onEnter} aria-label={`进入${item.title}`}>
        {showImage ? (
          <img
            className="dySearchAIResultImage"
            src={item.imageUrl}
            alt={item.title}
            loading="lazy"
            onError={(event) => {
              event.currentTarget.style.display = 'none';
              setImageFailed(true);
            }}
          />
        ) : null}
        {!showImage ? <span className="dySearchAIResultImage dySearchAIResultImageEmpty" aria-hidden="true">拍</span> : null}
      </a>
      <div className="dySearchAIResultInfo">
        <div className="dySearchAIResultTop">
          <span>{formatLotStatus(item.status)}</span>
          <small>{roomName || `直播间 ${item.roomId}`}</small>
        </div>
        <a href={href} className="dySearchAIResultTitle" onClick={onEnter}>{item.title}</a>
        <div className="dySearchAIResultMeta">
          <strong><em>{priceLabel(item.status)}</em>{formatMoney(item.currentPrice)}</strong>
        </div>
        {tags.length ? (
          <div className="dySearchAIReasonTags">
            {tags.map((tag) => <span key={tag}>{tag}</span>)}
          </div>
        ) : null}
        <p>{displayReason(item)}</p>
      </div>
      <a href={href} className="dySearchAIEnter" onClick={onEnter}>进入直播间</a>
    </article>
  );
}

export default SearchPage;
