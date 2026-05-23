import { useEffect, useMemo, useState } from 'react';
import { AlertTriangle, CircleDollarSign, Clock3, Package, ReceiptText, RefreshCw, Search, ShieldAlert } from 'lucide-react';
import { getLotResult, listAdminOrders, type AdminOrdersQuery } from '../order/api/orderApi';
import type { LotResultReply, OrderSummary } from '../../entities/order/model/orderTypes';
import { ORDER_STATUS_FILTERS, isAbnormalOrder, orderStatusLabel, orderStatusTone, paymentStatusLabel, paymentStatusTone } from '../../entities/order/model/orderStatus';
import { resultMessage } from '../../shared/api/result';
import { formatAmountText, formatDateTimeText } from '../../shared/lib/format';
import { ORDER_REFRESH_EVENTS, REALTIME_EVENT } from '../../shared/realtime/events';
import { useRoomSocket } from '../../shared/realtime/useRoomSocket';
import { StudioBadge, StudioButton, StudioCard, StudioEmptyState, StudioErrorState, StudioField, StudioMetricCard, StudioPageHeader, StudioTable, StudioTableSkeleton, StudioToastViewport, useStudioToast } from '../../pages/host-console/components/studio-ui';

type Props = {
  roomId: string;
};

const DEFAULT_PAGE_SIZE = 20;

export function OrderManagementPage({ roomId }: Props) {
  const [query, setQuery] = useState<AdminOrdersQuery>({ page: 1, pageSize: DEFAULT_PAGE_SIZE });
  const [orders, setOrders] = useState<OrderSummary[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [detail, setDetail] = useState<LotResultReply | null>(null);
  const [detailLoading, setDetailLoading] = useState(false);
  const { toasts, showToast } = useStudioToast();

  const totalPages = Math.max(1, Math.ceil(total / (query.pageSize || DEFAULT_PAGE_SIZE)));

  const syncOrders = async (nextQuery = query) => {
    setLoading(true);
    setError('');
    try {
      const page = await listAdminOrders(nextQuery);
      setOrders(page.orders);
      setTotal(page.total);
      setQuery((current) => ({ ...current, page: page.page, pageSize: page.pageSize }));
    } catch (e) {
      const message = resultMessage(e);
      setError(message);
      showToast({ id: 'admin-orders-sync-failed', tone: 'danger', title: '订单列表同步失败', description: message });
    } finally {
      setLoading(false);
    }
  };

  const updateQuery = (patch: Partial<AdminOrdersQuery>) => {
    setQuery((current) => ({ ...current, ...patch, page: patch.page ?? 1 }));
  };

  const openDetail = async (order: OrderSummary) => {
    setDetailLoading(true);
    setError('');
    try {
      const next = await getLotResult(order.lotId);
      setDetail(next);
    } catch (e) {
      const message = resultMessage(e);
      setError(message);
      showToast({ id: `order-detail-${order.id}`, tone: 'danger', title: '成交详情加载失败', description: message });
    } finally {
      setDetailLoading(false);
    }
  };

  useRoomSocket({
    roomId,
    handledEventTypes: ORDER_REFRESH_EVENTS,
    onEvent: (event) => {
      showToast({
        id: `order-refresh-${event.type}-${event.lotId || Date.now()}`,
        tone: event.type === REALTIME_EVENT.PAYMENT_SUCCESS ? 'success' : 'info',
        title: event.type === REALTIME_EVENT.PAYMENT_SUCCESS ? '支付状态已更新' : '成交状态已更新',
        description: event.lotId || '已收到房间事件，正在刷新订单列表',
      });
      void syncOrders();
    },
    onError: (e) => setError(resultMessage(e)),
  });

  useEffect(() => { void syncOrders(); }, []);
  useEffect(() => { void syncOrders(query); }, [query.page, query.status]);

  const metrics = useMemo(() => {
    const pending = orders.filter((order) => order.status === 'PENDING_PAYMENT').length;
    const paid = orders.filter((order) => order.status === 'PAID' || order.paymentStatus === 'SUCCESS').length;
    const abnormal = orders.filter((order) => isAbnormalOrder(order.status, order.paymentStatus)).length;
    return { pending, paid, abnormal };
  }, [orders]);

  return <section className="postLivePage orderReviewPage">
    <StudioToastViewport toasts={toasts} />
    <StudioCard padding="lg" className="postLiveHeader">
      <StudioPageHeader
        eyebrow="Admin orders"
        title="成交处理"
        description="订单列表来自 /api/admin/orders；成交详情通过带后台权限的 /api/lots/{lotId}/result 获取。"
        actions={<StudioButton type="button" variant="secondary" icon={<RefreshCw size={15} />} loading={loading} onClick={() => void syncOrders()}>{loading ? '同步中' : '刷新订单'}</StudioButton>}
      />
    </StudioCard>
    {error ? <div className="auctionMgmtNotice danger"><AlertTriangle size={16} />{error}</div> : null}
    <section className="postLiveMetricGrid">
      <StudioMetricCard icon={<ReceiptText />} label="订单总数" value={total} trend="P2 admin order list" tone="info" />
      <StudioMetricCard icon={<Clock3 />} label="待支付" value={metrics.pending} trend="PENDING_PAYMENT" tone="warning" />
      <StudioMetricCard icon={<CircleDollarSign />} label="已支付" value={metrics.paid} trend="PAID / SUCCESS" tone="success" />
      <StudioMetricCard icon={<ShieldAlert />} label="异常订单" value={metrics.abnormal} trend="取消 / 过期 / 支付失败" tone="danger" />
    </section>
    <StudioCard padding="md" className="postLiveHeader">
      <div className="auctionFilterBar queueFilters" aria-label="订单筛选">
        <label><Search size={15} /><input value={query.buyer || ''} onChange={(e) => updateQuery({ buyer: e.target.value })} onKeyDown={(e) => { if (e.key === 'Enter') void syncOrders({ ...query, page: 1 }); }} placeholder="搜索买家 ID / 昵称" /></label>
        <StudioField label="订单状态"><select value={query.status || ''} onChange={(e) => updateQuery({ status: e.target.value as AdminOrdersQuery['status'] })}>{ORDER_STATUS_FILTERS.map((item) => <option key={item.label} value={item.value}>{item.label}</option>)}</select></StudioField>
        <StudioField label="竞拍 ID"><input value={query.lotId || ''} onChange={(e) => updateQuery({ lotId: e.target.value })} placeholder="lotId" /></StudioField>
        <StudioButton type="button" variant="primary" icon={<Search size={15} />} onClick={() => void syncOrders({ ...query, page: 1 })}>查询</StudioButton>
      </div>
    </StudioCard>
    {loading ? <StudioTableSkeleton rows={6} columns={7} /> : error && !orders.length ? <StudioErrorState icon={<AlertTriangle size={34} />} title="订单列表加载失败" description={error} action={<StudioButton type="button" variant="secondary" onClick={() => void syncOrders()}>重试</StudioButton>} /> : <StudioTable
      className="orderReviewTable"
      rows={orders}
      rowKey={(order) => order.id}
      header={`共 ${total} 条 · 第 ${query.page || 1} / ${totalPages} 页`}
      filters={<div className="postLiveFilters"><span>分页</span><button type="button" disabled={(query.page || 1) <= 1 || loading} onClick={() => setQuery((current) => ({ ...current, page: Math.max(1, (current.page || 1) - 1) }))}>上一页</button><button type="button" disabled={(query.page || 1) >= totalPages || loading} onClick={() => setQuery((current) => ({ ...current, page: (current.page || 1) + 1 }))}>下一页</button></div>}
      empty={<StudioEmptyState icon={<Package size={34} />} title="暂无订单" description="当前筛选条件下后端没有返回成交订单。" action={<StudioButton type="button" variant="secondary" icon={<RefreshCw size={15} />} onClick={() => void syncOrders()}>重新同步</StudioButton>} compact />}
      columns={[
        { label: '拍品 / 订单', render: (order) => <div className="orderProductCell"><img src={order.lotImageUrl || '/vite.svg'} alt={order.lotTitle} /><div><b>{order.lotTitle || order.lotId}</b><span>{order.id}</span><small>{order.lotId}</small></div></div> },
        { label: '成交价', render: (order) => <strong className="moneyText">{formatAmountText(order.amount, order.currency)}</strong> },
        { label: '买家', render: (order) => <b>{order.buyerNickname || order.buyerUserId}</b> },
        { label: '创建时间', render: (order) => formatDateTimeText(order.createdAtUnixMs) },
        { label: '订单', render: (order) => <StudioBadge tone={orderStatusTone(order.status)}>{orderStatusLabel(order.status)}</StudioBadge> },
        { label: '支付', render: (order) => <StudioBadge tone={paymentStatusTone(order.paymentStatus)}>{paymentStatusLabel(order.paymentStatus)}</StudioBadge> },
        { label: '操作', render: (order) => <div className="laRowActions"><button type="button" disabled={detailLoading} onClick={() => void openDetail(order)}>成交详情</button><a href={`/admin/auctions/${order.lotId}/control`}>查看竞拍</a></div> },
      ]}
    />}
    {detail ? <OrderDetailDrawer detail={detail} loading={detailLoading} onClose={() => setDetail(null)} /> : null}
  </section>;
}

function OrderDetailDrawer({ detail, loading, onClose }: { detail: LotResultReply; loading: boolean; onClose: () => void }) {
  const order = detail.order;
  return <aside className="orderDetailDrawer">
    <StudioCard title="成交详情" subtitle={order?.id || detail.lot?.id || 'lot result'} actions={<StudioButton size="sm" variant="ghost" onClick={onClose}>关闭</StudioButton>}>
      {loading ? <StudioTableSkeleton rows={2} columns={2} /> : <>
        <div className="drawerOrderHero"><img src={order?.lotImageUrl || detail.lot?.imageUrl || '/vite.svg'} alt={order?.lotTitle || detail.lot?.title || '成交拍品'} /><div><h3>{order?.lotTitle || detail.lot?.title || '成交拍品'}</h3><strong>{order ? formatAmountText(order.amount, order.currency) : '订单不可见'}</strong><span>{order?.buyerNickname || order?.buyerUserId || detail.lot?.winnerNickname || '买家未同步'} · {formatDateTimeText(order?.createdAtUnixMs || detail.lot?.settledAtUnixMs)}</span></div></div>
        <div className="drawerInfoGrid">
          <span>订单状态<b>{orderStatusLabel(order?.status)}</b></span>
          <span>支付状态<b>{paymentStatusLabel(order?.paymentStatus)}</b></span>
          <span>竞拍状态<b>{detail.auctionState || detail.lot?.status || '未同步'}</b></span>
          <span>关联竞拍<b>{order?.lotId || detail.lot?.id || '未同步'}</b></span>
          <span>支付单号<b>{order?.paymentId || '未生成 / 不可见'}</b></span>
          <span>支付截止<b>{formatDateTimeText(order?.expiresAtUnixMs)}</b></span>
        </div>
        {order ? <StudioEmptyState compact tone="success" icon={<ReceiptText size={22} />} title="订单详情来自后端权限接口" description="后台不使用公开 WebSocket reason 中的订单或支付标识。" /> : <StudioErrorState compact icon={<AlertTriangle size={22} />} title="后端未返回订单详情" description="当前账号可能无权查看，或该成交尚未生成订单。" />}
      </>}
    </StudioCard>
  </aside>;
}
