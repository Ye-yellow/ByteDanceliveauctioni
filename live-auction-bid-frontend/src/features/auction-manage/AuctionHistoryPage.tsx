import { useEffect, useMemo, useState } from 'react';
import { AlertTriangle, ChevronLeft, ChevronRight, FileClock, Package, RefreshCw, Search, ShieldAlert, Trophy, XCircle } from 'lucide-react';
import { listAdminLots, type AdminLotsQuery } from '../auction/api/auctionApi';
import { HISTORY_LOT_STATUS_FILTERS, isSettlementLot, lotStatusLabel, lotStatusTone } from '../../entities/auction/model/auctionStatus';
import type { Lot } from '../../shared/api/types';
import { resultMessage } from '../../shared/api/result';
import { formatDateTimeText, formatDurationText, formatMoneyText } from '../../shared/lib/format';
import { ADMIN_ROOM } from '../../shared/config/studio';
import { StudioBadge, StudioButton, StudioCard, StudioEmptyState, StudioErrorState, StudioField, StudioMetricCard, StudioPageHeader, StudioTable, StudioTableSkeleton, StudioToastViewport, useStudioToast } from '../../pages/host-console/components/studio-ui';

const DEFAULT_PAGE_SIZE = 10;

export function AuctionHistoryPage({ roomId = ADMIN_ROOM.id }: { roomId?: string }) {
  const [query, setQuery] = useState<AdminLotsQuery>({ page: 1, pageSize: DEFAULT_PAGE_SIZE, roomId, view: 'history' });
  const [lots, setLots] = useState<Lot[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const { toasts, showToast } = useStudioToast();
  const totalPages = Math.max(1, Math.ceil(total / (query.pageSize || DEFAULT_PAGE_SIZE)));
  const currentPage = query.page || 1;

  const goPrevPage = () => setQuery((c) => ({ ...c, page: Math.max(1, (c.page || 1) - 1) }));
  const goNextPage = () => setQuery((c) => ({ ...c, page: (c.page || 1) + 1 }));

  const syncLots = async (nextQuery = query) => {
    setLoading(true);
    setError('');
    try {
      const page = await listAdminLots({ ...nextQuery, roomId, view: 'history', pageSize: DEFAULT_PAGE_SIZE });
      setLots(page.lots);
      setTotal(page.total);
      setQuery((current) => ({ ...current, roomId, view: 'history', page: page.page, pageSize: DEFAULT_PAGE_SIZE }));
    } catch (e) {
      const message = resultMessage(e);
      setError(message);
      showToast({ id: 'auction-history-sync-failed', tone: 'danger', title: '历史记录同步失败', description: message });
    } finally {
      setLoading(false);
    }
  };

  const updateQuery = (patch: Partial<AdminLotsQuery>) => {
    setQuery((current) => ({ ...current, ...patch, view: 'history', page: patch.page ?? 1 }));
  };

  useEffect(() => { void syncLots({ ...query, roomId, view: 'history' }); }, [roomId]);
  useEffect(() => { void syncLots(query); }, [query.page, query.status]);

  const metrics = useMemo(() => ({
    settled: lots.filter(isSettlementLot).length,
    cancelled: lots.filter((lot) => lot.status === 'LOT_STATUS_CANCELLED').length,
    failed: lots.filter((lot) => lot.status === 'LOT_STATUS_FAILED').length,
  }), [lots]);

  return <section className="postLivePage auctionHistoryPage">
    <StudioToastViewport toasts={toasts} />
    <StudioCard padding="lg" className="postLiveHeader auctionHistoryHero">
      <StudioPageHeader
        eyebrow="Lot history"
        title="拍品历史记录"
        description="已成交、已取消和异常拍品从本场队列拆出，保留给运营复盘、订单核对和取消原因审计。"
        actions={<><a className="studioButton studioButton-secondary studioButton-md" href="/admin/auctions">返回本场队列</a><StudioButton type="button" variant="secondary" icon={<RefreshCw size={15} />} loading={loading} onClick={() => void syncLots()}>{loading ? '同步中' : '刷新历史'}</StudioButton></>}
      />
    </StudioCard>
    {error ? <div className="auctionMgmtNotice danger"><AlertTriangle size={16} />{error}</div> : null}
    <section className="postLiveMetricGrid auctionHistoryMetrics">
      <StudioMetricCard icon={<FileClock />} label="历史总数" value={total} trend="当前筛选结果" tone="info" />
      <StudioMetricCard icon={<Trophy />} label="本页成交" value={metrics.settled} trend="SETTLED / SOLD" tone="success" />
      <StudioMetricCard icon={<XCircle />} label="本页取消" value={metrics.cancelled} trend="运营撤回" tone="warning" />
      <StudioMetricCard icon={<ShieldAlert />} label="本页异常" value={metrics.failed} trend="FAILED" tone="danger" />
    </section>
    <StudioCard padding="md" className="postLiveHeader auctionHistoryFiltersCard">
      <div className="auctionFilterBar queueFilters" aria-label="拍品历史筛选">
        <label><Search size={15} /><input value={query.keyword || ''} onChange={(e) => updateQuery({ keyword: e.target.value })} onKeyDown={(e) => { if (e.key === 'Enter') void syncLots({ ...query, page: 1 }); }} placeholder="搜索拍品名 / 竞拍 ID / 取消原因" /></label>
        <StudioField label="历史状态"><select value={query.status || ''} onChange={(e) => updateQuery({ status: e.target.value as AdminLotsQuery['status'] })}>{HISTORY_LOT_STATUS_FILTERS.map((item) => <option key={item.label} value={item.value}>{item.label}</option>)}</select></StudioField>
        <StudioButton type="button" variant="primary" icon={<Search size={15} />} onClick={() => void syncLots({ ...query, page: 1 })}>查询</StudioButton>
      </div>
    </StudioCard>
    {loading ? <StudioTableSkeleton rows={DEFAULT_PAGE_SIZE} columns={7} /> : error && !lots.length ? <StudioErrorState icon={<AlertTriangle size={34} />} title="历史记录加载失败" description={error} action={<StudioButton type="button" variant="secondary" onClick={() => void syncLots()}>重试</StudioButton>} /> : <StudioTable
      className="auctionHistoryTable"
      rows={lots}
      rowKey={(lot) => lot.id}
      header={`共 ${total} 条 · 每页 ${DEFAULT_PAGE_SIZE} 条`}
      filters={
        <div className="orderPager">
          <button type="button" disabled={currentPage <= 1 || loading} onClick={goPrevPage}><ChevronLeft size={15} /><span>上一页</span></button>
          <span className="orderPagerIndex">第 {currentPage} / {totalPages} 页</span>
          <button type="button" disabled={currentPage >= totalPages || loading} onClick={goNextPage}><span>下一页</span><ChevronRight size={15} /></button>
        </div>
      }
      empty={<StudioEmptyState icon={<Package size={34} />} title="暂无历史记录" description="当前筛选条件下没有已成交、已取消或异常拍品。" action={<a className="studioButton studioButton-secondary studioButton-md" href="/admin/auctions">返回本场队列</a>} compact />}
      columns={[
        { label: '拍品', render: (lot) => <div className="historyLotCell"><img src={lot.imageUrl || '/vite.svg'} alt={lot.title} /><div><b>{lot.title}</b><span>{lot.id}</span><small>{roomId}</small></div></div> },
        { label: '状态', render: (lot) => <StudioBadge tone={lotStatusTone(lot.status)}>{lotStatusLabel(lot.status)}</StudioBadge> },
        { label: '结论时间', render: (lot) => historyTimeText(lot) },
        { label: '成交 / 当前价', render: (lot) => <strong className="moneyText">{formatMoneyText(isSettlementLot(lot) ? lot.finalPrice : lot.currentPrice)}</strong> },
        { label: '规则', render: (lot) => <div className="historyRuleCell"><span>起拍 {formatMoneyText(lot.rule.startPrice)}</span><span>加价 {formatMoneyText(lot.rule.minIncrement)}</span><span>时长 {formatDurationText(lot.rule.durationSeconds)}</span></div> },
        { label: '原因 / 买家', render: (lot) => historyReasonText(lot) },
        { label: '操作', render: (lot) => <div className="laRowActions">{isSettlementLot(lot) ? <a href="/admin/orders">成交处理</a> : null}</div> },
      ]}
    />}
  </section>;
}

function historyTimeText(lot: Lot) {
  if (lot.status === 'LOT_STATUS_CANCELLED') return formatDateTimeText(lot.cancelledAtUnixMs);
  if (isSettlementLot(lot)) return formatDateTimeText(lot.settledAtUnixMs);
  return formatDateTimeText(lot.startedAtUnixMs, '未记录');
}

function historyReasonText(lot: Lot) {
  if (lot.status === 'LOT_STATUS_CANCELLED') return <span className="historyReasonText">{lot.cancelReason || '未填写取消原因'}</span>;
  if (isSettlementLot(lot)) return <span className="historyReasonText">{lot.winnerNickname || lot.winnerUserId || '买家未同步'}</span>;
  return <span className="historyReasonText">异常状态待运营复核</span>;
}
