import { useCallback, useEffect, useMemo, useState } from 'react';
import { AlertTriangle, ClipboardList, Clock3, Gavel, ListChecks, Package, Radio, ReceiptText, RefreshCw, ShieldAlert, Users, Wifi } from 'lucide-react';
import { getRoomPresence, getRoomSnapshot, listAdminLots, listRoomEvents } from '../auction/api/auctionApi';
import { listAdminOrders } from '../order/api/orderApi';
import { isLiveLot, isSettlementLot, lotStatusLabel, lotStatusTone } from '../../entities/auction/model/auctionStatus';
import type { OrderStatus, PaymentStatus } from '../../entities/order/model/orderStatus';
import type { AuctionEvent, Bid, Lot, RoomPresence, RoomSnapshot } from '../../shared/api/types';
import { resultMessage } from '../../shared/api/result';
import { formatDateTimeText, formatDurationText, formatMoneyText } from '../../shared/lib/format';
import { ADMIN_ROOM } from '../../shared/config/studio';
import { REALTIME_CONSOLE_EVENTS, REALTIME_EVENT } from '../../shared/realtime/events';
import { roomSocketStatusLabel, useRoomSocket } from '../../shared/realtime/useRoomSocket';
import { StudioBadge, StudioButton, StudioCard, StudioEmptyState, StudioMetricCard, StudioPageHeader, StudioTableSkeleton } from '../../pages/host-console/components/studio-ui';

type LotStats = {
  total: number;
  queued: number;
  live: number;
  settled: number;
  abnormal: number;
};

type OrderStats = {
  pending: number;
  abnormal: number;
};

const emptyLotStats: LotStats = { total: 0, queued: 0, live: 0, settled: 0, abnormal: 0 };
const emptyOrderStats: OrderStats = { pending: 0, abnormal: 0 };
const abnormalOrderStatuses: OrderStatus[] = ['CANCELLED', 'EXPIRED', 'REFUNDED'];
const abnormalPaymentStatuses: PaymentStatus[] = ['FAILED', 'CLOSED'];

export function AdminDashboardPage({ roomId = ADMIN_ROOM.id }: { roomId?: string }) {
  const [snapshot, setSnapshot] = useState<RoomSnapshot | null>(null);
  const [snapshotReceivedAt, setSnapshotReceivedAt] = useState(0);
  const [presence, setPresence] = useState<RoomPresence | null>(null);
  const [lots, setLots] = useState<Lot[]>([]);
  const [queuedLots, setQueuedLots] = useState<Lot[]>([]);
  const [events, setEvents] = useState<AuctionEvent[]>([]);
  const [lotStats, setLotStats] = useState<LotStats>(emptyLotStats);
  const [orderStats, setOrderStats] = useState<OrderStats>(emptyOrderStats);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [nowMs, setNowMs] = useState(() => Date.now());

  const commitSnapshot = useCallback((next: RoomSnapshot) => {
    setSnapshot(next);
    setSnapshotReceivedAt(Date.now());
  }, []);

  const syncLots = useCallback(async () => {
    const [inventory, queued, live, extended, settled, cancelled, failed] = await Promise.all([
      listAdminLots({ page: 1, pageSize: 100, roomId }),
      listAdminLots({ page: 1, pageSize: 8, roomId, status: 'LOT_STATUS_QUEUED' }),
      listAdminLots({ page: 1, pageSize: 1, roomId, status: 'LOT_STATUS_LIVE' }),
      listAdminLots({ page: 1, pageSize: 1, roomId, status: 'LOT_STATUS_EXTENDED' }),
      listAdminLots({ page: 1, pageSize: 1, roomId, status: 'LOT_STATUS_SETTLED' }),
      listAdminLots({ page: 1, pageSize: 1, roomId, status: 'LOT_STATUS_CANCELLED' }),
      listAdminLots({ page: 1, pageSize: 1, roomId, status: 'LOT_STATUS_FAILED' }),
    ]);
    setLots(inventory.lots);
    setQueuedLots(queued.lots);
    setLotStats({
      total: inventory.total,
      queued: queued.total,
      live: live.total + extended.total,
      settled: settled.total,
      abnormal: cancelled.total + failed.total,
    });
  }, [roomId]);

  const syncOrders = useCallback(async () => {
    const [pending, ...abnormalPages] = await Promise.all([
      listAdminOrders({ page: 1, pageSize: 1, status: 'PENDING_PAYMENT' }),
      ...abnormalOrderStatuses.map((status) => listAdminOrders({ page: 1, pageSize: 1, status })),
      ...abnormalPaymentStatuses.map((paymentStatus) => listAdminOrders({ page: 1, pageSize: 1, paymentStatus })),
    ]);
    setOrderStats({
      pending: pending.total,
      abnormal: abnormalPages.reduce((sum, page) => sum + page.total, 0),
    });
  }, []);

  const sync = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const [nextSnapshot, nextPresence, eventPage] = await Promise.all([
        getRoomSnapshot(roomId),
        getRoomPresence(roomId),
        listRoomEvents(roomId, { pageSize: 12 }),
        syncLots(),
        syncOrders(),
      ]);
      commitSnapshot(nextSnapshot);
      setPresence(nextPresence);
      setEvents(eventPage.events);
    } catch (e) {
      setError(resultMessage(e));
    } finally {
      setLoading(false);
    }
  }, [commitSnapshot, roomId, syncLots, syncOrders]);

  const refreshPresence = useCallback(() => {
    void getRoomPresence(roomId).then(setPresence).catch((e) => setError(resultMessage(e)));
  }, [roomId]);

  const upsertLot = useCallback((lot: Lot) => {
    setLots((current) => upsertById(current, lot).slice(0, 100));
    setQueuedLots((current) => (isQueuedLot(lot) ? upsertById(current, lot) : current.filter((item) => item.id !== lot.id)).slice(0, 8));
  }, []);

  const applyEvent = useCallback((event: AuctionEvent) => {
    if (event.id) {
      setEvents((current) => [event, ...current.filter((item) => item.id !== event.id)].slice(0, 12));
    }
    if (event.lot) upsertLot(event.lot);
    if (event.snapshot) {
      commitSnapshot(event.snapshot);
    } else if (event.lot || event.bid || event.ranking?.length) {
      setSnapshot((current) => patchSnapshot(current, event));
      setSnapshotReceivedAt(Date.now());
    }
    if (shouldRefreshLots(event.type)) void syncLots().catch((e) => setError(resultMessage(e)));
    if (shouldRefreshOrders(event.type)) void syncOrders().catch((e) => setError(resultMessage(e)));
  }, [commitSnapshot, syncLots, syncOrders, upsertLot]);

  const socket = useRoomSocket({
    roomId,
    handledEventTypes: REALTIME_CONSOLE_EVENTS,
    recoverSnapshot: async () => {
      const next = await getRoomSnapshot(roomId);
      commitSnapshot(next);
      return next;
    },
    onSnapshot: commitSnapshot,
    onEvent: applyEvent,
    onStatusChange: refreshPresence,
    onError: (e) => setError(resultMessage(e)),
  });

  useEffect(() => { void sync(); }, [sync]);
  useEffect(() => {
    const timer = window.setInterval(() => setNowMs(Date.now()), 1000);
    return () => window.clearInterval(timer);
  }, []);

  const currentLot = snapshot?.currentLot ?? null;
  const nextLot = queuedLots[0] ?? null;
  const serverNowMs = snapshot ? Number(snapshot.serverTimeUnixMs || 0) + Math.max(0, nowMs - snapshotReceivedAt) : nowMs;
  const remainingMs = currentLot ? Number(currentLot.endsAtUnixMs || 0) - serverNowMs : 0;
  const recentBids = useMemo(() => {
    const bids = snapshot?.recentBids ?? [];
    return [...bids].sort((a, b) => Number(b.createdAtUnixMs || 0) - Number(a.createdAtUnixMs || 0)).slice(0, 3);
  }, [snapshot]);
  const todoItems = [
    { label: '待开拍数量', value: lotStats.queued, hint: '来自本场队列' },
    { label: '成交待处理', value: orderStats.pending, hint: '待支付或待跟进订单' },
    { label: '异常待处理', value: lotStats.abnormal + orderStats.abnormal, hint: '取消、失败、退款、支付关闭' },
  ];

  return <section className="dashboardPage">
    <StudioCard padding="lg" className="dashboardWelcomeCard">
      <StudioPageHeader eyebrow="Live operations" title="今日直播工作台" description="围绕当前直播间处理本场排品、竞拍状态、成交订单和实时链路。" actions={<><a className="studioButton studioButton-primary studioButton-md" href="/admin/auctions/create">添加拍品</a><a className="studioButton studioButton-secondary studioButton-md" href="/admin/auctions">查看本场队列</a><StudioButton type="button" variant="soft" icon={<RefreshCw size={15} />} loading={loading} onClick={() => void sync()}>刷新</StudioButton></>} />
    </StudioCard>
    {error ? <div className="auctionMgmtNotice danger"><AlertTriangle size={16} />{error}</div> : null}
    <section className="todayRoomStrip">
      <div><span>当前直播间</span><b>{ADMIN_ROOM.name}</b></div>
      <div><span>roomId</span><b>{roomId}</b></div>
      <div><span>当前账号角色</span><b>主播团队工作台</b></div>
      <div><span>在线人数</span><b>{presence ? Number(presence.viewerConnections).toLocaleString('zh-CN') : '加载中'}</b></div>
      <div><span>实时同步状态</span><StudioBadge tone={socket.status === 'connected' ? 'success' : 'warning'}>{roomSocketStatusLabel(socket.status)}</StudioBadge></div>
      <div><span>最近心跳</span><b>{socket.lastEventAtText}</b></div>
    </section>

    <section className="todayCoreGrid">
      <StudioCard title="Now Live 当前竞拍" actions={<StudioBadge tone={currentLot ? 'success' : 'neutral'}>{currentLot ? lotStatusLabel(currentLot.status) : '空闲'}</StudioBadge>} className="todayAuctionCard">
        {loading ? <StudioTableSkeleton rows={4} columns={3} /> : currentLot ? <div>
          <div className="todayAuctionMedia"><span><Gavel size={30} /></span><StudioBadge tone="success">LIVE</StudioBadge></div>
          <div className="todayAuctionBody">
            <h3>{currentLot.title}</h3>
            <div className="todayCurrentPrice"><span>当前价</span><b>{formatMoneyText(currentLot.currentPrice)}</b></div>
            <div className="todayAuctionFacts">
              <span>倒计时 <b>{formatCountdown(remainingMs)}</b></span>
              <span>领先用户 <b>{currentLot.leadingNickname || currentLot.leadingUserId || '暂无'}</b></span>
              <span>在线观众 <b>{presence ? Number(presence.viewerConnections).toLocaleString('zh-CN') : '0'}</b></span>
            </div>
            <BidPreview bids={recentBids} />
          </div>
        </div> : <StudioEmptyState compact icon={<Radio size={28} />} title="当前没有正在拍" description="从本场队列选择下一件拍品开拍。" action={<a className="studioButton studioButton-primary studioButton-sm" href="/admin/auctions">进入中控台</a>} />}
      </StudioCard>

      <StudioCard title="Up Next 下一件拍品" actions={<a className="studioButton studioButton-ghost studioButton-sm" href="/admin/auctions">队列</a>} className="todayNextCard">
        {loading ? <StudioTableSkeleton rows={4} columns={3} /> : nextLot ? <>
          <div className="todayNextProduct"><span><Package size={25} /></span><div><h3>{nextLot.title}</h3><small>{nextLot.id}</small></div></div>
          <div className="todayNextRules">
            <span>起拍价 <b>{formatMoneyText(nextLot.rule.startPrice)}</b></span>
            <span>加价幅度 <b>{formatMoneyText(nextLot.rule.minIncrement)}</b></span>
            <span>预计时长 <b>{formatDurationText(nextLot.rule.durationSeconds)}</b></span>
          </div>
          <div className="todayNextActions"><a className="studioButton studioButton-primary studioButton-sm" href="/admin/auctions">设为下一件</a><StudioBadge tone="warning">等待当前结束</StudioBadge></div>
        </> : <StudioEmptyState compact icon={<Package size={28} />} title="暂无下一件拍品" description="本场队列没有待开拍拍品。" action={<a className="studioButton studioButton-primary studioButton-sm" href="/admin/auctions/create">添加拍品</a>} />}
      </StudioCard>

      <StudioCard title="Team Todo 今日待办" actions={<StudioBadge tone="info">实时</StudioBadge>} className="todayTodoCard">
        {loading ? <StudioTableSkeleton rows={4} columns={2} /> : <div className="todayTodoList">{todoItems.map((item) => <div key={item.label}><span>{item.label}</span><b>{item.value.toLocaleString('zh-CN')}</b><small>{item.hint}</small></div>)}</div>}
      </StudioCard>
    </section>

    <section className="todayMetricGrid todayPrimaryMetricGrid">
      <StudioMetricCard icon={<ListChecks />} label="今日队列" value={lotStats.total.toLocaleString('zh-CN')} trend="当前直播间拍品" tone="info" />
      <StudioMetricCard icon={<Radio />} label="正在拍" value={lotStats.live.toLocaleString('zh-CN')} trend="LIVE / EXTENDED" tone="success" />
      <StudioMetricCard icon={<ReceiptText />} label="已成交" value={lotStats.settled.toLocaleString('zh-CN')} trend="进入成交处理" tone="purple" />
      <StudioMetricCard icon={<ShieldAlert />} label="异常/待处理" value={(lotStats.abnormal + orderStats.abnormal).toLocaleString('zh-CN')} trend="拍品 + 订单" tone="danger" />
    </section>

    <section className="todayLowerGrid">
      <StudioCard title="本场拍品队列摘要" className="todayQueueCard" actions={<a className="studioButton studioButton-ghost studioButton-sm" href="/admin/auctions">查看全部</a>}>
        {loading ? <StudioTableSkeleton rows={4} columns={4} /> : lots.length ? <div className="todayQueueList">{lots.slice(0, 8).map((lot) => <div key={lot.id}><div><b>{lot.title}</b><span>{lot.id}</span></div><StudioBadge tone={lotStatusTone(lot.status)}>{lotStatusLabel(lot.status)}</StudioBadge><strong>{formatMoneyText(lot.currentPrice)}</strong></div>)}</div> : <StudioEmptyState compact icon={<Gavel size={24} />} title="暂无拍品" description="当前直播间没有拍品。" />}
      </StudioCard>
      <StudioCard title="最近出价 / 操作日志" className="todayActivityCard" actions={<StudioBadge tone={socket.status === 'connected' ? 'success' : 'warning'}>{socket.lastEventType}</StudioBadge>}>
        {loading ? <StudioTableSkeleton rows={4} columns={4} /> : events.length ? <div className="todayActivityList">{events.map((event) => <div key={event.id}><span /><div><b>{eventTitle(event)}</b><small>{eventDetail(event)} · {formatDateTimeText(event.occurredAtUnixMs)}</small></div></div>)}</div> : <StudioEmptyState compact icon={<ClipboardList size={24} />} title="暂无实时动态" description="当前直播间还没有事件记录。" />}
      </StudioCard>
    </section>

    <section className="todayMetricGrid todayOpsGrid">
      <StudioMetricCard icon={<Wifi />} label="实时中控" value={roomSocketStatusLabel(socket.status)} trend={`重连 ${socket.reconnectCount} 次`} tone={socket.status === 'connected' ? 'success' : 'warning'} />
      <StudioMetricCard icon={<Users />} label="后台连接" value={presence ? Number(presence.operatorConnections).toLocaleString('zh-CN') : '0'} trend="主播/运营/管理员" tone="info" />
      <StudioMetricCard icon={<Clock3 />} label="服务端时间" value={presence ? formatDateTimeText(presence.serverTimeUnixMs) : '未同步'} trend="presence 接口" tone="neutral" />
    </section>
  </section>;
}

function BidPreview({ bids }: { bids: Bid[] }) {
  if (!bids.length) return <div className="todayActivityList"><div><span /><div><b>暂无出价</b><small>当前拍品还没有有效出价</small></div></div></div>;
  return <div className="todayActivityList">{bids.map((bid) => <div key={bid.id}><span /><div><b>{bid.nickname || bid.userId}</b><small>{formatMoneyText(bid.amount)} · {formatDateTimeText(bid.createdAtUnixMs)}</small></div></div>)}</div>;
}

function upsertById<T extends { id: string }>(items: T[], item: T) {
  if (!items.some((current) => current.id === item.id)) return [item, ...items];
  return items.map((current) => (current.id === item.id ? item : current));
}

function patchSnapshot(current: RoomSnapshot | null, event: AuctionEvent): RoomSnapshot | null {
  if (!current) return current;
  const next: RoomSnapshot = { ...current };
  if (event.lot) {
    if (isLiveLot(event.lot)) next.currentLot = event.lot;
    if (current.currentLot?.id === event.lot.id && !isLiveLot(event.lot)) next.currentLot = undefined;
    next.playbookStage = event.lot.playbookStage || next.playbookStage;
  }
  if (event.ranking?.length) next.ranking = event.ranking;
  if (event.bid) next.recentBids = [event.bid, ...current.recentBids.filter((bid) => bid.id !== event.bid?.id)].slice(0, 20);
  return next;
}

function shouldRefreshLots(type: string) {
  return new Set<string>([
    REALTIME_EVENT.LOT_CREATED,
    REALTIME_EVENT.LOT_STARTED,
    REALTIME_EVENT.LOT_UPDATED,
    REALTIME_EVENT.LOT_QUEUED,
    REALTIME_EVENT.LOT_SETTLED,
    REALTIME_EVENT.LOT_CANCELLED,
    REALTIME_EVENT.AUCTION_CLOSED,
  ]).has(type);
}

function shouldRefreshOrders(type: string) {
  return new Set<string>([REALTIME_EVENT.ORDER_CREATED, REALTIME_EVENT.PAYMENT_SUCCESS, REALTIME_EVENT.AUCTION_CLOSED, REALTIME_EVENT.LOT_SETTLED]).has(type);
}

function isAbnormalLot(lot: Pick<Lot, 'status'>) {
  return lot.status === 'LOT_STATUS_CANCELLED' || lot.status === 'LOT_STATUS_FAILED';
}

function isQueuedLot(lot: Pick<Lot, 'status'>) {
  return lot.status === 'LOT_STATUS_QUEUED' || lot.status === 'LOT_STATUS_SCHEDULED';
}

function formatCountdown(ms: number) {
  const totalSeconds = Math.max(0, Math.floor(ms / 1000));
  const minutes = Math.floor(totalSeconds / 60);
  const seconds = totalSeconds % 60;
  return `${String(minutes).padStart(2, '0')}:${String(seconds).padStart(2, '0')}`;
}

function eventTitle(event: AuctionEvent) {
  if (event.type === REALTIME_EVENT.BID_ACCEPTED || event.type === REALTIME_EVENT.BID_OUTBID) return '收到有效出价';
  if (event.type === REALTIME_EVENT.LOT_STARTED) return '拍品已开拍';
  if (event.type === REALTIME_EVENT.LOT_SETTLED) return '拍品已落锤';
  if (event.type === REALTIME_EVENT.LOT_CANCELLED) return '拍品已取消';
  if (event.type === REALTIME_EVENT.ORDER_CREATED) return '成交订单已创建';
  if (event.type === REALTIME_EVENT.PAYMENT_SUCCESS) return '订单支付成功';
  if (event.type === REALTIME_EVENT.TRUST_REVEALED) return '讲解卡已揭示';
  return event.type.replace('AUCTION_EVENT_TYPE_', '');
}

function eventDetail(event: AuctionEvent) {
  if (event.bid) return `${event.bid.nickname || event.bid.userId} · ${formatMoneyText(event.bid.amount)}`;
  if (event.lot) return `${event.lot.title} · ${lotStatusLabel(event.lot.status)}`;
  if (event.orderId) return `订单 ${event.orderId}`;
  if (event.paymentId) return `支付 ${event.paymentId}`;
  return event.reason || event.lotId || event.roomId;
}
