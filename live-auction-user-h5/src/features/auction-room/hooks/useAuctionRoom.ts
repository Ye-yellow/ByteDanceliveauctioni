import { useCallback, useEffect, useReducer, useState } from 'react';
import { getLotResult, getRoomSnapshot, listMyOrders } from '../../auction/api/auctionApi';
import { auctionRoomReducer, createInitialAuctionRoomState } from '../model/roomState';
import type { AuctionSocketEvent, Money, MyOrdersQuery, OrderSummary, PaymentSummary } from '../../../shared/api/types';

export function useAuctionRoom(roomId: string) {
  const [state, dispatch] = useReducer(auctionRoomReducer, roomId, createInitialAuctionRoomState);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  const reload = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const snapshot = await getRoomSnapshot(roomId);
      dispatch({ type: 'snapshotReceived', snapshot });
      return snapshot;
    } catch (e) {
      setError(e instanceof Error ? e.message : '房间状态加载失败');
      throw e;
    } finally {
      setLoading(false);
    }
  }, [roomId]);

  const refreshOrders = useCallback(async (query: MyOrdersQuery = { page: 1, pageSize: 20 }) => {
    const orderList = await listMyOrders(query);
    dispatch({ type: 'ordersLoaded', orders: orderList.orders });
    return orderList;
  }, []);

  const refreshLotResult = useCallback(async (lotId: string) => {
    const result = await getLotResult(lotId);
    dispatch({ type: 'lotResultLoaded', result });
    return result;
  }, []);

  const applyEvent = useCallback((event: AuctionSocketEvent) => {
    dispatch({ type: 'eventReceived', event });
  }, []);

  const markBidStarted = useCallback((lotId: string, amount: Money, idempotencyKey: string) => {
    dispatch({ type: 'localBidStarted', lotId, amount, idempotencyKey });
  }, []);

  const markBidSettled = useCallback((idempotencyKey?: string) => {
    dispatch({ type: 'localBidSettled', idempotencyKey });
  }, []);

  const markPaymentStarted = useCallback((orderId: string, idempotencyKey: string) => {
    dispatch({ type: 'localPaymentStarted', orderId, idempotencyKey });
  }, []);

  const markPaymentSettled = useCallback((order?: OrderSummary, payment?: PaymentSummary) => {
    dispatch({ type: 'localPaymentSettled', order, payment });
  }, []);

  useEffect(() => {
    const timer = window.setTimeout(() => {
      void reload().catch(() => undefined);
    }, 0);
    return () => window.clearTimeout(timer);
  }, [reload]);

  return {
    room: state,
    loading,
    error,
    reload,
    refreshOrders,
    refreshLotResult,
    applyEvent,
    markBidStarted,
    markBidSettled,
    markPaymentStarted,
    markPaymentSettled,
    dispatch,
  };
}
