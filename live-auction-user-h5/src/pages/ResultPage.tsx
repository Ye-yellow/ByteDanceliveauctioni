import { useEffect, useMemo, useState } from 'react';
import { isOrderFailed, isOrderPaid, orderStatusLabel } from '../entities/order/model/privacy';
import { getLotResult } from '../features/auction/api/auctionApi';
import type { LotResult, MoneyInput, OrderSummary } from '../shared/api/types';
import { formatMoney } from '../shared/lib/money';

const FALLBACK_LOT_TITLE = '直播竞拍拍品';
const FALLBACK_AMOUNT = 15000;

type ResultStatus = {
  label: string;
  title: string;
  tone: string;
  note: string;
};

type ResultView = {
  lotId: string;
  roomId: string;
  title: string;
  winner: string;
  imageUrl: string;
  settledAt: string;
  status: ResultStatus;
  finalAmount: MoneyInput;
  source: 'server' | 'fallback';
};

function resultLotId(): string {
  const match = location.pathname.match(/^\/m\/result\/([^/]+)/);
  return match ? decodeURIComponent(match[1]) : '';
}

function queryText(params: URLSearchParams, keys: string[], fallback: string): string {
  const value = keys.map((key) => params.get(key)).find((item) => item && item.trim());
  return value ? value.trim() : fallback;
}

function queryAmount(params: URLSearchParams): number {
  const raw = queryText(params, ['amount', 'finalPrice', 'price'], '');
  const normalized = raw.replace(/[^\d.]/g, '');
  const value = Number(normalized);
  if (!Number.isFinite(value) || value <= 0) return FALLBACK_AMOUNT;
  if (normalized.includes('.')) return Math.round(value * 100);
  return value >= 1000 ? Math.round(value) : Math.round(value * 100);
}

function queryStatusCopy(status: string): ResultStatus {
  const normalized = status.toLowerCase();
  if (['paid', 'completed', 'success'].includes(normalized)) {
    return {
      label: '已支付',
      title: '成交完成',
      tone: 'success',
      note: '订单已完成，平台会继续保留成交记录。',
    };
  }
  if (['failed', 'expired', 'cancelled', 'closed'].includes(normalized)) {
    return {
      label: '已关闭',
      title: '订单已关闭',
      tone: 'danger',
      note: '本次落锤结果已结束，可回直播间继续查看其他拍品。',
    };
  }
  return {
    label: '待支付',
    title: '竞拍成功',
    tone: 'warning',
    note: '请在支付时限内确认地址并完成付款，保证金将在规则内处理。',
  };
}

function serverStatusCopy(result: LotResult): ResultStatus {
  const order = result.order;
  if (order) {
    if (isOrderPaid(order)) {
      return {
        label: orderStatusLabel(order),
        title: '成交完成',
        tone: 'success',
        note: '订单已完成，成交记录来自服务器结果。',
      };
    }
    if (isOrderFailed(order)) {
      return {
        label: orderStatusLabel(order),
        title: '订单已关闭',
        tone: 'danger',
        note: '订单状态来自服务器，请以我的订单页为准。',
      };
    }
    return {
      label: orderStatusLabel(order),
      title: '竞拍成功',
      tone: 'warning',
      note: '请在支付时限内确认地址并完成付款，保证金将在规则内处理。',
    };
  }

  const lotStatus = String(result.lot.status || '').toUpperCase();
  if (lotStatus.includes('CANCELLED') || lotStatus.includes('FAILED')) {
    return {
      label: '未成交',
      title: '竞拍结束',
      tone: 'danger',
      note: '本次拍品未产生有效成交，结果来自服务器。',
    };
  }
  if (lotStatus.includes('SETTLED')) {
    return {
      label: '已落锤',
      title: result.winnerNickname || result.lot.winnerNickname ? '竞拍成功' : '竞拍结束',
      tone: result.winnerNickname || result.lot.winnerNickname ? 'success' : 'warning',
      note: '成交结果来自服务器，订单生成后可在我的订单中查看。',
    };
  }
  return {
    label: '同步中',
    title: '等待结果',
    tone: 'warning',
    note: '服务器结果正在同步，请稍后刷新或回直播间查看。',
  };
}

function maskId(id: string): string {
  if (!id) return '暂无订单号';
  if (id.length <= 10) return id;
  return `${id.slice(0, 6)}...${id.slice(-4)}`;
}

function formatResultTime(value: number | string | undefined, fallback: string): string {
  const time = Number(value || 0);
  if (!Number.isFinite(time) || time <= 0) return fallback;
  return new Date(time).toLocaleString('zh-CN', {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
  });
}

function finalAmountForResult(result: LotResult, fallback: MoneyInput): MoneyInput {
  return result.order?.amount || result.finalPrice || result.lot.finalPrice || result.lot.currentPrice || result.lot.rule.startPrice || fallback;
}

function fallbackView(params: URLSearchParams, lotId: string): ResultView {
  return {
    lotId,
    roomId: params.get('roomId') || '',
    title: queryText(params, ['title', 'lotTitle', 'name'], FALLBACK_LOT_TITLE),
    winner: queryText(params, ['winner', 'nickname', 'buyer'], '中标用户'),
    imageUrl: queryText(params, ['imageUrl', 'image', 'cover'], ''),
    settledAt: queryText(params, ['settledAt', 'time'], '刚刚落锤'),
    status: queryStatusCopy(params.get('status') || 'pending'),
    finalAmount: queryAmount(params),
    source: 'fallback',
  };
}

function serverView(result: LotResult, fallback: ResultView): ResultView {
  const order: OrderSummary | undefined = result.order;
  return {
    lotId: result.lot.id || fallback.lotId,
    roomId: result.lot.roomId || order?.roomId || fallback.roomId,
    title: result.lot.title || order?.lotTitle || fallback.title,
    winner: result.winnerNickname || result.lot.winnerNickname || order?.buyerNickname || fallback.winner,
    imageUrl: result.lot.imageUrl || order?.lotImageUrl || fallback.imageUrl,
    settledAt: formatResultTime(result.lot.settledAtUnixMs || order?.paidAtUnixMs || order?.createdAtUnixMs, fallback.settledAt),
    status: serverStatusCopy(result),
    finalAmount: finalAmountForResult(result, fallback.finalAmount),
    source: 'server',
  };
}

function unverifiedView(view: ResultView): ResultView {
  return {
    ...view,
    winner: '待服务器确认',
    status: {
      label: '未确认',
      title: '结果待同步',
      tone: 'danger',
      note: '服务器结果同步失败。页面不会采信链接中的成交状态，请回直播间或订单页查看真实结果。',
    },
  };
}

function goBack(fallback: string) {
  if (window.history.length > 1) {
    window.history.back();
    return;
  }
  window.location.assign(fallback);
}

export function ResultPage() {
  const params = useMemo(() => new URLSearchParams(location.search), []);
  const lotId = useMemo(() => resultLotId(), []);
  const fallback = useMemo(() => fallbackView(params, lotId), [lotId, params]);
  const [view, setView] = useState<ResultView>(fallback);
  const [loading, setLoading] = useState(Boolean(lotId));
  const [error, setError] = useState(lotId ? '' : '缺少拍品 ID，正在展示临时结果信息');
  const roomHref = view.roomId ? `/m/room/${encodeURIComponent(view.roomId)}` : '/home';
  const historyHref = '/shop/orders?from=result';
  const displayView = error && view.source === 'fallback' ? unverifiedView(view) : view;
  const lotInitial = Array.from(displayView.title)[0] || '拍';

  useEffect(() => {
    if (!lotId) return undefined;

    let disposed = false;
    void getLotResult(lotId)
      .then((result) => {
        if (!disposed) setView(serverView(result, fallback));
      })
      .catch((e) => {
        if (!disposed) setError(e instanceof Error ? e.message : '服务器结果同步失败');
      })
      .finally(() => {
        if (!disposed) setLoading(false);
      });

    return () => {
      disposed = true;
    };
  }, [fallback, lotId]);

  return (
    <main className="mobileShell resultPageShell">
      <header className="resultPageTopbar">
        <button type="button" onClick={() => goBack(roomHref)} aria-label="返回上一页">‹</button>
        <h1>竞拍结果</h1>
        <a href={historyHref}>订单</a>
      </header>

      <section className="resultPageHero" aria-label="成交结果">
        <div className="resultPageHeroMedia">
          {displayView.imageUrl ? <img src={displayView.imageUrl} alt={displayView.title} /> : <span>{lotInitial}</span>}
        </div>
        <div className="resultPageHeroShade" aria-hidden="true" />
        <div className="resultPageStatusRail" aria-label="结果状态">
          <span className={`resultPageState ${displayView.status.tone}`}>{loading ? '同步中' : displayView.status.label}</span>
          <b>{displayView.status.title}</b>
          <small>{displayView.settledAt}</small>
        </div>
      </section>

      <section className="resultPageCard" aria-label="拍品成交信息">
        <div className="resultPageProduct">
          <div>
            {displayView.imageUrl ? <img src={displayView.imageUrl} alt="" /> : <span>{lotInitial}</span>}
          </div>
          <section>
            <p>LiveAuction · {displayView.source === 'server' ? '服务器成交结果' : '待服务器确认'}</p>
            <h2>{displayView.title}</h2>
            <small>拍品 ID {maskId(displayView.lotId)}</small>
          </section>
        </div>

        <section className="resultPageAmount" aria-label="成交价">
          <span>{error && displayView.source === 'fallback' ? '链接参考金额' : '成交价'}</span>
          <strong className="scrollAmount" title={formatMoney(displayView.finalAmount)}>{formatMoney(displayView.finalAmount)}</strong>
        </section>

        <dl className="resultPageRows">
          <div>
            <dt>竞得者</dt>
            <dd>{displayView.winner}</dd>
          </div>
          <div>
            <dt>订单状态</dt>
            <dd>{displayView.status.label}</dd>
          </div>
          <div>
            <dt>落锤时间</dt>
            <dd>{displayView.settledAt}</dd>
          </div>
        </dl>

        <p className="resultPageNotice">
          {loading ? '正在从服务器同步竞拍结果...' : error ? `服务器结果同步失败：${error}。请以直播间或我的订单中的服务器结果为准。` : displayView.status.note}
        </p>
      </section>

      <section className="resultPageActions" aria-label="结果操作">
        <a className="primary" href={historyHref}>查看我的订单</a>
        <a href={roomHref}>回直播间</a>
        <a href="/home">继续刷直播</a>
      </section>
    </main>
  );
}
