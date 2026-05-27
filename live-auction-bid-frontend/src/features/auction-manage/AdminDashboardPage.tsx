import { useCallback, useEffect, useMemo, useRef, useState, type CSSProperties, type ReactNode } from 'react';
import {
  AlertTriangle,
  ArrowDownRight,
  ArrowUpRight,
  BarChart3,
  CheckCircle2,
  CircleDollarSign,
  Clock3,
  CreditCard,
  Gavel,
  ListChecks,
  Package,
  Percent,
  ReceiptText,
  RefreshCw,
  ShieldAlert,
  ShoppingBag,
  TrendingUp,
  Users,
} from 'lucide-react';
import { getRoomSnapshot, listAdminLots } from '../auction/api/auctionApi';
import { listAdminOrders } from '../order/api/orderApi';
import type { OrderSummary } from '../order/model/orderTypes';
import { isSettlementLot, lotStatusLabel, lotStatusTone, settlementOutcomeDisplay } from '../../entities/auction/model/auctionStatus';
import { isAbnormalOrder, paymentStatusLabel, paymentStatusTone, type PaymentStatus } from '../../entities/order/model/orderStatus';
import type { AuctionEvent, Bid, Lot, Money, RoomSnapshot } from '../../shared/api/types';
import { resultMessage } from '../../shared/api/result';
import { formatDateTimeText, formatMoneyText } from '../../shared/lib/format';
import { ADMIN_ROOM } from '../../shared/config/studio';
import { REALTIME_CONSOLE_EVENTS, REALTIME_EVENT } from '../../shared/realtime/events';
import { useRoomSocket } from '../../shared/realtime/useRoomSocket';
import { StudioBadge, StudioButton, StudioCard, StudioEmptyState, StudioPageHeader, StudioTableSkeleton, type StudioTone } from '../../pages/host-console/components/studio-ui';
import './admin-dashboard.css';

type DashboardRange = 'today' | 'live' | '7d' | '30d';
type MetricFormat = 'money' | 'number' | 'percent';

type TimeBucket = {
  label: string;
  start: number;
  end: number;
  gmv: number;
  paid: number;
  pending: number;
  abnormal: number;
  orders: number;
};

type FunnelStep = {
  label: string;
  value: number;
  hint: string;
};

type LotPerformance = {
  lot: Lot;
  amountYuan: number;
  amountLabel: string;
  startYuan: number;
  premiumRate: number;
  participantCount: number;
  bidCount?: number;
  paymentStatus?: PaymentStatus;
  paid: boolean;
  statusLabel: string;
  statusTone: StudioTone;
};

type DashboardAnalytics = {
  rangeLots: Lot[];
  rangeOrders: OrderSummary[];
  paidOrders: OrderSummary[];
  pendingOrders: OrderSummary[];
  abnormalOrders: OrderSummary[];
  paidAmountYuan: number;
  pendingAmountYuan: number;
  abnormalAmountYuan: number;
  gmvYuan: number;
  paymentRate: number;
  dealRate: number;
  participantCount: number;
  averageDealYuan: number;
  queuedLots: Lot[];
  abnormalLots: Lot[];
  lowConversionLots: Lot[];
  timeSeries: TimeBucket[];
  funnel: FunnelStep[];
  topLots: LotPerformance[];
  lotPerformance: LotPerformance[];
  statusDistribution: Array<{ label: string; value: number; tone: StudioTone }>;
};

type MetricDefinition = {
  label: string;
  value: number;
  format: MetricFormat;
  icon: ReactNode;
  tone: 'green' | 'blue' | 'amber' | 'rose' | 'violet' | 'slate';
  trendValues: number[];
};

type CategoryPerformanceRow = {
  label: string;
  count: number;
  paidCount: number;
  amountYuan: number;
  premiumSum: number;
  participantSum: number;
};

const DAY_MS = 24 * 60 * 60 * 1000;
const MAX_PAGE_COUNT = 5;
const PAGE_SIZE = 100;
const VIEW_ANIMATION_TRIGGER_RATIO = 0.72;

const DONUT_SEGMENT_COLORS: Record<StudioTone, string> = {
  success: 'var(--studio-color-success)',
  warning: 'var(--studio-color-warning)',
  danger: 'var(--studio-color-error)',
  info: 'var(--studio-color-info)',
  purple: 'var(--studio-color-purple-text)',
  neutral: 'var(--studio-color-neutral)',
};

const DONUT_EMPTY_COLOR = 'var(--studio-color-neutral-border)';

const rangeOptions: Array<{ value: DashboardRange; label: string; detail: string }> = [
  { value: 'today', label: '今日', detail: '当天成交' },
  { value: 'live', label: '本场', detail: '当前直播间' },
  { value: '7d', label: '近7天', detail: '滚动一周' },
  { value: '30d', label: '近30天', detail: '默认视图' },
];

export function AdminDashboardPage({ roomId = ADMIN_ROOM.id }: { roomId?: string }) {
  const [range, setRange] = useState<DashboardRange>('30d');
  const [snapshot, setSnapshot] = useState<RoomSnapshot | null>(null);
  const [snapshotReceivedAt, setSnapshotReceivedAt] = useState(0);
  const [lots, setLots] = useState<Lot[]>([]);
  const [orders, setOrders] = useState<OrderSummary[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [lastUpdatedAt, setLastUpdatedAt] = useState(0);
  const [refreshSeq, setRefreshSeq] = useState(0);
  const [nowMs, setNowMs] = useState(() => Date.now());

  const commitSnapshot = useCallback((next: RoomSnapshot) => {
    setSnapshot(next);
    setSnapshotReceivedAt(Date.now());
  }, []);

  const sync = useCallback(async (silent = false) => {
    if (!silent) setLoading(true);
    setError('');
    try {
      const [nextSnapshot, nextLots, nextOrders] = await Promise.all([
        getRoomSnapshot(roomId),
        fetchAllLots(roomId),
        fetchMerchantOrders(roomId),
      ]);
      commitSnapshot(nextSnapshot);
      setLots(nextLots);
      setOrders(nextOrders);
      setLastUpdatedAt(Date.now());
      setRefreshSeq((current) => current + 1);
    } catch (e) {
      setError(resultMessage(e));
    } finally {
      if (!silent) setLoading(false);
    }
  }, [commitSnapshot, roomId]);

  const applyEvent = useCallback((event: AuctionEvent) => {
    if (event.snapshot) commitSnapshot(event.snapshot);
    if (event.lot) setLots((current) => upsertById(current, event.lot as Lot));
    if (event.bid || event.ranking?.length || event.lot) {
      setSnapshot((current) => patchSnapshot(current, event));
      setSnapshotReceivedAt(Date.now());
    }
    if (shouldRefreshBusinessData(event.type)) void sync(true);
  }, [commitSnapshot, sync]);

  useRoomSocket({
    roomId,
    handledEventTypes: REALTIME_CONSOLE_EVENTS,
    recoverSnapshot: async () => {
      const next = await getRoomSnapshot(roomId);
      commitSnapshot(next);
      return next;
    },
    onSnapshot: commitSnapshot,
    onEvent: applyEvent,
    onError: (e) => setError(resultMessage(e)),
  });

  useEffect(() => { void sync(); }, [sync]);
  useEffect(() => {
    const timer = window.setInterval(() => setNowMs(Date.now()), 1000);
    return () => window.clearInterval(timer);
  }, []);

  const analytics = useMemo(() => buildDashboardAnalytics({ lots, orders, snapshot, range, nowMs }), [lots, orders, snapshot, range, nowMs]);
  const metricCards = useMemo(() => createMetricCards(analytics), [analytics]);
  const currentLot = snapshot?.currentLot ?? null;
  const serverNowMs = snapshot ? Number(snapshot.serverTimeUnixMs || 0) + Math.max(0, nowMs - snapshotReceivedAt) : nowMs;
  const remainingMs = currentLot ? Number(currentLot.endsAtUnixMs || 0) - serverNowMs : 0;
  const recentBids = useMemo(() => [...(snapshot?.recentBids ?? [])].sort((a, b) => Number(b.createdAtUnixMs || 0) - Number(a.createdAtUnixMs || 0)).slice(0, 5), [snapshot]);
  const initialLoading = loading && !lots.length && !orders.length;
  const [chartGridRef, chartGridVisible] = useScrollReveal<HTMLElement>(VIEW_ANIMATION_TRIGGER_RATIO);
  const [performanceGridRef, performanceGridVisible] = useScrollReveal<HTMLElement>(VIEW_ANIMATION_TRIGGER_RATIO);
  const [riskGridRef, riskGridVisible] = useScrollReveal<HTMLElement>(VIEW_ANIMATION_TRIGGER_RATIO);

  return <section className="merchantDashboardPage">
    <StudioCard padding="lg" className="merchantDashboardHero">
      <StudioPageHeader
        eyebrow="Business cockpit"
        title="主播/商家经营数据看板"
        description="围绕成交、支付、拍品表现和待处理风险查看当前商家经营情况。"
        actions={<div className="merchantHeroActions">
          <a className="studioButton studioButton-primary studioButton-md" href="/admin/auctions/create">添加拍品</a>
          <a className="studioButton studioButton-secondary studioButton-md" href="/admin/orders">处理订单</a>
          <StudioButton type="button" variant="soft" icon={<RefreshCw size={15} />} loading={loading} onClick={() => void sync()}>刷新</StudioButton>
        </div>}
      />
      <div className="merchantRangeBar" aria-label="经营数据时间范围">
        {rangeOptions.map((option) => <button key={option.value} type="button" className={range === option.value ? 'active' : ''} aria-pressed={range === option.value} onClick={() => setRange(option.value)}>
          <b>{option.label}</b>
          <span>{option.detail}</span>
        </button>)}
      </div>
      <div className="merchantHeroMeta">
        <span>当前直播间 <b>{ADMIN_ROOM.name}</b></span>
        <span>数据范围 <b>{rangeOptions.find((item) => item.value === range)?.label}</b></span>
        <span>更新时间 <b>{lastUpdatedAt ? formatDateTimeText(lastUpdatedAt, '刚刚') : '加载中'}</b></span>
      </div>
    </StudioCard>

    {error ? <div className="auctionMgmtNotice danger"><AlertTriangle size={16} />{error}</div> : null}

    {initialLoading ? <StudioTableSkeleton rows={5} columns={4} /> : <>
      <section className="merchantMetricGrid" aria-label="经营核心指标">
        {metricCards.map((metric) => <MetricCard key={metric.label} metric={metric} />)}
      </section>

      <section className="merchantTopGrid">
        <LiveBusinessPanel currentLot={currentLot} recentBids={recentBids} remainingMs={remainingMs} participantCount={currentLot?.stats.participantCount ?? 0} />
        <ActionChecklist analytics={analytics} />
      </section>

      <section ref={chartGridRef} className={`merchantChartGrid merchantViewAnimate${chartGridVisible ? ' isVisible' : ''}`}>
        <StudioCard title="成交漏斗" subtitle="Auction funnel" className="merchantPanel merchantFunnelPanel">
          <FunnelChart steps={analytics.funnel} refreshSeq={refreshSeq} />
        </StudioCard>
        <StudioCard title="GMV / 支付趋势" subtitle="Revenue trend" className="merchantPanel merchantTrendPanel">
          <TrendChart buckets={analytics.timeSeries} refreshSeq={`${range}-${refreshSeq}`} />
        </StudioCard>
      </section>

      <section ref={performanceGridRef} className={`merchantPerformanceGrid merchantViewAnimate${performanceGridVisible ? ' isVisible' : ''}`}>
        <StudioCard title="成交额 Top 10" subtitle="Lot ranking" className="merchantPanel">
          <LotRanking lots={analytics.topLots} />
        </StudioCard>
        <StudioCard title="品类表现" subtitle="Category performance" className="merchantPanel">
          <CategoryPerformancePanel lots={analytics.lotPerformance} />
        </StudioCard>
      </section>

      <section ref={riskGridRef} className={`merchantRiskGrid merchantViewAnimate${riskGridVisible ? ' isVisible' : ''}`}>
        <StudioCard title="订单风险" subtitle="Order risk" className="merchantPanel">
          <OrderRiskPanel analytics={analytics} nowMs={nowMs} />
        </StudioCard>
        <StudioCard title="拍品表现明细" subtitle="Lot table" className="merchantPanel merchantLotTablePanel">
          <LotPerformanceTable lots={analytics.lotPerformance} refreshSeq={refreshSeq} />
        </StudioCard>
      </section>
    </>}
  </section>;
}

function MetricCard({ metric }: { metric: MetricDefinition }) {
  const delta = calcDelta(metric.trendValues);
  return <article className={`merchantMetricCard metric-${metric.tone}`}>
    <header>
      <span className="merchantMetricIcon">{metric.icon}</span>
      <TrendBadge delta={delta} />
    </header>
    <p>{metric.label}</p>
    <strong><AnimatedMetricValue value={metric.value} format={metric.format} /></strong>
    <MetricSparkline values={metric.trendValues} />
  </article>;
}

function TrendBadge({ delta }: { delta: number | null }) {
  if (delta === null) return <span className="merchantTrendBadge flat">暂无趋势</span>;
  const positive = delta >= 0;
  return <span className={`merchantTrendBadge ${positive ? 'up' : 'down'}`}>{positive ? <ArrowUpRight size={14} /> : <ArrowDownRight size={14} />}{positive ? '+' : ''}{delta.toFixed(1)}%</span>;
}

function AnimatedMetricValue({ value, format }: { value: number; format: MetricFormat }) {
  const reduceMotion = usePrefersReducedMotion();
  const [ref, visible] = useInView<HTMLSpanElement>();
  const [displayValue, setDisplayValue] = useState(() => (reduceMotion ? value : 0));
  const previousValue = useRef(reduceMotion ? value : 0);

  useEffect(() => {
    if (reduceMotion) {
      previousValue.current = value;
      setDisplayValue(value);
      return;
    }
    if (!visible) return;
    const startValue = previousValue.current;
    const startedAt = performance.now();
    let animationFrame = 0;
    const tick = (time: number) => {
      const progress = Math.min(1, (time - startedAt) / 980);
      const eased = easeOutCubic(progress);
      setDisplayValue(startValue + (value - startValue) * eased);
      if (progress < 1) {
        animationFrame = window.requestAnimationFrame(tick);
      } else {
        previousValue.current = value;
      }
    };
    animationFrame = window.requestAnimationFrame(tick);
    return () => window.cancelAnimationFrame(animationFrame);
  }, [format, reduceMotion, value, visible]);

  return <span ref={ref} className="merchantAnimatedValue">{formatMetricValue(displayValue, format)}</span>;
}

function MetricSparkline({ values }: { values: number[] }) {
  const geometry = getSparklineGeometry(values, 112, 38);
  const animationKey = values.map((value) => Number(value.toFixed(3))).join('|');
  return <svg key={animationKey} className="merchantSparkline" viewBox="0 0 112 38" role="img" aria-label="迷你趋势线" focusable="false">
    <path className="merchantSparklineGuide" d={`M3 ${geometry.guideY.toFixed(2)}H109`} />
    <path className="merchantSparklineArea" d={geometry.areaPath} />
    <path className="merchantSparklineLine" d={geometry.linePath} pathLength={1} />
    <circle className="merchantSparklineHalo" cx={geometry.lastPoint.x} cy={geometry.lastPoint.y} r="4.8" />
    <circle className="merchantSparklineDot" cx={geometry.lastPoint.x} cy={geometry.lastPoint.y} r="2.7" />
  </svg>;
}

function LiveBusinessPanel({ currentLot, recentBids, remainingMs, participantCount }: { currentLot: Lot | null; recentBids: Bid[]; remainingMs: number; participantCount: number }) {
  return <StudioCard title="本场焦点" subtitle="Live business" className="merchantPanel merchantLivePanel" actions={currentLot ? <StudioBadge tone={lotStatusTone(currentLot.status)}>{lotStatusLabel(currentLot.status)}</StudioBadge> : <StudioBadge tone="neutral">待开拍</StudioBadge>}>
    {currentLot ? <div className="merchantLiveLot">
      <div className="merchantLiveImage">{currentLot.imageUrl ? <img src={currentLot.imageUrl} alt={currentLot.title} /> : <Gavel size={34} />}</div>
      <div className="merchantLiveBody">
        <h3>{currentLot.title}</h3>
        <div className="merchantLivePrice"><span>当前价</span><b>{formatMoneyText(currentLot.currentPrice)}</b></div>
        <div className="merchantLiveFacts">
          <span>剩余时间 <b>{formatCountdown(remainingMs)}</b></span>
          <span>领先用户 <b>{currentLot.leadingNickname || currentLot.leadingUserId || '暂无'}</b></span>
          <span>参拍人数 <b>{participantCount.toLocaleString('zh-CN')}</b></span>
        </div>
        <div className="merchantBidTape" aria-label="最近出价">
          {recentBids.length ? recentBids.map((bid) => <div key={bid.id}><span>{bid.nickname || bid.userId || '买家'}</span><b>{formatMoneyText(bid.amount)}</b></div>) : <div><span>暂无出价</span><b>等待开拍</b></div>}
        </div>
      </div>
    </div> : <StudioEmptyState compact icon={<Gavel size={28} />} title="当前没有正在拍" description="开拍后这里展示当前价、领先用户和最近出价。" action={<a className="studioButton studioButton-primary studioButton-sm" href="/admin/auctions">进入本场队列</a>} />}
  </StudioCard>;
}

function ActionChecklist({ analytics }: { analytics: DashboardAnalytics }) {
  const actions = [
    {
      label: '催付',
      href: '/admin/orders',
      count: analytics.pendingOrders.length,
      detail: `${formatYuan(analytics.pendingAmountYuan)} 待支付`,
      icon: <Clock3 size={18} />,
      tone: 'warning' as StudioTone,
    },
    {
      label: '补队列',
      href: '/admin/auctions/create',
      count: Math.max(0, 3 - analytics.queuedLots.length),
      detail: analytics.queuedLots.length >= 3 ? '本场队列充足' : `待开拍仅 ${analytics.queuedLots.length} 件`,
      icon: <Package size={18} />,
      tone: analytics.queuedLots.length >= 3 ? 'success' as StudioTone : 'warning' as StudioTone,
    },
    {
      label: '处理异常',
      href: '/admin/orders',
      count: analytics.abnormalOrders.length + analytics.abnormalLots.length,
      detail: `${analytics.abnormalOrders.length} 个订单，${analytics.abnormalLots.length} 件拍品`,
      icon: <ShieldAlert size={18} />,
      tone: 'danger' as StudioTone,
    },
    {
      label: '复盘低转化拍品',
      href: '/admin/auctions/history',
      count: analytics.lowConversionLots.length,
      detail: '低溢价、流拍或付款超时',
      icon: <ListChecks size={18} />,
      tone: 'info' as StudioTone,
    },
  ];
  return <StudioCard title="行动清单" subtitle="Next actions" className="merchantPanel merchantActionPanel">
    <div className="merchantActionList">{actions.map((action) => <a key={action.label} href={action.href} className={`merchantActionItem merchantAction-${action.tone}`}>
      <span>{action.icon}</span>
      <div><b>{action.label}</b><small>{action.detail}</small></div>
      <strong>{action.count.toLocaleString('zh-CN')}</strong>
    </a>)}</div>
  </StudioCard>;
}

function FunnelChart({ steps, refreshSeq }: { steps: FunnelStep[]; refreshSeq: number }) {
  const max = Math.max(1, ...steps.map((step) => step.value));
  if (!steps.some((step) => step.value > 0)) return <StudioEmptyState compact icon={<BarChart3 size={28} />} title="暂无漏斗数据" description="当前范围没有可统计的拍品。" />;
  return <div className="merchantFunnel" key={refreshSeq}>{steps.map((step, index) => {
    const width = Math.max(8, (step.value / max) * 100);
    return <div className="merchantFunnelRow" key={step.label}>
      <div className="merchantFunnelLabel"><b>{step.label}</b><span>{step.hint}</span></div>
      <div className="merchantFunnelTrack"><span style={{ '--funnel-width': `${width}%`, '--delay': `${index * 80}ms` } as CSSProperties} /></div>
      <strong>{step.value.toLocaleString('zh-CN')}</strong>
    </div>;
  })}</div>;
}

function TrendChart({ buckets, refreshSeq }: { buckets: TimeBucket[]; refreshSeq: string }) {
  const [activeIndex, setActiveIndex] = useState(Math.max(0, buckets.length - 1));
  useEffect(() => setActiveIndex(Math.max(0, buckets.length - 1)), [buckets]);
  if (!buckets.some((bucket) => bucket.gmv || bucket.paid || bucket.pending || bucket.abnormal)) {
    return <StudioEmptyState compact icon={<TrendingUp size={28} />} title="暂无趋势数据" description="当前范围还没有成交订单。" />;
  }
  const width = 760;
  const height = 270;
  const padding = { top: 20, right: 24, bottom: 34, left: 46 };
  const chartWidth = width - padding.left - padding.right;
  const chartHeight = height - padding.top - padding.bottom;
  const maxValue = Math.max(1, ...buckets.map((bucket) => Math.max(bucket.gmv, bucket.paid + bucket.pending + bucket.abnormal)));
  const xFor = (index: number) => padding.left + (buckets.length <= 1 ? chartWidth / 2 : (chartWidth / (buckets.length - 1)) * index);
  const yFor = (value: number) => padding.top + chartHeight - (value / maxValue) * chartHeight;
  const linePoints = buckets.map((bucket, index) => `${xFor(index)},${yFor(bucket.gmv)}`).join(' ');
  const areaPoints = `${padding.left},${padding.top + chartHeight} ${linePoints} ${padding.left + chartWidth},${padding.top + chartHeight}`;
  const active = buckets[activeIndex] ?? buckets[buckets.length - 1];

  return <div className="merchantTrendChart" key={refreshSeq}>
    <div className="merchantLegend">
      <span><i className="gmv" />GMV</span>
      <span><i className="paid" />已支付</span>
      <span><i className="pending" />待支付</span>
      <span><i className="abnormal" />异常</span>
    </div>
    <svg viewBox={`0 0 ${width} ${height}`} role="img" aria-label="GMV 和支付趋势图">
      {[0, 0.25, 0.5, 0.75, 1].map((tick) => <g key={tick}>
        <line x1={padding.left} x2={width - padding.right} y1={padding.top + chartHeight * tick} y2={padding.top + chartHeight * tick} />
        <text x={10} y={padding.top + chartHeight * tick + 4}>{formatCompactYuan(maxValue * (1 - tick))}</text>
      </g>)}
      <polygon className="merchantArea" points={areaPoints} />
      {buckets.map((bucket, index) => {
        const x = xFor(index);
        const barWidth = Math.max(8, Math.min(18, chartWidth / Math.max(1, buckets.length) * 0.42));
        let yCursor = padding.top + chartHeight;
        const segments = [
          { key: 'paid', label: '已支付', value: bucket.paid },
          { key: 'pending', label: '待支付', value: bucket.pending },
          { key: 'abnormal', label: '异常', value: bucket.abnormal },
        ];
        return <g key={bucket.label}>
          {segments.map((segment, segmentIndex) => {
            const segmentHeight = (segment.value / maxValue) * chartHeight;
            yCursor -= segmentHeight;
            return <rect
              key={segment.key}
              className={`bar-${segment.key}`}
              x={x - barWidth / 2}
              y={yCursor}
              width={barWidth}
              height={Math.max(0, segmentHeight)}
              rx={3}
              style={{ '--bar-delay': `${index * 70 + segmentIndex * 42}ms` } as CSSProperties}
            >
              <title>{bucket.label} {segment.label}: {formatYuan(segment.value)}</title>
            </rect>;
          })}
          <text x={x} y={height - 10}>{bucket.label}</text>
          <rect className="merchantHoverBand" x={x - chartWidth / Math.max(1, buckets.length) / 2} y={padding.top} width={chartWidth / Math.max(1, buckets.length)} height={chartHeight} onPointerEnter={() => setActiveIndex(index)} />
        </g>;
      })}
      <polyline className="merchantLine" points={linePoints} pathLength={1} />
      {buckets.map((bucket, index) => <circle
        key={`${bucket.label}-point`}
        className="merchantLinePoint"
        cx={xFor(index)}
        cy={yFor(bucket.gmv)}
        r={index === activeIndex ? 5 : 3}
        style={{ '--point-delay': `${520 + index * 45}ms` } as CSSProperties}
      />)}
    </svg>
    <div className="merchantChartTooltip">
      <b>{active?.label}</b>
      <span>GMV {formatYuan(active?.gmv ?? 0)}</span>
      <span>已支付 {formatYuan(active?.paid ?? 0)}</span>
      <span>待支付 {formatYuan(active?.pending ?? 0)}</span>
      <span>异常 {formatYuan(active?.abnormal ?? 0)}</span>
    </div>
  </div>;
}

function LotRanking({ lots }: { lots: LotPerformance[] }) {
  const max = Math.max(1, ...lots.map((lot) => lot.amountYuan));
  if (!lots.length) return <StudioEmptyState compact icon={<ShoppingBag size={28} />} title="暂无拍品成交额" description="当前范围没有成交拍品。" />;
  return <div className="merchantLotRanking">{lots.slice(0, 10).map((item, index) => <div key={item.lot.id}>
    <span>#{String(index + 1).padStart(2, '0')}</span>
    <div><b>{item.lot.title}</b><small>溢价率 {formatPercent(item.premiumRate)}</small></div>
    <div className="merchantRankBar"><i style={{ width: `${Math.max(5, (item.amountYuan / max) * 100)}%`, '--delay': `${index * 70}ms` } as CSSProperties} /></div>
    <strong>{formatYuan(item.amountYuan)}</strong>
  </div>)}</div>;
}

function CategoryPerformancePanel({ lots }: { lots: LotPerformance[] }) {
  const rows = buildCategoryPerformance(lots).slice(0, 6);
  if (!rows.length) return <StudioEmptyState compact icon={<BarChart3 size={28} />} title="暂无品类表现" description="当前范围没有可统计的成交品类。" />;
  const totalAmount = rows.reduce((sum, row) => sum + row.amountYuan, 0);
  const totalCount = rows.reduce((sum, row) => sum + row.count, 0);
  const maxAmount = Math.max(1, ...rows.map((row) => row.amountYuan));
  const leader = rows.reduce((best, row) => row.amountYuan > best.amountYuan ? row : best, rows[0]);
  return <div className="merchantCategoryPanel">
    <div className="merchantCategorySummary">
      <div><span>主力品类</span><b>{leader.label}</b><small>{formatYuan(leader.amountYuan)} · {formatPercent(totalAmount ? leader.amountYuan / totalAmount * 100 : 0)}</small></div>
      <div><span>覆盖品类</span><b>{rows.length.toLocaleString('zh-CN')} 类</b><small>{totalCount.toLocaleString('zh-CN')} 件成交，已支付 {rows.reduce((sum, row) => sum + row.paidCount, 0).toLocaleString('zh-CN')} 件</small></div>
    </div>
    <div className="merchantCategoryRows">
      {rows.map((row) => {
        const width = Math.max(5, row.amountYuan / maxAmount * 100);
        const payRate = row.count ? row.paidCount / row.count * 100 : 0;
        return <article key={row.label} className={row.label === leader.label ? 'isLeader' : ''}>
          <header><div><b>{row.label}</b><span>{row.count} 件成交 · 平均参拍 {formatOneDecimal(row.participantSum / row.count)} 人</span></div><strong>{formatYuan(row.amountYuan)}</strong></header>
          <div className="merchantCategoryTrack"><i style={{ '--category-width': `${width}%` } as CSSProperties} /></div>
          <footer><span>支付率 {formatPercent(payRate)}</span><span>平均溢价 {formatPercent(row.premiumSum / row.count)}</span><span>{formatPercent(totalAmount ? row.amountYuan / totalAmount * 100 : 0)} GMV</span></footer>
        </article>;
      })}
    </div>
  </div>;
}

function buildCategoryPerformance(lots: LotPerformance[]): CategoryPerformanceRow[] {
  const rows = new Map<string, CategoryPerformanceRow>();
  lots.forEach((item) => {
    if (item.amountYuan <= 0) return;
    const label = item.lot.category?.trim() || '未分类';
    const row = rows.get(label) ?? { label, count: 0, paidCount: 0, amountYuan: 0, premiumSum: 0, participantSum: 0 };
    row.count += 1;
    row.amountYuan += item.amountYuan;
    row.premiumSum += item.premiumRate;
    row.participantSum += item.participantCount;
    if (item.paid) row.paidCount += 1;
    rows.set(label, row);
  });
  return [...rows.values()].sort((a, b) => b.amountYuan - a.amountYuan);
}

function OrderRiskPanel({ analytics, nowMs }: { analytics: DashboardAnalytics; nowMs: number }) {
  const total = analytics.statusDistribution.reduce((sum, item) => sum + item.value, 0);
  const pending = analytics.pendingOrders
    .slice()
    .sort((a, b) => (Number(a.expiresAtUnixMs || 0) - Number(b.expiresAtUnixMs || 0)) || (moneyCents(b.amount) - moneyCents(a.amount)))
    .slice(0, 5);
  const gradient = createRiskGradient(analytics.statusDistribution);
  return <div className="merchantRiskPanelBody">
    {total ? <div className="merchantRiskDonut" key={`${total}-${gradient}`} style={{ '--risk-gradient': gradient } as CSSProperties} aria-label={`订单状态分布，共 ${total} 个订单`}><strong>{total.toLocaleString('zh-CN')}</strong><span>订单分布</span></div> : <StudioEmptyState compact icon={<ReceiptText size={28} />} title="暂无订单分布" description="当前范围没有订单。" />}
    <div className="merchantRiskLegend">{analytics.statusDistribution.map((item) => <div key={item.label}><StudioBadge tone={item.tone}>{item.label}</StudioBadge><b>{item.value.toLocaleString('zh-CN')}</b></div>)}</div>
    <div className="merchantCountdownList">
      <h3>待支付倒计时</h3>
      {pending.length ? pending.map((order) => {
        const leftMs = Number(order.expiresAtUnixMs || 0) - nowMs;
        return <a href="/admin/orders" key={order.id} className={leftMs <= 0 ? 'expired' : ''}>
          <div><b>{order.lotTitle || '落锤订单'}</b><span>{order.buyerNickname || order.buyerUserId || '买家'} · {formatYuan(moneyYuan(order.amount))}</span></div>
          <strong>{leftMs <= 0 ? '已超时' : formatCountdown(leftMs)}</strong>
        </a>;
      }) : <StudioEmptyState compact icon={<CheckCircle2 size={22} />} title="暂无待支付订单" description="当前范围没有需要催付的订单。" />}
    </div>
  </div>;
}

function LotPerformanceTable({ lots, refreshSeq }: { lots: LotPerformance[]; refreshSeq: number }) {
  if (!lots.length) return <StudioEmptyState compact icon={<ShoppingBag size={28} />} title="暂无拍品明细" description="当前范围没有拍品表现数据。" />;
  return <div className="merchantLotTable" key={refreshSeq}>
    <div className="merchantLotTableHead">
      <span>拍品</span>
      <span>状态</span>
      <span>起拍价</span>
      <span>结果金额</span>
      <span>溢价率</span>
      <span>出价次数</span>
      <span>支付</span>
      <span>操作</span>
    </div>
    {lots.slice(0, 12).map((item) => <div className="merchantLotRow" key={item.lot.id}>
      <div className="merchantLotIdentity" data-label="拍品">
        <span>{item.lot.imageUrl ? <img src={item.lot.imageUrl} alt={item.lot.title} /> : <ShoppingBag size={20} />}</span>
        <div><b>{item.lot.title}</b><small>{item.lot.category || '未分类'}</small></div>
      </div>
      <div data-label="状态"><StudioBadge tone={item.statusTone}>{item.statusLabel}</StudioBadge></div>
      <div data-label="起拍价">{formatYuan(item.startYuan)}</div>
      <div data-label={item.amountLabel}><strong>{item.amountYuan > 0 ? formatYuan(item.amountYuan) : '未成交'}</strong></div>
      <div data-label="溢价率">{formatPercent(item.premiumRate)}</div>
      <div data-label="出价次数">{item.bidCount === undefined ? '—' : item.bidCount.toLocaleString('zh-CN')}</div>
      <div data-label="支付">{item.paymentStatus ? <StudioBadge tone={paymentStatusTone(item.paymentStatus)}>{paymentStatusLabel(item.paymentStatus)}</StudioBadge> : <StudioBadge tone="neutral">无订单</StudioBadge>}</div>
      <div data-label="操作"><a href={`/admin/auctions/create?lotId=${encodeURIComponent(item.lot.id)}`}>查看</a></div>
    </div>)}
  </div>;
}

function createMetricCards(analytics: DashboardAnalytics): MetricDefinition[] {
  return [
    { label: 'GMV', value: analytics.gmvYuan, format: 'money', icon: <BarChart3 size={21} />, tone: 'green', trendValues: analytics.timeSeries.map((item) => item.gmv) },
    { label: '已支付金额', value: analytics.paidAmountYuan, format: 'money', icon: <CircleDollarSign size={21} />, tone: 'blue', trendValues: analytics.timeSeries.map((item) => item.paid) },
    { label: '待支付金额', value: analytics.pendingAmountYuan, format: 'money', icon: <CreditCard size={21} />, tone: 'amber', trendValues: analytics.timeSeries.map((item) => item.pending) },
    { label: '订单数', value: analytics.rangeOrders.length, format: 'number', icon: <ReceiptText size={21} />, tone: 'slate', trendValues: analytics.timeSeries.map((item) => item.orders) },
    { label: '支付率', value: analytics.paymentRate, format: 'percent', icon: <Percent size={21} />, tone: 'green', trendValues: analytics.timeSeries.map((item) => item.orders ? (item.paid / Math.max(1, item.gmv)) * 100 : 0) },
    { label: '成交率', value: analytics.dealRate, format: 'percent', icon: <Gavel size={21} />, tone: 'violet', trendValues: analytics.timeSeries.map((_, index) => (index + 1) * analytics.dealRate / Math.max(1, analytics.timeSeries.length)) },
    { label: '参拍人数', value: analytics.participantCount, format: 'number', icon: <Users size={21} />, tone: 'blue', trendValues: analytics.timeSeries.map((item) => item.orders) },
    { label: '平均成交价', value: analytics.averageDealYuan, format: 'money', icon: <TrendingUp size={21} />, tone: 'rose', trendValues: analytics.timeSeries.map((item) => item.orders ? item.gmv / item.orders : 0) },
  ];
}

function buildDashboardAnalytics({ lots, orders, snapshot, range, nowMs }: { lots: Lot[]; orders: OrderSummary[]; snapshot: RoomSnapshot | null; range: DashboardRange; nowMs: number }): DashboardAnalytics {
  const startMs = getRangeStartMs(range, nowMs);
  const rangeOrders = filterOrdersByRange(orders, range, startMs);
  const rangeLots = filterLotsByRange(lots, range, startMs);
  const paidOrders = rangeOrders.filter(isPaidOrder);
  const pendingOrders = rangeOrders.filter(isPendingOrder);
  const abnormalOrders = rangeOrders.filter((order) => isAbnormalOrder(order.status, order.paymentStatus));
  const paidAmountYuan = sumOrderYuan(paidOrders);
  const pendingAmountYuan = sumOrderYuan(pendingOrders);
  const abnormalAmountYuan = sumOrderYuan(abnormalOrders);
  const gmvYuan = paidAmountYuan;
  const settledLots = rangeLots.filter(isSettlementLot);
  const startedLots = rangeLots.filter(hasStartedLot);
  const withBidLots = rangeLots.filter((lot) => lotHasBid(lot, snapshot, rangeOrders));
  const paidLotIds = new Set(paidOrders.map((order) => order.lotId).filter(Boolean));
  const queuedLots = lots.filter(isQueueLot);
  const abnormalLots = lots.filter(isAbnormalLot);
  const participantCount = rangeLots.reduce((sum, lot) => sum + (lot.stats?.participantCount ?? 0), 0);
  const lotPerformance = buildLotPerformance(rangeLots, rangeOrders, nowMs).sort((a, b) => b.amountYuan - a.amountYuan);
  const lowConversionLots = lotPerformance.filter((item) => item.lot.status === 'LOT_STATUS_FAILED' || item.premiumRate <= 0 || (!item.paid && isSettlementLot(item.lot)));
  const timeSeries = buildTimeSeries(range, rangeOrders, nowMs);

  return {
    rangeLots,
    rangeOrders,
    paidOrders,
    pendingOrders,
    abnormalOrders,
    paidAmountYuan,
    pendingAmountYuan,
    abnormalAmountYuan,
    gmvYuan,
    paymentRate: rangeOrders.length ? paidOrders.length / rangeOrders.length * 100 : 0,
    dealRate: startedLots.length ? paidLotIds.size / startedLots.length * 100 : 0,
    participantCount,
    averageDealYuan: paidOrders.length ? paidAmountYuan / paidOrders.length : 0,
    queuedLots,
    abnormalLots,
    lowConversionLots: lowConversionLots.map((item) => item.lot),
    timeSeries,
    funnel: [
      { label: '已创建拍品', value: rangeLots.length, hint: '当前范围内可统计拍品' },
      { label: '已上架', value: rangeLots.filter(isListedLot).length, hint: '已进入上架或队列' },
      { label: '已开拍', value: startedLots.length, hint: '产生开拍时间' },
      { label: '有出价', value: withBidLots.length, hint: '有领先用户或最近出价' },
      { label: '已落锤', value: settledLots.length, hint: '已进入结算的拍品' },
      { label: '已成交', value: paidLotIds.size, hint: '支付成功订单关联拍品' },
    ],
    topLots: lotPerformance.filter((item) => item.amountYuan > 0).slice(0, 10),
    lotPerformance,
    statusDistribution: [
      { label: '已支付', value: paidOrders.length, tone: 'success' },
      { label: '待支付', value: pendingOrders.length, tone: 'warning' },
      { label: '超时', value: rangeOrders.filter((order) => order.status === 'EXPIRED').length, tone: 'danger' },
      { label: '异常', value: abnormalOrders.length, tone: 'danger' },
    ],
  };
}

async function fetchAllLots(roomId: string) {
  const first = await listAdminLots({ page: 1, pageSize: PAGE_SIZE, roomId });
  const pageCount = Math.min(MAX_PAGE_COUNT, Math.ceil(first.total / PAGE_SIZE));
  if (pageCount <= 1) return first.lots;
  const pages = await Promise.all(Array.from({ length: pageCount - 1 }, (_, index) => listAdminLots({ page: index + 2, pageSize: PAGE_SIZE, roomId })));
  return [first, ...pages].flatMap((page) => page.lots);
}

async function fetchMerchantOrders(roomId: string) {
  const first = await listAdminOrders({ page: 1, pageSize: PAGE_SIZE });
  const pageCount = Math.min(MAX_PAGE_COUNT, Math.ceil(first.total / PAGE_SIZE));
  const pages = pageCount <= 1 ? [] : await Promise.all(Array.from({ length: pageCount - 1 }, (_, index) => listAdminOrders({ page: index + 2, pageSize: PAGE_SIZE })));
  const allOrders = [first, ...pages].flatMap((page) => page.orders);
  const roomOrders = allOrders.filter((order) => order.roomId === roomId);
  return roomOrders.length || !allOrders.length ? roomOrders : allOrders;
}

function buildLotPerformance(lots: Lot[], orders: OrderSummary[], nowMs: number): LotPerformance[] {
  const ordersByLot = new Map<string, OrderSummary[]>();
  orders.forEach((order) => {
    ordersByLot.set(order.lotId, [...(ordersByLot.get(order.lotId) ?? []), order]);
  });
  return lots.map((lot) => {
    const relatedOrders = ordersByLot.get(lot.id) ?? [];
    const paidOrder = relatedOrders.find(isPaidOrder);
    const latestOrder = paidOrder ?? relatedOrders[0];
    const outcome = isSettlementLot(lot) ? settlementOutcomeDisplay(lot, latestOrder, nowMs) : null;
    const amountYuan = latestOrder ? moneyYuan(latestOrder.amount) : moneyYuan(lotResultMoney(lot));
    const startYuan = moneyYuan(lot.rule.startPrice);
    return {
      lot,
      amountYuan,
      amountLabel: outcome?.priceLabel ?? (amountYuan > 0 ? '当前价' : '起拍价'),
      startYuan,
      premiumRate: startYuan > 0 ? (amountYuan - startYuan) / startYuan * 100 : 0,
      participantCount: lot.stats?.participantCount ?? (latestOrder ? 1 : 0),
      bidCount: lot.stats?.bidCount ?? 0,
      paymentStatus: latestOrder?.paymentStatus,
      paid: latestOrder ? isPaidOrder(latestOrder) : false,
      statusLabel: outcome?.label ?? lotStatusLabel(lot.status),
      statusTone: outcome?.tone ?? lotStatusTone(lot.status),
    };
  });
}

function buildTimeSeries(range: DashboardRange, orders: OrderSummary[], nowMs: number): TimeBucket[] {
  const count = range === 'today' ? 12 : range === '7d' ? 7 : range === '30d' ? 10 : 8;
  const startMs = range === 'live'
    ? Math.min(...orders.map((order) => orderTime(order)).filter(Boolean), nowMs - 4 * 60 * 60 * 1000)
    : getRangeStartMs(range, nowMs);
  const span = Math.max(1, nowMs - startMs);
  const bucketSize = Math.max(1, span / count);
  const buckets = Array.from({ length: count }, (_, index) => {
    const start = startMs + bucketSize * index;
    const end = index === count - 1 ? nowMs + 1 : start + bucketSize;
    return { label: formatBucketLabel(start, range), start, end, gmv: 0, paid: 0, pending: 0, abnormal: 0, orders: 0 } satisfies TimeBucket;
  });
  orders.forEach((order) => {
    const time = orderTime(order);
    if (!time) return;
    const bucket = buckets.find((item) => time >= item.start && time < item.end);
    if (!bucket) return;
    const amount = moneyYuan(order.amount);
    bucket.orders += 1;
    if (isPaidOrder(order)) {
      bucket.gmv += amount;
      bucket.paid += amount;
    } else if (isPendingOrder(order)) {
      bucket.pending += amount;
    }
    if (isAbnormalOrder(order.status, order.paymentStatus)) bucket.abnormal += amount;
  });
  return buckets;
}

function lotResultMoney(lot: Lot) {
  if (isSettlementLot(lot) && moneyCents(lot.finalPrice) > 0) return lot.finalPrice;
  if (moneyCents(lot.currentPrice) > 0) return lot.currentPrice;
  return lot.rule.startPrice;
}

function filterOrdersByRange(orders: OrderSummary[], range: DashboardRange, startMs: number) {
  if (range === 'live') return orders;
  return orders.filter((order) => orderTime(order) >= startMs);
}

function filterLotsByRange(lots: Lot[], range: DashboardRange, startMs: number) {
  if (range === 'live') return lots;
  return lots.filter((lot) => {
    const time = lotBusinessTime(lot);
    if (time) return time >= startMs;
    return range === '30d';
  });
}

function getRangeStartMs(range: DashboardRange, nowMs: number) {
  if (range === 'today') {
    const start = new Date(nowMs);
    start.setHours(0, 0, 0, 0);
    return start.getTime();
  }
  if (range === '7d') return nowMs - 7 * DAY_MS;
  if (range === '30d') return nowMs - 30 * DAY_MS;
  return 0;
}

function patchSnapshot(current: RoomSnapshot | null, event: AuctionEvent): RoomSnapshot | null {
  if (!current) return event.snapshot ?? current;
  if (event.snapshot) return event.snapshot;
  const next: RoomSnapshot = { ...current };
  if (event.lot) {
    if (event.lot.status === 'LOT_STATUS_LIVE' || event.lot.status === 'LOT_STATUS_EXTENDED') next.currentLot = event.lot;
    if (current.currentLot?.id === event.lot.id && event.lot.status !== 'LOT_STATUS_LIVE' && event.lot.status !== 'LOT_STATUS_EXTENDED') next.currentLot = undefined;
    next.playbookStage = event.lot.playbookStage || next.playbookStage;
  }
  if (event.ranking?.length) next.ranking = event.ranking;
  if (event.bid) next.recentBids = [event.bid, ...current.recentBids.filter((bid) => bid.id !== event.bid?.id)].slice(0, 20);
  return next;
}

function shouldRefreshBusinessData(type: string) {
  return new Set<string>([
    REALTIME_EVENT.LOT_CREATED,
    REALTIME_EVENT.LOT_STARTED,
    REALTIME_EVENT.LOT_UPDATED,
    REALTIME_EVENT.LOT_QUEUED,
    REALTIME_EVENT.LOT_SETTLED,
    REALTIME_EVENT.LOT_CANCELLED,
    REALTIME_EVENT.AUCTION_CLOSED,
    REALTIME_EVENT.ORDER_CREATED,
    REALTIME_EVENT.PAYMENT_SUCCESS,
  ]).has(type);
}

function upsertById<T extends { id: string }>(items: T[], item: T) {
  if (!items.some((current) => current.id === item.id)) return [item, ...items];
  return items.map((current) => (current.id === item.id ? item : current));
}

function isPaidOrder(order: OrderSummary) {
  return order.status === 'PAID' || order.paymentStatus === 'SUCCESS';
}

function isPendingOrder(order: OrderSummary) {
  if (isAbnormalOrder(order.status, order.paymentStatus) || isPaidOrder(order)) return false;
  return order.status === 'CREATED' || order.status === 'PENDING_PAYMENT' || order.paymentStatus === 'INIT' || order.paymentStatus === 'PROCESSING';
}

function hasStartedLot(lot: Lot) {
  return Number(lot.startedAtUnixMs || 0) > 0 || ['LOT_STATUS_LIVE', 'LOT_STATUS_EXTENDED', 'LOT_STATUS_SETTLED', 'LOT_STATUS_FAILED'].includes(lot.status);
}

function isListedLot(lot: Lot) {
  return !['LOT_STATUS_UNSPECIFIED', 'LOT_STATUS_DRAFT'].includes(lot.status);
}

function isQueueLot(lot: Lot) {
  return lot.status === 'LOT_STATUS_QUEUED' || lot.status === 'LOT_STATUS_READY';
}

function isAbnormalLot(lot: Lot) {
  return lot.status === 'LOT_STATUS_CANCELLED' || lot.status === 'LOT_STATUS_FAILED';
}

function lotHasBid(lot: Lot, snapshot: RoomSnapshot | null, orders: OrderSummary[]) {
  if ((lot.stats?.bidCount ?? 0) > 0) return true;
  if (lot.leadingUserId || lot.winnerUserId) return true;
  if (moneyCents(lot.currentPrice) > moneyCents(lot.rule.startPrice)) return true;
  if (snapshot?.recentBids.some((bid) => bid.lotId === lot.id)) return true;
  return orders.some((order) => order.lotId === lot.id);
}

function lotBusinessTime(lot: Lot) {
  return Math.max(
    Number(lot.settledAtUnixMs || 0),
    Number(lot.cancelledAtUnixMs || 0),
    Number(lot.startedAtUnixMs || 0),
    Number(lot.endsAtUnixMs || 0),
  );
}

function orderTime(order: OrderSummary) {
  return Number(order.createdAtUnixMs || order.paidAtUnixMs || order.updatedAtUnixMs || 0);
}

function sumOrderYuan(orders: OrderSummary[]) {
  return orders.reduce((sum, order) => sum + moneyYuan(order.amount), 0);
}

function moneyCents(value?: Money | number | string | null) {
  if (typeof value === 'object' && value !== null && 'amount' in value) return Number(value.amount || 0);
  return Number(value || 0);
}

function moneyYuan(value?: Money | number | string | null) {
  return moneyCents(value) / 100;
}

function calcDelta(values: number[]) {
  const usable = values.filter((value) => Number.isFinite(value));
  if (usable.length < 2 || usable.every((value) => value === 0)) return null;
  const half = Math.max(1, Math.floor(usable.length / 2));
  const first = usable.slice(0, half).reduce((sum, value) => sum + value, 0);
  const second = usable.slice(half).reduce((sum, value) => sum + value, 0);
  if (first === 0) return second > 0 ? 100 : null;
  return (second - first) / Math.abs(first) * 100;
}

function getSparklineGeometry(values: number[], width: number, height: number) {
  const safeValues = values.length ? values : [0];
  const max = Math.max(...safeValues);
  const min = Math.min(...safeValues);
  const range = max - min;
  const padding = { top: 5, right: 4, bottom: 7, left: 4 };
  const chartWidth = width - padding.left - padding.right;
  const chartHeight = height - padding.top - padding.bottom;
  const points = safeValues.map((value, index) => {
    const x = safeValues.length === 1 ? width / 2 : padding.left + (chartWidth / (safeValues.length - 1)) * index;
    const y = range === 0 ? padding.top + chartHeight * 0.58 : padding.top + (1 - (value - min) / range) * chartHeight;
    return { x: Number(x.toFixed(2)), y: Number(y.toFixed(2)) };
  });
  const linePath = createSmoothPath(points);
  const firstPoint = points[0];
  const lastPoint = points[points.length - 1];
  const baselineY = height - 4;
  return {
    linePath,
    areaPath: `${linePath} L ${lastPoint.x.toFixed(2)} ${baselineY.toFixed(2)} L ${firstPoint.x.toFixed(2)} ${baselineY.toFixed(2)} Z`,
    guideY: baselineY,
    lastPoint,
  };
}

function createSmoothPath(points: Array<{ x: number; y: number }>) {
  if (points.length === 1) return `M ${points[0].x.toFixed(2)} ${points[0].y.toFixed(2)}`;
  return points.reduce((path, point, index) => {
    if (index === 0) return `M ${point.x.toFixed(2)} ${point.y.toFixed(2)}`;
    const previous = points[index - 1];
    const beforePrevious = points[index - 2] ?? previous;
    const next = points[index + 1] ?? point;
    const controlA = {
      x: previous.x + (point.x - beforePrevious.x) / 6,
      y: previous.y + (point.y - beforePrevious.y) / 6,
    };
    const controlB = {
      x: point.x - (next.x - previous.x) / 6,
      y: point.y - (next.y - previous.y) / 6,
    };
    return `${path} C ${controlA.x.toFixed(2)} ${controlA.y.toFixed(2)}, ${controlB.x.toFixed(2)} ${controlB.y.toFixed(2)}, ${point.x.toFixed(2)} ${point.y.toFixed(2)}`;
  }, '');
}

function usePrefersReducedMotion() {
  const [reduced, setReduced] = useState(false);
  useEffect(() => {
    const query = window.matchMedia('(prefers-reduced-motion: reduce)');
    const update = () => setReduced(query.matches);
    update();
    query.addEventListener('change', update);
    return () => query.removeEventListener('change', update);
  }, []);
  return reduced;
}

function useInView<T extends HTMLElement>(threshold = 0.2, rootMargin = '0px') {
  const ref = useRef<T | null>(null);
  const [visible, setVisible] = useState(false);
  useEffect(() => {
    const node = ref.current;
    if (!node || !('IntersectionObserver' in window)) {
      setVisible(true);
      return;
    }
    const observer = new IntersectionObserver(([entry]) => {
      if (entry.isIntersecting) setVisible(true);
    }, { rootMargin, threshold });
    observer.observe(node);
    return () => observer.disconnect();
  }, [rootMargin, threshold]);
  return [ref, visible] as const;
}

function useScrollReveal<T extends HTMLElement>(triggerRatio = 0.72) {
  const [node, setNode] = useState<T | null>(null);
  const [visible, setVisible] = useState(false);
  const ref = useCallback((nextNode: T | null) => {
    setNode(nextNode);
    if (!nextNode) setVisible(false);
  }, []);

  useEffect(() => {
    if (!node || visible) return;

    const scrollTarget = findScrollParent(node);
    let animationFrame = 0;
    let revealed = false;

    const cleanup = () => {
      scrollTarget.removeEventListener('scroll', scheduleCheck);
      window.removeEventListener('resize', scheduleCheck);
      if (animationFrame) cancelAnimationFrame(animationFrame);
    };

    const revealIfReady = () => {
      if (revealed) return;
      const containerRect = scrollTarget === window ? { top: 0, height: window.innerHeight } : (scrollTarget as HTMLElement).getBoundingClientRect();
      const rect = node.getBoundingClientRect();
      const triggerY = containerRect.top + containerRect.height * triggerRatio;
      const visibleFloor = containerRect.top + containerRect.height * 0.08;
      if (rect.top <= triggerY && rect.bottom >= visibleFloor) {
        revealed = true;
        setVisible(true);
        cleanup();
      }
    };

    function scheduleCheck() {
      if (animationFrame) cancelAnimationFrame(animationFrame);
      animationFrame = requestAnimationFrame(revealIfReady);
    }

    scrollTarget.addEventListener('scroll', scheduleCheck, { passive: true });
    window.addEventListener('resize', scheduleCheck);
    return cleanup;
  }, [node, triggerRatio, visible]);

  return [ref, visible] as const;
}

function findScrollParent(node: HTMLElement): HTMLElement | Window {
  let parent = node.parentElement;
  while (parent && parent !== document.body) {
    const style = window.getComputedStyle(parent);
    const overflowY = `${style.overflowY} ${style.overflow}`;
    if (/(auto|scroll|overlay)/.test(overflowY) && parent.scrollHeight > parent.clientHeight) return parent;
    parent = parent.parentElement;
  }
  return window;
}

function easeOutCubic(progress: number) {
  return 1 - Math.pow(1 - progress, 3);
}

function formatMetricValue(value: number, format: MetricFormat) {
  if (format === 'money') return formatYuan(value);
  if (format === 'percent') return formatPercent(value);
  return Math.round(value).toLocaleString('zh-CN');
}

function formatYuan(value: number) {
  return `¥${value.toLocaleString('zh-CN', { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`;
}

function formatCompactYuan(value: number) {
  if (value >= 10000) return `¥${(value / 10000).toFixed(1)}万`;
  return `¥${Math.round(value).toLocaleString('zh-CN')}`;
}

function formatPercent(value: number) {
  return `${value.toLocaleString('zh-CN', { minimumFractionDigits: 1, maximumFractionDigits: 1 })}%`;
}

function formatOneDecimal(value: number) {
  return value.toLocaleString('zh-CN', { minimumFractionDigits: 1, maximumFractionDigits: 1 });
}

function formatCountdown(ms: number) {
  const totalSeconds = Math.max(0, Math.floor(ms / 1000));
  const hours = Math.floor(totalSeconds / 3600);
  const minutes = Math.floor((totalSeconds % 3600) / 60);
  const seconds = totalSeconds % 60;
  return hours > 0 ? `${String(hours).padStart(2, '0')}:${String(minutes).padStart(2, '0')}:${String(seconds).padStart(2, '0')}` : `${String(minutes).padStart(2, '0')}:${String(seconds).padStart(2, '0')}`;
}

function formatBucketLabel(timeMs: number, range: DashboardRange) {
  const date = new Date(timeMs);
  if (range === 'today' || range === 'live') return date.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' });
  return date.toLocaleDateString('zh-CN', { month: '2-digit', day: '2-digit' });
}

function createRiskGradient(items: Array<{ value: number; tone: StudioTone }>) {
  const total = items.reduce((sum, item) => sum + item.value, 0);
  if (!total) return DONUT_EMPTY_COLOR;
  let cursor = 0;
  const parts = items.filter((item) => item.value > 0).map((item) => {
    const start = cursor;
    cursor += item.value / total * 360;
    return `${DONUT_SEGMENT_COLORS[item.tone]} ${start}deg ${cursor}deg`;
  });
  return `conic-gradient(${parts.join(', ')})`;
}
