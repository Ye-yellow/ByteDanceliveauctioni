import { useEffect, useMemo, useState } from 'react';
import { AlertTriangle, ClipboardList, Gavel, Package, Radio, ReceiptText, RefreshCw, ShieldAlert, Wifi } from 'lucide-react';
import { listAdminLots } from '../auction/api/auctionApi';
import { listAdminOrders } from '../order/api/orderApi';
import { isLiveLot, isQueueReadyLot, isSettlementLot, lotStatusLabel, lotStatusTone } from '../../entities/auction/model/auctionStatus';
import { isAbnormalOrder } from '../../entities/order/model/orderStatus';
import type { Lot } from '../../shared/api/types';
import type { OrderSummary } from '../../entities/order/model/orderTypes';
import { resultMessage } from '../../shared/api/result';
import { formatAmountText, formatMoneyText } from '../../shared/lib/format';
import { ADMIN_ROOM } from '../../shared/config/studio';
import { StudioBadge, StudioButton, StudioCard, StudioEmptyState, StudioMetricCard, StudioPageHeader, StudioTableSkeleton } from '../../pages/host-console/components/studio-ui';

export function AdminDashboardPage({ roomId = ADMIN_ROOM.id }: { roomId?: string }) {
  const [lots, setLots] = useState<Lot[]>([]);
  const [orders, setOrders] = useState<OrderSummary[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  const sync = async () => {
    setLoading(true);
    setError('');
    try {
      const [lotPage, orderPage] = await Promise.all([
        listAdminLots({ page: 1, pageSize: 8, roomId }),
        listAdminOrders({ page: 1, pageSize: 8 }),
      ]);
      setLots(lotPage.lots);
      setOrders(orderPage.orders);
    } catch (e) {
      setError(resultMessage(e));
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { void sync(); }, [roomId]);

  const metrics = useMemo(() => ({
    waiting: lots.filter(isQueueReadyLot).length,
    live: lots.filter(isLiveLot).length,
    settled: lots.filter(isSettlementLot).length,
    abnormalOrders: orders.filter((order) => isAbnormalOrder(order.status, order.paymentStatus)).length,
  }), [lots, orders]);

  return <section className="dashboardPage">
    <StudioCard padding="lg" className="dashboardWelcomeCard">
      <StudioPageHeader eyebrow="Admin management P2" title="今日直播工作台" description="围绕当前直播间查看本场排品、竞拍状态、成交处理和实时同步情况；入口已拆分到 feature 页面。" actions={<><a className="studioButton studioButton-primary studioButton-md" href="/admin/auctions/create">添加拍品</a><a className="studioButton studioButton-secondary studioButton-md" href="/admin/auctions">查看本场队列</a><StudioButton type="button" variant="soft" icon={<RefreshCw size={15} />} loading={loading} onClick={() => void sync()}>刷新</StudioButton></>} />
    </StudioCard>
    {error ? <div className="auctionMgmtNotice danger"><AlertTriangle size={16} />{error}</div> : null}
    <section className="todayRoomStrip"><div><span>当前直播间</span><b>{ADMIN_ROOM.name}</b></div><div><span>roomId</span><b>{roomId}</b></div><div><span>实时同步状态</span><StudioBadge tone="success">RoomSocket</StudioBadge></div><div><span>平均延迟</span><b>{ADMIN_ROOM.latency}</b></div></section>
    <section className="todayMetricGrid">
      <StudioMetricCard icon={<Package />} label="本页拍品" value={lots.length} trend="/api/admin/lots" tone="info" />
      <StudioMetricCard icon={<Radio />} label="正在拍" value={metrics.live} trend="LIVE / EXTENDED" tone="success" />
      <StudioMetricCard icon={<ReceiptText />} label="最近订单" value={orders.length} trend="/api/admin/orders" tone="purple" />
      <StudioMetricCard icon={<ShieldAlert />} label="异常订单" value={metrics.abnormalOrders} trend="需核对支付/取消" tone="danger" />
    </section>
    <section className="todayLowerGrid">
      <StudioCard title="本场拍品队列摘要" actions={<a className="studioButton studioButton-ghost studioButton-sm" href="/admin/auctions">查看全部</a>}>
        {loading ? <StudioTableSkeleton rows={4} columns={4} /> : lots.length ? <div className="todayQueueList">{lots.map((lot) => <div key={lot.id}><div><b>{lot.title}</b><span>{lot.id}</span></div><StudioBadge tone={lotStatusTone(lot.status)}>{lotStatusLabel(lot.status)}</StudioBadge><strong>{formatMoneyText(lot.currentPrice)}</strong></div>)}</div> : <StudioEmptyState compact icon={<Gavel size={24} />} title="暂无拍品" description="当前直播间还没有返回拍品。" />}
      </StudioCard>
      <StudioCard title="最近成交订单" actions={<a className="studioButton studioButton-ghost studioButton-sm" href="/admin/orders">成交处理</a>}>
        {loading ? <StudioTableSkeleton rows={4} columns={4} /> : orders.length ? <div className="todayActivityList">{orders.map((order) => <div key={order.id}><span /><div><b>{order.lotTitle || order.lotId}</b><small>{formatAmountText(order.amount, order.currency)} · {order.buyerNickname || order.buyerUserId} · {order.status}</small></div></div>)}</div> : <StudioEmptyState compact icon={<ClipboardList size={24} />} title="暂无订单" description="最近订单列表为空。" />}
      </StudioCard>
    </section>
    <section className="todayMetricGrid">
      <StudioMetricCard icon={<Wifi />} label="实时中控" value="已拆分" trend="/admin/realtime + /control" tone="success" />
      <StudioMetricCard icon={<Gavel />} label="待开拍" value={metrics.waiting} trend="可从队列开拍" tone="warning" />
      <StudioMetricCard icon={<ReceiptText />} label="已成交拍品" value={metrics.settled} trend="进入成交处理" tone="info" />
    </section>
  </section>;
}
