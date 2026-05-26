import { useCallback, useEffect, useMemo, useState } from 'react';
import { isBiddableLotStatus, isPrivateRefreshEventType, isSettlementEventType, lotIdFromPublicEvent } from '../../../entities/auction/model/status';
import { ownOrderForLot } from '../../../entities/order/model/privacy';
import { listRoomLots, placeBid } from '../../auction/api/auctionApi';
import { createIdempotencyKey } from '../../../shared/lib/idempotency';
import { normalizeMoney } from '../../../shared/api/adapters';
import { AuthExpiredError } from '../../../shared/api/errors';
import { AUCTION_EVENT_TYPE, type AuctionSocketEvent, type Lot, type OrderSummary, type PaymentSummary } from '../../../shared/api/types';
import { normalizeBuyerUsername, validateBuyerCredentials } from '../../../shared/auth/credentialRules';
import { useAuthSession } from '../../../shared/auth/useAuthSession';
import { DEFAULT_DEMO_ROOM_PROFILE, getDemoRoomProfile } from '../../../shared/config/demoRooms';
import { noticeForAuctionEvent } from '../model/notices';
import { useAuctionRoom } from './useAuctionRoom';
import { useAuctionSocket } from './useAuctionSocket';

const DEFAULT_ACTIVITY_QUERY = { page: 1, pageSize: 20 };
const DEPOSIT_CONFIRM_STORAGE_PREFIX = 'live-auction-h5.deposit-confirmed.v1';

export type AuctionPanelTab = 'current' | 'queue' | 'mine';
type DepositPrompt = {
  lot: Lot;
  bidAmount: number;
  userId: string;
};

function bidFailureMessage(reason: unknown): string {
  const message = reason instanceof Error ? reason.message : typeof reason === 'string' ? reason : '请稍后重试';
  return message.startsWith('出价失败') ? message : `出价失败：${message}`;
}

function shouldOpenBuyerAuth(reason: unknown): boolean {
  if (reason instanceof AuthExpiredError) return true;
  const message = reason instanceof Error ? reason.message : typeof reason === 'string' ? reason : '';
  return message.includes('请先登录') || message.includes('登录已过期') || message.includes('登录会话已失效') || message.includes('登录凭证无效');
}

function nicknameFromUsername(username: string): string {
  const normalized = username.trim();
  return normalized ? `买家${normalized.slice(0, 8)}` : `买家${Date.now().toString().slice(-4)}`;
}

function eventMayChangeLotList(event: AuctionSocketEvent): boolean {
  return event.type === AUCTION_EVENT_TYPE.LOT_CREATED ||
    event.type === AUCTION_EVENT_TYPE.LOT_QUEUED ||
    event.type === AUCTION_EVENT_TYPE.LOT_STARTED ||
    event.type === AUCTION_EVENT_TYPE.LOT_UPDATED ||
    event.type === AUCTION_EVENT_TYPE.AUCTION_EXTENDED ||
    event.type === AUCTION_EVENT_TYPE.AUCTION_CLOSED ||
    event.type === AUCTION_EVENT_TYPE.LOT_SETTLED ||
    event.type === AUCTION_EVENT_TYPE.LOT_CANCELLED ||
    event.type === AUCTION_EVENT_TYPE.ORDER_CREATED ||
    event.type === AUCTION_EVENT_TYPE.PAYMENT_SUCCESS;
}

function depositConfirmKey(roomId: string, lotId: string, userId: string): string {
  return `${DEPOSIT_CONFIRM_STORAGE_PREFIX}:${roomId}:${lotId}:${userId}`;
}

function hasConfirmedDeposit(roomId: string, lotId: string, userId: string): boolean {
  try {
    return localStorage.getItem(depositConfirmKey(roomId, lotId, userId)) === '1';
  } catch {
    return false;
  }
}

function markDepositConfirmed(roomId: string, lotId: string, userId: string) {
  try {
    localStorage.setItem(depositConfirmKey(roomId, lotId, userId), '1');
  } catch {
    // Local confirmation is best-effort in the H5 demo flow.
  }
}

export function useLiveRoomController(roomId: string) {
  const { user, status, reason, authMode, ensureReadyForBid, loginBuyer, registerBuyer, resetBuyerPassword } = useAuthSession();
  const {
    room,
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
  } = useAuctionRoom(roomId);

  const [notices, setNotices] = useState<string[]>([]);
  const [bidError, setBidError] = useState('');
  const [bidding, setBidding] = useState(false);
  const [authFormMode, setAuthFormMode] = useState<'login' | 'register' | 'reset'>('login');
  const [authUsername, setAuthUsername] = useState('');
  const [authPassword, setAuthPassword] = useState('');
  const [authNickname, setAuthNickname] = useState('');
  const [authBusy, setAuthBusy] = useState(false);
  const [authError, setAuthError] = useState('');
  const [resultLot, setResultLot] = useState<Lot | null>(null);
  const [resultOrder, setResultOrder] = useState<OrderSummary | null>(null);
  const [payOrder, setPayOrder] = useState<OrderSummary | null>(null);
  const [depositPrompt, setDepositPrompt] = useState<DepositPrompt | null>(null);
  const [auctionPanelOpen, setAuctionPanelOpen] = useState(false);
  const [auctionPanelTab, setAuctionPanelTab] = useState<AuctionPanelTab>('current');
  const [roomLots, setRoomLots] = useState<Lot[]>([]);
  const [roomLotsLoading, setRoomLotsLoading] = useState(false);
  const [roomLotsError, setRoomLotsError] = useState('');

  const meId = user?.id ?? '';
  const currentLot = room.currentLot;
  const roomProfile = getDemoRoomProfile(roomId);
  const roomName = roomProfile?.roomName || room.snapshot?.roomName || DEFAULT_DEMO_ROOM_PROFILE.roomName;
  const isBidPending = Boolean(room.localOptimistic.pendingBid) || bidding;
  const accountRoleMessage = status !== 'authenticated' && reason === '该账号不是买家账号' ? reason : '';
  const showBuyerAuth = !user && (authMode === 'real' || reason?.includes('请先登录'));

  const ranking = useMemo(
    () => room.ranking.map((item) => ({ ...item, isMe: item.userId === meId })),
    [meId, room.ranking],
  );

  const visibleResultOrder = ownOrderForLot(resultOrder || room.activeOrder, meId, resultLot?.id);

  const refreshRoomLots = useCallback(async () => {
    setRoomLotsLoading(true);
    setRoomLotsError('');
    try {
      const lots = await listRoomLots(roomId);
      setRoomLots(lots);
      return lots;
    } catch (e) {
      const message = e instanceof Error ? e.message : '拍品队列加载失败';
      setRoomLotsError(message);
      throw e;
    } finally {
      setRoomLotsLoading(false);
    }
  }, [roomId]);

  const pushNotice = useCallback((notice: string) => {
    if (!notice) return;
    setNotices((current) => current.includes(notice) ? current : [notice, ...current].slice(0, 6));
    window.setTimeout(() => setNotices((current) => current.filter((item) => item !== notice)), 3600);
  }, []);

  const syncPrivateResult = useCallback(async (
    lotId: string,
    options: { showModal?: boolean; refreshOrderList?: boolean; silent?: boolean } = {},
  ) => {
    if (!lotId) return null;
    try {
      const result = await refreshLotResult(lotId);
      if (options.showModal ?? true) {
        setResultLot(result.lot);
        setResultOrder(result.order || null);
      }
      if (result.order && !options.silent) pushNotice('你的订单已同步');
      if (options.refreshOrderList) await refreshOrders(DEFAULT_ACTIVITY_QUERY).catch(() => undefined);
      return result;
    } catch (e) {
      if (!options.silent) pushNotice(e instanceof Error ? e.message : '成交结果同步失败');
      return null;
    }
  }, [pushNotice, refreshLotResult, refreshOrders]);

  const recoverRealtimeState = useCallback(async () => {
    await reload().catch(() => undefined);
    await refreshRoomLots().catch(() => undefined);
    const lotId = resultLot?.id || currentLot?.id || '';
    if (lotId) {
      await syncPrivateResult(lotId, {
        showModal: Boolean(resultLot),
        refreshOrderList: true,
        silent: true,
      });
    } else {
      await refreshOrders(DEFAULT_ACTIVITY_QUERY).catch(() => undefined);
    }
  }, [currentLot?.id, refreshOrders, refreshRoomLots, reload, resultLot, syncPrivateResult]);

  const handleSocketEvent = useCallback((event: AuctionSocketEvent) => {
    if (event.roomId && event.roomId !== roomId) return;
    if (event.type === AUCTION_EVENT_TYPE.SERVER_HEARTBEAT) return;

    const previousLeaderId = room.currentLot?.leadingUserId;
    applyEvent(event);

    const notice = noticeForAuctionEvent(event, meId, previousLeaderId);
    if (notice) pushNotice(notice);
    if (eventMayChangeLotList(event)) void refreshRoomLots().catch(() => undefined);

    if (isSettlementEventType(event.type)) {
      const lotId = lotIdFromPublicEvent(event, resultLot?.id || currentLot?.id || '');
      void syncPrivateResult(lotId, {
        showModal: true,
        refreshOrderList: isPrivateRefreshEventType(event.type),
      });
    }
  }, [applyEvent, currentLot?.id, meId, pushNotice, refreshRoomLots, resultLot?.id, room.currentLot?.leadingUserId, roomId, syncPrivateResult]);

  const wsState = useAuctionSocket(roomId, handleSocketEvent, recoverRealtimeState);

  useEffect(() => {
    const timer = window.setTimeout(() => {
      void refreshRoomLots().catch(() => undefined);
    }, 0);
    return () => window.clearTimeout(timer);
  }, [refreshRoomLots]);

  const submitBuyerAuth = useCallback(async () => {
    if (authBusy) return;
    const validationError = validateBuyerCredentials({
      username: authUsername,
      password: authPassword,
      nickname: authNickname,
      requireNickname: authFormMode === 'register',
    });
    if (validationError) {
      setAuthError(validationError);
      return;
    }

    setAuthBusy(true);
    setAuthError('');
    try {
      const username = normalizeBuyerUsername(authUsername);
      if (authFormMode === 'reset') {
        await resetBuyerPassword(username, authPassword);
        setAuthFormMode('login');
        setAuthPassword('');
        pushNotice('密码已重置，请用新密码登录');
        return;
      }
      if (authFormMode === 'login') await loginBuyer(username, authPassword);
      else await registerBuyer(username, authPassword, authNickname.trim() || nicknameFromUsername(username));
      setBidError('');
      pushNotice(authFormMode === 'login' ? '登录成功，可以出价' : '注册成功，可以出价');
    } catch (e) {
      setAuthError(e instanceof Error ? e.message : '账号处理失败，请重试');
    } finally {
      setAuthBusy(false);
    }
  }, [authBusy, authFormMode, authNickname, authPassword, authUsername, loginBuyer, pushNotice, registerBuyer, resetBuyerPassword]);

  const openAuctionPanel = useCallback((tab: AuctionPanelTab = 'current') => {
    setAuctionPanelTab(tab);
    setAuctionPanelOpen(true);
    setBidError('');
    void reload().catch(() => undefined);
    void refreshRoomLots().catch(() => undefined);
    if (tab === 'mine') void refreshOrders(DEFAULT_ACTIVITY_QUERY).catch(() => undefined);
  }, [refreshOrders, refreshRoomLots, reload]);

  const openBuyerAuthPanel = useCallback(() => {
    setAuthFormMode('login');
    setAuctionPanelTab('current');
    setAuctionPanelOpen(true);
  }, []);

  const submitBid = useCallback(async (amount: number) => {
    if (isBidPending) return;
    if (!currentLot) {
      setBidError('当前暂无竞拍，等待主播开拍');
      return;
    }
    if (!isBiddableLotStatus(currentLot.status)) {
      setBidError('当前拍品还未开拍或已结束');
      return;
    }
    if (meId && currentLot.leadingUserId === meId) {
      const message = '你已领先，等待其他买家出价后再加价';
      setBidError(message);
      pushNotice(message);
      return;
    }

    let session: Awaited<ReturnType<typeof ensureReadyForBid>>;

    try {
      session = await ensureReadyForBid();
    } catch (e) {
      const message = bidFailureMessage(e);
      setBidError(message);
      if (shouldOpenBuyerAuth(e)) openBuyerAuthPanel();
      pushNotice(message);
      return;
    }

    if (!hasConfirmedDeposit(roomId, currentLot.id, session.user.id)) {
      setDepositPrompt({ lot: currentLot, bidAmount: amount, userId: session.user.id });
      return;
    }

    setBidding(true);
    setBidError('');
    let idempotencyKey = '';

    try {
      idempotencyKey = createIdempotencyKey('bid', currentLot.id, session.user.id, amount);
      markBidStarted(currentLot.id, normalizeMoney(amount), idempotencyKey);

      const res = await placeBid(currentLot.id, {
        amount: normalizeMoney(amount),
        clientKnownVersion: currentLot.version,
        idempotencyKey,
      });

      if (res.accepted) {
        applyEvent({
          type: AUCTION_EVENT_TYPE.BID_ACCEPTED,
          roomId,
          lotId: currentLot.id,
          lot: res.lot,
          bid: res.bid,
          ranking: res.ranking,
          serverTimeUnixMs: Date.now(),
        });
        pushNotice(res.lot?.leadingUserId === session.user.id || res.bid?.userId === session.user.id ? '出价成功，你已领先' : '出价成功，等待排名更新');
      } else {
        const message = bidFailureMessage(res.rejectReason || '后端未接受本次出价');
        setBidError(message);
        pushNotice(message);
      }
    } catch (e) {
      const message = bidFailureMessage(e);
      setBidError(message);
      if (shouldOpenBuyerAuth(e)) openBuyerAuthPanel();
      pushNotice(message);
    } finally {
      markBidSettled(idempotencyKey);
      setBidding(false);
    }
  }, [applyEvent, currentLot, ensureReadyForBid, isBidPending, markBidSettled, markBidStarted, meId, openBuyerAuthPanel, pushNotice, roomId]);

  const confirmDepositPayment = useCallback(() => {
    if (!depositPrompt) return;
    markDepositConfirmed(roomId, depositPrompt.lot.id, depositPrompt.userId);
    const nextBidAmount = depositPrompt.bidAmount;
    setDepositPrompt(null);
    void submitBid(nextBidAmount);
  }, [depositPrompt, roomId, submitBid]);

  const handlePaymentPaid = useCallback(async (order?: OrderSummary, payment?: PaymentSummary) => {
    markPaymentSettled(order, payment);
    if (order) {
      setResultOrder(order);
      pushNotice(order.paymentStatus === 'SUCCESS' || payment?.status === 'SUCCESS' ? '支付成功，订单已刷新' : '支付状态已同步');
    }

    await refreshOrders(DEFAULT_ACTIVITY_QUERY).catch(() => undefined);
    const lotId = order?.lotId || payOrder?.lotId || resultLot?.id || currentLot?.id || '';
    if (lotId) {
      const latest = await syncPrivateResult(lotId, { showModal: Boolean(resultLot), silent: true });
      if (latest?.order) setResultOrder(latest.order);
    }
  }, [currentLot?.id, markPaymentSettled, payOrder?.lotId, pushNotice, refreshOrders, resultLot, syncPrivateResult]);

  const closeResult = useCallback(() => {
    setResultLot(null);
    setResultOrder(null);
  }, []);

  const nextLot = useCallback(() => {
    setResultLot(null);
    setResultOrder(null);
    void reload().catch(() => undefined);
    void refreshRoomLots().catch(() => undefined);
  }, [refreshRoomLots, reload]);

  return {
    roomId,
    room,
    loading,
    error,
    roomName,
    currentLot,
    ranking,
    meId,
    wsState,
    notices,
    bidError,
    isBidPending,
    accountRoleMessage,
    showBuyerAuth,
    buyerAuth: {
      mode: authFormMode,
      username: authUsername,
      password: authPassword,
      nickname: authNickname,
      busy: authBusy,
      error: authError,
      setMode: setAuthFormMode,
      setUsername: setAuthUsername,
      setPassword: setAuthPassword,
      setNickname: setAuthNickname,
      submit: submitBuyerAuth,
    },
    auctionPanel: {
      open: auctionPanelOpen,
      tab: auctionPanelTab,
      lots: roomLots,
      loading: roomLotsLoading,
      error: roomLotsError,
    },
    resultLot,
    visibleResultOrder,
    payOrder,
    depositPrompt,
    actions: {
      submitBid,
      confirmDepositPayment,
      closeDepositPrompt: () => setDepositPrompt(null),
      closeResult,
      nextLot,
      openAuctionPanel,
      closeAuctionPanel: () => setAuctionPanelOpen(false),
      setAuctionPanelTab,
      refreshRoomLots,
      refreshOrders: () => refreshOrders(DEFAULT_ACTIVITY_QUERY),
      setPayOrder,
      markPaymentStarted,
      handlePaymentPaid,
    },
  };
}

export type LiveRoomController = ReturnType<typeof useLiveRoomController>;
