import { useCallback, useEffect, useState } from 'react';
import { listMyBids, listMyOrders } from '../../auction/api/auctionApi';
import type { BidRecord, OrderList, OrderSummary, BidRecordList } from '../../../shared/api/types';

export type BuyerActivityTab = 'orders' | 'bids';

export function useBuyerActivity(ensureBuyerSession: () => Promise<unknown>) {
  const [tab, setTab] = useState<BuyerActivityTab>('orders');
  const [orders, setOrders] = useState<OrderSummary[]>([]);
  const [bids, setBids] = useState<BidRecord[]>([]);
  const [ordersMeta, setOrdersMeta] = useState<Omit<OrderList, 'orders'>>({ total: 0, page: 1, pageSize: 20 });
  const [bidsMeta, setBidsMeta] = useState<Omit<BidRecordList, 'bids'>>({ total: 0, page: 1, pageSize: 20 });
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  const loadTab = useCallback(async (nextTab: BuyerActivityTab = tab) => {
    setLoading(true);
    setError('');
    try {
      await ensureBuyerSession();
      if (nextTab === 'orders') {
        const orderList = await listMyOrders({ page: 1, pageSize: 20 });
        setOrders(orderList.orders);
        setOrdersMeta({ total: orderList.total, page: orderList.page, pageSize: orderList.pageSize });
      } else {
        const bidList = await listMyBids({ page: 1, pageSize: 20 });
        setBids(bidList.bids);
        setBidsMeta({ total: bidList.total, page: bidList.page, pageSize: bidList.pageSize });
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : '记录加载失败');
    } finally {
      setLoading(false);
    }
  }, [ensureBuyerSession, tab]);

  const switchTab = useCallback((nextTab: BuyerActivityTab) => {
    setTab(nextTab);
    if (nextTab === tab) void loadTab(nextTab);
  }, [loadTab, tab]);

  const updateOrder = useCallback((order?: OrderSummary) => {
    if (!order) return;
    setOrders((current) => [order, ...current.filter((item) => item.id !== order.id)]);
  }, []);

  useEffect(() => {
    const timer = window.setTimeout(() => {
      void loadTab(tab);
    }, 0);
    return () => window.clearTimeout(timer);
  }, [loadTab, tab]);

  return {
    tab,
    orders,
    bids,
    ordersMeta,
    bidsMeta,
    loading,
    error,
    switchTab,
    updateOrder,
    refresh: () => loadTab(tab),
  };
}
