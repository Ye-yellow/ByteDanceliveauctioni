import { useEffect, useMemo, useState } from 'react';
import { AlertTriangle, CheckCircle2, ChevronLeft, ChevronRight, Clock3, Gavel, Package, Radio, RefreshCw, Search, ShieldAlert, ShieldCheck, Trophy, Wifi } from 'lucide-react';
import { cancelLot, getRoomSnapshot, listAdminLots, settleLot, startLot, type AdminLotsQuery } from '../auction/api/auctionApi';
import { CURRENT_LOT_STATUS_FILTERS, isLiveLot, isPreStartCancellableLot, isQueueReadyLot, isRemovedFromCurrentQueueLot, isSettlementLot, lotStatusLabel, lotStatusTone, settlementOutcomeDisplay, uiStatusOfLot } from '../../entities/auction/model/auctionStatus';
import type { Lot, RoomSnapshot } from '../../shared/api/types';
import { resultMessage } from '../../shared/api/result';
import { formatDateTimeText, formatDurationText, formatMoneyText } from '../../shared/lib/format';
import { getLotLeftMs, formatAuctionLeftMs } from '../../shared/lib/time';
import { AUCTION_REFRESH_EVENTS } from '../../shared/realtime/events';
import { roomSocketStatusLabel, useRoomSocket } from '../../shared/realtime/useRoomSocket';
import { StudioBadge, StudioButton, StudioCard, StudioEmptyState, StudioErrorState, StudioField, StudioMetricCard, StudioPageHeader, StudioTableSkeleton, StudioToastViewport, useStudioToast } from '../../pages/host-console/components/studio-ui';

const DEFAULT_PAGE_SIZE = 5;

type Props = {
  roomId: string;
  roomName?: string;
};

export function AuctionManagementPage({ roomId, roomName = roomId }: Props) {
  const [query, setQuery] = useState<AdminLotsQuery>({ page: 1, pageSize: DEFAULT_PAGE_SIZE, roomId, view: 'current' });
  const [lots, setLots] = useState<Lot[]>([]);
  const [total, setTotal] = useState(0);
  const [snapshot, setSnapshot] = useState<RoomSnapshot | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [selectedLot, setSelectedLot] = useState<Lot | null>(null);
  const [cancelTarget, setCancelTarget] = useState<Lot | null>(null);
  const { toasts, showToast } = useStudioToast();
  const totalPages = Math.max(1, Math.ceil(total / (query.pageSize || DEFAULT_PAGE_SIZE)));

  const syncLots = async (nextQuery = query) => {
    setLoading(true);
    setError('');
    try {
      const [page, nextSnapshot] = await Promise.all([
        listAdminLots({ ...nextQuery, roomId, view: 'current', pageSize: DEFAULT_PAGE_SIZE }),
        getRoomSnapshot(roomId),
      ]);
      setLots(page.lots);
      setTotal(page.total);
      setSnapshot(nextSnapshot);
      setQuery((current) => ({ ...current, roomId, view: 'current', page: page.page, pageSize: DEFAULT_PAGE_SIZE }));
    } catch (e) {
      const message = resultMessage(e);
      setError(message);
      showToast({ id: 'admin-lots-sync-failed', tone: 'danger', title: '本场队列同步失败', description: message });
    } finally {
      setLoading(false);
    }
  };

  const updateQuery = (patch: Partial<AdminLotsQuery>) => {
    setQuery((current) => ({ ...current, ...patch, page: patch.page ?? 1 }));
  };

  const startAuction = async (lot: Lot) => {
    setError('');
    try {
      const updated = await startLot(lot.id);
      setLots((current) => upsertLot(current, updated));
      showToast({ tone: 'success', title: '竞拍已开始', description: updated.title });
      await syncLots();
    } catch (e) {
      const message = resultMessage(e);
      setError(message);
      showToast({ tone: 'danger', title: '开始竞拍失败', description: message });
    }
  };

  const settleAuction = async (lot: Lot) => {
    setError('');
    try {
      const updated = await settleLot(lot.id);
      setLots((current) => upsertLot(current, updated));
      showToast({ tone: 'success', title: '已请求落锤成交', description: updated.title });
      await syncLots();
    } catch (e) {
      const message = resultMessage(e);
      setError(message);
      showToast({ tone: 'danger', title: '落锤成交失败', description: message });
    }
  };

  const confirmCancel = async (lot: Lot, reason: string) => {
    setError('');
    try {
      const updated = await cancelLot(lot.id, reason);
      setLots((current) => current.filter((item) => item.id !== updated.id));
      setTotal((current) => Math.max(0, current - 1));
      setCancelTarget(null);
      showToast({ tone: 'success', title: '拍品已取消', description: updated.title });
      await syncLots();
    } catch (e) {
      const message = resultMessage(e);
      setError(message);
      showToast({ tone: 'danger', title: '取消拍品失败', description: message });
    }
  };

  const socket = useRoomSocket({
    roomId,
    handledEventTypes: AUCTION_REFRESH_EVENTS,
    recoverSnapshot: async () => {
      const next = await getRoomSnapshot(roomId);
      setSnapshot(next);
      return next;
    },
    onEvent: (event) => {
      if (event.snapshot) setSnapshot(event.snapshot);
      if (event.lot) setLots((current) => upsertCurrentLot(current, event.lot as Lot));
      if (AUCTION_REFRESH_EVENTS.has(event.type)) void syncLots();
    },
    onSnapshot: (nextSnapshot) => setSnapshot(nextSnapshot),
    onError: (e) => setError(resultMessage(e)),
  });

  useEffect(() => { void syncLots({ ...query, roomId, view: 'current' }); }, [roomId]);
  useEffect(() => { void syncLots(query); }, [query.page, query.status]);

  const currentLot = snapshot?.currentLot || lots.find(isLiveLot) || null;
  const nextLot = lots.find(isQueueReadyLot) || null;
  const wsState = roomSocketStatusLabel(socket.status);
  const currentPage = query.page || 1;
  const pageStart = total ? ((currentPage - 1) * DEFAULT_PAGE_SIZE) + 1 : 0;
  const pageEnd = total ? Math.min(total, currentPage * DEFAULT_PAGE_SIZE) : 0;
  const metrics = useMemo(() => ({
    waiting: lots.filter(isQueueReadyLot).length,
    live: lots.filter(isLiveLot).length,
    settled: lots.filter(isSettlementLot).length,
    abnormal: lots.filter((lot) => lot.status === 'LOT_STATUS_FAILED').length,
  }), [lots]);

  return <section className="auctionMgmtPage">
    <StudioToastViewport toasts={toasts} />
    <StudioCard padding="lg" className="auctionMgmtHeader">
      <StudioPageHeader
        eyebrow="Admin lots"
        title="本场拍品队列"
        description="用于主播和运营团队管理本场开拍节奏、未开拍拍品取消和成交处理；队列按 5 件一页聚焦展示。"
        actions={<><a className="studioButton studioButton-primary studioButton-md" href="/admin/auctions/create">添加拍品</a><a className="studioButton studioButton-secondary studioButton-md" href="/admin/auctions/history">历史记录</a><StudioButton type="button" variant="secondary" icon={<RefreshCw size={15} />} loading={loading} onClick={() => void syncLots()}>{loading ? '同步中' : '同步队列'}</StudioButton></>}
      />
    </StudioCard>
    {error ? <div className="auctionMgmtNotice danger"><AlertTriangle size={16} />{error}</div> : null}
    <section className="realtimeSyncCapsule">
      <div><Wifi size={18} /><span>实时同步状态</span><StudioBadge tone={socket.status === 'connected' ? 'success' : socket.status === 'reconnecting' ? 'warning' : 'danger'}>{wsState}</StudioBadge></div>
      <div className="syncCapsuleMetrics"><span>当前直播间：<b>{roomId}</b></span><span>最近心跳：<b>{new Date().toLocaleTimeString('zh-CN', { hour12: false })}</b></span><span>重连次数：<b>{socket.reconnectCount}</b></span><span>当前竞拍：<b>{currentLot?.id || '无 LIVE'}</b></span></div>
      <div><button type="button" onClick={() => void syncLots()}>重新同步</button><a href="/admin/realtime">实时诊断</a></div>
    </section>
    <section className="queueTopCards">
      <QueueFocusCard lot={currentLot} snapshot={snapshot} onCancel={setCancelTarget} />
      <NextLotCard lot={nextLot} disabled={Boolean(currentLot)} onStart={startAuction} />
      <article className="queueTopCard health"><header><span><ShieldCheck size={18} />队列健康</span><StudioBadge tone={socket.status === 'connected' ? 'success' : 'warning'}>{wsState}</StudioBadge></header><div className="queueHealthGrid"><span>返回总数：<b>{total}</b></span><span>本页待拍：<b>{metrics.waiting}</b></span><span>本页落锤：<b>{metrics.settled}</b></span><span>本页异常：<b>{metrics.abnormal}</b></span></div><p>当前直播间 {roomName}</p></article>
    </section>
    <section className="auctionMgmtStats">
      <StudioMetricCard icon={<Clock3 />} label="待开拍" value={metrics.waiting} trend="READY / QUEUED" tone="info" />
      <StudioMetricCard icon={<Radio />} label="进行中" value={metrics.live} trend="LIVE / EXTENDED" tone="success" />
      <StudioMetricCard icon={<Trophy />} label="已落锤" value={metrics.settled} trend="SETTLED" tone="warning" />
      <StudioMetricCard icon={<ShieldAlert />} label="异常" value={metrics.abnormal} trend="FAILED，取消进历史" tone="danger" />
    </section>
    <StudioCard padding="md" className="queueToolbarCard">
      <div className="queueToolbarHeader">
        <div><strong>队列检索</strong><span>每页 5 条，优先保证主播开拍前的扫视和操作准确性。</span></div>
        <div className="queuePageMeta"><span>共 <b>{total}</b> 条</span><span>显示 <b>{pageStart}-{pageEnd}</b></span><span>第 <b>{currentPage}</b> / {totalPages} 页</span></div>
      </div>
      <div className="auctionFilterBar queueFilters" aria-label="拍品队列筛选">
        <label><Search size={15} /><input value={query.keyword || ''} onChange={(e) => updateQuery({ keyword: e.target.value })} onKeyDown={(e) => { if (e.key === 'Enter') void syncLots({ ...query, page: 1 }); }} placeholder="搜索拍品名 / 竞拍 ID" /></label>
        <StudioField label="状态"><select value={query.status || ''} onChange={(e) => updateQuery({ status: e.target.value as AdminLotsQuery['status'] })}>{CURRENT_LOT_STATUS_FILTERS.map((item) => <option key={item.label} value={item.value}>{item.label}</option>)}</select></StudioField>
        <StudioButton type="button" variant="primary" icon={<Search size={15} />} onClick={() => void syncLots({ ...query, page: 1 })}>查询</StudioButton>
        <div className="queuePager" aria-label="拍品队列分页">
          <button type="button" disabled={currentPage <= 1 || loading} onClick={() => setQuery((current) => ({ ...current, page: Math.max(1, (current.page || 1) - 1) }))}><ChevronLeft size={15} /><span>上一页</span></button>
          <button type="button" disabled={currentPage >= totalPages || loading} onClick={() => setQuery((current) => ({ ...current, page: (current.page || 1) + 1 }))}><span>下一页</span><ChevronRight size={15} /></button>
        </div>
      </div>
    </StudioCard>
    {loading ? <StudioTableSkeleton className="auctionMgmtSkeleton" rows={DEFAULT_PAGE_SIZE} columns={7} /> : error && !lots.length ? <StudioErrorState className="auctionMgmtEmpty" icon={<AlertTriangle size={40} />} title="本场拍品队列加载失败" description={error} action={<StudioButton type="button" variant="secondary" icon={<RefreshCw size={15} />} onClick={() => void syncLots()}>重试加载</StudioButton>} /> : lots.length ? <section className="auctionQueueList" aria-label="本场拍品队列列表">{lots.map((lot, index) => <AuctionQueueRow key={lot.id} lot={lot} position={((currentPage - 1) * DEFAULT_PAGE_SIZE) + index + 1} currentLot={currentLot} snapshot={snapshot} onDetail={setSelectedLot} onCancel={setCancelTarget} onStart={startAuction} onSettle={settleAuction} />)}</section> : <StudioEmptyState icon={<Package size={34} />} title="暂无拍品" description="当前筛选条件下没有拍品，可以添加新拍品或清空筛选。" action={<a className="studioButton studioButton-primary studioButton-md" href="/admin/auctions/create">添加拍品</a>} />}
    {selectedLot ? <AuctionDetailDrawer lot={selectedLot} snapshot={snapshot} onClose={() => setSelectedLot(null)} /> : null}
    {cancelTarget ? <CancelAuctionDialog lot={cancelTarget} onClose={() => setCancelTarget(null)} onConfirm={confirmCancel} /> : null}
  </section>;
}

function upsertLot(list: Lot[], lot: Lot) {
  return list.some((item) => item.id === lot.id) ? list.map((item) => item.id === lot.id ? lot : item) : [lot, ...list];
}

function upsertCurrentLot(list: Lot[], lot: Lot) {
  if (isRemovedFromCurrentQueueLot(lot)) return list.filter((item) => item.id !== lot.id);
  return upsertLot(list, lot);
}

function QueueFocusCard({ lot, snapshot, onCancel }: { lot: Lot | null; snapshot: RoomSnapshot | null; onCancel: (lot: Lot) => void }) {
  if (!lot) return <article className="queueTopCard current isEmpty"><header><span><Radio size={18} />当前竞拍</span><StudioBadge tone="neutral">空闲</StudioBadge></header><h3>当前没有正在拍</h3><p>可以从下一件拍品开始，或继续完善今日队列。</p><a className="studioButton studioButton-primary studioButton-sm" href="/admin/auctions/create">添加拍品</a></article>;
  return <article className="queueTopCard current isLive"><header><span><Radio size={18} />当前竞拍</span><StudioBadge tone={lotStatusTone(lot.status)}>{lotStatusLabel(lot.status)}</StudioBadge></header><h3>{lot.title}</h3><div className="queuePriceLine"><span>当前价</span><b>{formatMoneyText(lot.currentPrice)}</b><small>{formatAuctionLeftMs(getLotLeftMs(lot, snapshot?.serverTimeUnixMs), 'queue')}</small></div><p>领先用户：{lot.leadingNickname || '暂无'} · 出价 {snapshot?.recentBids?.length || 0} 次</p><div className="queueTopActions"><a className="studioButton studioButton-primary studioButton-sm" href={`/admin/auctions/${lot.id}/control`}>进入中控台</a></div></article>;
}

function NextLotCard({ lot, disabled, onStart }: { lot: Lot | null; disabled: boolean; onStart: (lot: Lot) => void }) {
  if (!lot) return <article className="queueTopCard next isEmpty"><header><span><Gavel size={18} />下一件拍品</span><StudioBadge tone="neutral">暂无</StudioBadge></header><h3>待开拍为空</h3><p>添加拍品后会进入本场队列。</p></article>;
  return <article className="queueTopCard next hasNext"><header><span><Gavel size={18} />下一件拍品</span><StudioBadge tone={lotStatusTone(lot.status)}>{lotStatusLabel(lot.status)}</StudioBadge></header><h3>{lot.title}</h3><div className="queueRulePills"><span>起拍：<b>{formatMoneyText(lot.rule.startPrice)}</b></span><span>加价：<b>{formatMoneyText(lot.rule.minIncrement)}</b></span><span>封顶：<b>{formatMoneyText(lot.rule.capPrice)}</b></span></div><p>预计 {formatDurationText(lot.rule.durationSeconds)} · 等待运营确认开拍节奏</p><div className="queueTopActions"><button type="button" className="studioButton studioButton-secondary studioButton-sm" disabled={disabled} onClick={() => void onStart(lot)}>{disabled ? '等待当前结束' : '开始竞拍'}</button></div></article>;
}

function AuctionQueueRow({ lot, position, currentLot, snapshot, onDetail, onCancel, onStart, onSettle }: { lot: Lot; position: number; currentLot: Lot | null; snapshot: RoomSnapshot | null; onDetail: (lot: Lot) => void; onCancel: (lot: Lot) => void; onStart: (lot: Lot) => void; onSettle: (lot: Lot) => void }) {
  const status = uiStatusOfLot(lot);
  const isCurrent = Boolean(currentLot?.id === lot.id || isLiveLot(lot));
  const isNext = !isCurrent && isQueueReadyLot(lot);
  return <article className={`queueRowCard ${isCurrent ? 'isCurrent' : ''} ${isNext ? 'isNext' : ''} ${status === '已取消' ? 'isCancelled' : ''}`} onClick={() => onDetail(lot)}>
    <div className="queueRowLeft"><span className="queueNo">#{String(position).padStart(2, '0')}</span><img src={lot.imageUrl || '/vite.svg'} alt={lot.title} /><div><h3>{lot.title}</h3><div className="queueTags"><StudioBadge tone={lotStatusTone(lot.status)}>{lotStatusLabel(lot.status)}</StudioBadge><span>竞拍 ID {lot.id}</span><span>规则 v{lot.version || 1}</span></div></div></div>
    <div className="queueRowMiddle"><span><b>状态进度：</b>{statusProgressText(lot, snapshot)}</span><span><b>开拍时间：</b>{formatDateTimeText(lot.startedAtUnixMs, '未开拍')}</span><span><b>起拍 / 加价：</b>{formatMoneyText(lot.rule.startPrice)} / {formatMoneyText(lot.rule.minIncrement)}</span><span><b>封顶 / 时长：</b>{formatMoneyText(lot.rule.capPrice)} / {formatDurationText(lot.rule.durationSeconds)}</span></div>
    <div className="queueRowRight">{orderStateText(lot)}<div className="auctionRowActions" onClick={(e) => e.stopPropagation()}><button type="button" className="queueActionPlain" onClick={() => onDetail(lot)}>详情</button>{isQueueReadyLot(lot) ? <button type="button" className="queueActionPrimary" disabled={Boolean(currentLot)} onClick={() => void onStart(lot)}>开始竞拍</button> : null}{isLiveLot(lot) ? <><a className="queueActionPrimary" href={`/admin/auctions/${lot.id}/control`}>进入中控</a><button type="button" className="queueActionPrimary" onClick={() => void onSettle(lot)}>落锤成交</button></> : null}{isPreStartCancellableLot(lot) ? <button type="button" className="queueActionDanger danger" onClick={() => onCancel(lot)}>取消拍品</button> : null}{isSettlementLot(lot) ? <a className="queueActionPrimary" href="/admin/orders">成交处理</a> : null}</div></div>
  </article>;
}

function statusProgressText(lot: Lot, snapshot: RoomSnapshot | null) {
  if (isLiveLot(lot)) return `倒计时 ${formatAuctionLeftMs(getLotLeftMs(lot, snapshot?.serverTimeUnixMs), 'queue')}`;
  if (isSettlementLot(lot)) return `落锤时间 ${formatDateTimeText(lot.settledAtUnixMs)}`;
  if (lot.status === 'LOT_STATUS_CANCELLED') return lot.cancelReason || '已取消';
  return `状态 ${lotStatusLabel(lot.status)}`;
}

function orderStateText(lot: Lot) {
  if (isSettlementLot(lot)) {
    const outcome = settlementOutcomeDisplay(lot);
    return <div className={`orderState ${outcome.state === 'failed' ? 'danger' : ''}`}><b>{outcome.label}</b><span>{outcome.priceLabel} {formatMoneyText(lotResultMoney(lot))}</span></div>;
  }
  if (isLiveLot(lot)) return <div className="orderState"><b>等待成交</b><span>{lot.leadingNickname || '暂无领先用户'}</span></div>;
  if (lot.status === 'LOT_STATUS_CANCELLED') return <div className="orderState danger"><b>取消原因</b><span>{lot.cancelReason || '已取消'}</span></div>;
  return <span className="mutedText">未成交</span>;
}

function drawerPersonText(lot: Lot) {
  if (isLiveLot(lot)) return `领先用户：${lot.leadingNickname || '暂无'}`;
  if (isSettlementLot(lot)) {
    const outcome = settlementOutcomeDisplay(lot);
    return `${outcome.personLabel}：${lot.winnerNickname || lot.winnerUserId || '买家未同步'}`;
  }
  if (lot.status === 'LOT_STATUS_CANCELLED') return `取消原因：${lot.cancelReason || '已取消'}`;
  return '状态同步中';
}

function drawerPrimaryPrice(lot: Lot) {
  return lotResultMoney(lot);
}

function lotResultMoney(lot: Lot) {
  if (isSettlementLot(lot) && Number(lot.finalPrice?.amount || 0) > 0) return lot.finalPrice;
  if (Number(lot.currentPrice?.amount || 0) > 0) return lot.currentPrice;
  return lot.rule.startPrice;
}

function AuctionDetailDrawer({ lot, snapshot, onClose }: { lot: Lot; snapshot: RoomSnapshot | null; onClose: () => void }) {
  const bids = snapshot?.currentLot?.id === lot.id ? snapshot.recentBids : [];
  const outcome = isSettlementLot(lot) ? settlementOutcomeDisplay(lot) : null;
  return <aside className="auctionDrawer"><div className="drawerMask" onClick={onClose} /><section><header><div><p>竞拍详情</p><h3>{lot.title}</h3><span>{lot.id}</span></div><button type="button" onClick={onClose}>关闭</button></header><div className="drawerOverview"><StudioBadge tone={outcome?.tone ?? lotStatusTone(lot.status)}>{outcome?.label ?? lotStatusLabel(lot.status)}</StudioBadge><b>{formatMoneyText(drawerPrimaryPrice(lot))}</b><p>{lot.description || '暂无描述'}</p><span>{drawerPersonText(lot)}</span><span>规则版本：v{lot.version || 1}</span></div><div className="ruleSnapshotGrid">{[['起拍价', formatMoneyText(lot.rule.startPrice)], ['加价幅度', formatMoneyText(lot.rule.minIncrement)], ['竞拍时长', formatDurationText(lot.rule.durationSeconds)], ['封顶价', formatMoneyText(lot.rule.capPrice)], ['延时窗口', `${lot.rule.antiSnipeWindowSeconds}s`], ['最大延时', `${lot.rule.maxExtendCount}`]].map(([label, value]) => <div key={label}><span>{label}</span><b>{value}</b></div>)}</div><div className="drawerBidList">{bids.length ? bids.map((bid) => <div key={bid.id}><span>{bid.nickname || bid.userId}</span><b>{formatMoneyText(bid.amount)}</b><small>{formatDateTimeText(bid.createdAtUnixMs)} · {bid.userId === lot.leadingUserId ? '领先' : '非领先'}</small></div>) : <StudioEmptyState compact icon={<CheckCircle2 size={22} />} title="暂无实时出价" description="当前房间快照没有 recentBids。" />}</div></section></aside>;
}

function CancelAuctionDialog({ lot, onClose, onConfirm }: { lot: Lot; onClose: () => void; onConfirm: (lot: Lot, reason: string) => Promise<void> }) {
  const [reason, setReason] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const submit = async () => {
    if (!reason.trim()) return;
    setSubmitting(true);
    try { await onConfirm(lot, reason.trim()); } finally { setSubmitting(false); }
  };
  return <div className="cancelDialog"><div onClick={onClose} /><section><header><AlertTriangle size={22} /><div><h3>取消拍品</h3><p>仅未开拍或草稿拍品可取消，取消后会写入后端并广播当前直播间。</p></div></header><b>{lot.title}</b><textarea value={reason} onChange={(e) => setReason(e.target.value)} rows={4} placeholder="请输入取消原因，例如误加入队列、资料需要重填" /><footer><button type="button" onClick={onClose}>返回</button><button type="button" className="danger" disabled={!reason.trim() || submitting} onClick={() => void submit()}>{submitting ? '提交中...' : '确认取消拍品'}</button></footer></section></div>;
}
