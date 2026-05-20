import { useEffect, useMemo, useState } from 'react';
import { Activity, BadgeCheck, BarChart3, CircleDollarSign, ClipboardList, Gavel, Radio, RefreshCw, ShieldAlert, ShoppingBag, UserCog, Users } from 'lucide-react';
import { cancelLot, createLot, listLots, revealTrustCard, settleLot, startDuel, startLot } from '../../features/auction/api/auctionApi';
import { AuthPanel } from '../../features/auth/ui/AuthPanel';
import { resultMessage } from '../../shared/api/result';
import { cny, formatMoney } from '../../shared/lib/money';
import { AdminLayout, DataTable, EmptyState, StatCard, StatusBadge } from '../../shared/ui/admin/AdminPrimitives';
import type { Lot, LotStatus, TrustRevealCard } from '../../shared/types/auction';

const STATUS_LABEL: Record<LotStatus, string> = {
  LOT_STATUS_UNSPECIFIED: '未知',
  LOT_STATUS_DRAFT: '草稿',
  LOT_STATUS_LIVE: '直播中',
  LOT_STATUS_SETTLED: '已成交',
  LOT_STATUS_CANCELLED: '已取消',
};

const STATUS_TONE: Record<LotStatus, 'neutral' | 'success' | 'warning' | 'danger' | 'info' | 'purple'> = {
  LOT_STATUS_UNSPECIFIED: 'neutral',
  LOT_STATUS_DRAFT: 'info',
  LOT_STATUS_LIVE: 'success',
  LOT_STATUS_SETTLED: 'purple',
  LOT_STATUS_CANCELLED: 'danger',
};

function toMs(v: number | string | undefined) {
  const n = Number(v ?? 0);
  return Number.isFinite(n) ? n : 0;
}

function formatClock(ms: number | string | undefined) {
  const n = toMs(ms);
  if (!n) return '未开始';
  return new Date(n).toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit', second: '2-digit' });
}

function statusText(status?: LotStatus) {
  return status ? STATUS_LABEL[status] : '未选择';
}

function trustSummary(cards?: TrustRevealCard[]) {
  const total = cards?.length ?? 0;
  const revealed = cards?.filter((card) => card.revealed).length ?? 0;
  return `${revealed}/${total}`;
}

export function HostConsolePage() {
  const [lots, setLots] = useState<Lot[]>([]);
  const [selected, setSelected] = useState<Lot | null>(null);
  const [title, setTitle] = useState('Vintage Cartier 手镯');
  const [cancelReason, setCancelReason] = useState('主播设备/商品状态异常，竞拍取消');
  const [busy, setBusy] = useState(false);
  const [notice, setNotice] = useState('');
  const [error, setError] = useState('');

  const showError = (e: unknown) => {
    const msg = resultMessage(e);
    setError(msg || '操作失败，请确认后端服务是否启动。');
    setNotice('');
  };

  const load = async () => {
    const xs = await listLots('demo');
    setLots(xs);
    setSelected((prev) => xs.find((x) => x.id === prev?.id) ?? xs.find((x) => x.status === 'LOT_STATUS_LIVE') ?? xs[xs.length - 1] ?? null);
  };

  useEffect(() => {
    load().catch(showError);
  }, []);

  const overview = useMemo(() => {
    const live = lots.filter((lot) => lot.status === 'LOT_STATUS_LIVE').length;
    const drafts = lots.filter((lot) => lot.status === 'LOT_STATUS_DRAFT').length;
    const settled = lots.filter((lot) => lot.status === 'LOT_STATUS_SETTLED').length;
    const cancelled = lots.filter((lot) => lot.status === 'LOT_STATUS_CANCELLED').length;
    const gmv = lots.reduce((sum, lot) => lot.status === 'LOT_STATUS_SETTLED' ? sum + Number(lot.finalPrice?.amount || lot.currentPrice?.amount || 0) : sum, 0);
    const trustTotal = selected?.trustCards?.length ?? 0;
    const trustRevealed = selected?.trustCards?.filter((card) => card.revealed).length ?? 0;
    return { live, drafts, settled, cancelled, gmv, trustTotal, trustRevealed };
  }, [lots, selected]);

  const create = async () => {
    if (busy) return;
    setBusy(true);
    setError('');
    setNotice('正在创建草稿...');
    try {
      const lot = await createLot({
        roomId: 'demo',
        title,
        description: '二手奢侈品竞拍样品，含证书、瑕疵说明与售后承诺。',
        imageUrl: 'https://images.unsplash.com/photo-1611591437281-460bfbe1220a?w=900',
        rule: {
          startPrice: cny(268800),
          minIncrement: cny(10000),
          durationSeconds: 300,
          antiSnipeWindowSeconds: 15,
          antiSnipeExtendSeconds: 15,
          maxExtendCount: 3,
        },
        trustCards: [
          { id: 'cert', type: 'TRUST_CARD_TYPE_CERTIFICATE', title: '鉴定证书', content: '证书编号已核验，支持回放查看。' },
          { id: 'flaw', type: 'TRUST_CARD_TYPE_FLAW', title: '瑕疵说明', content: '边角轻微磨损，已在细节图中标注。' },
          { id: 'service', type: 'TRUST_CARD_TYPE_SERVICE', title: '售后承诺', content: '支持平台复检与保真服务。' },
        ],
      });
      setSelected(lot);
      await load();
      setNotice(`草稿创建成功：${lot.title}。下一步点击“开拍”。`);
    } catch (e) {
      showError(e);
    } finally {
      setBusy(false);
    }
  };

  const act = async (label: string, fn: (id: string) => Promise<Lot>) => {
    if (!selected || busy) return;
    setBusy(true);
    setError('');
    setNotice(`正在${label}...`);
    try {
      const lot = await fn(selected.id);
      setSelected(lot);
      await load();
      setNotice(`${label}成功。`);
    } catch (e) {
      showError(e);
    } finally {
      setBusy(false);
    }
  };

  const reveal = async (cardId: string) => {
    if (!selected || busy) return;
    setBusy(true);
    setError('');
    setNotice('正在揭示信任卡片...');
    try {
      const r = await revealTrustCard(selected.id, cardId);
      setSelected(r.lot);
      await load();
      setNotice(`已揭示：${r.trustCard.title}`);
    } catch (e) {
      showError(e);
    } finally {
      setBusy(false);
    }
  };

  const cancelSelected = async () => {
    if (!selected || busy || selected.status !== 'LOT_STATUS_LIVE') return;
    const reason = cancelReason.trim();
    if (!reason) {
      setError('请输入异常取消原因。');
      setNotice('');
      return;
    }
    setBusy(true);
    setError('');
    setNotice('正在异常取消竞拍...');
    try {
      const lot = await cancelLot(selected.id, reason);
      setSelected(lot);
      await load();
      setNotice(`竞拍已取消：${reason}`);
    } catch (e) {
      showError(e);
    } finally {
      setBusy(false);
    }
  };

  const navItems = [
    { id: 'dashboard', label: '运营总览', description: '拍品与状态', active: true, icon: <BarChart3 size={18} /> },
    { id: 'lots', label: '拍品订单', description: '竞拍主链路', active: true, icon: <ShoppingBag size={18} /> },
    { id: 'control', label: '直播控场', description: 'Duel / 落锤', active: true, icon: <Radio size={18} /> },
    { id: 'users', label: '用户管理', description: '契约待扩展', disabled: true, icon: <Users size={18} /> },
    { id: 'payment', label: '支付结算', description: '契约待扩展', disabled: true, icon: <CircleDollarSign size={18} /> },
  ];

  return (
    <AdminLayout
      title="直播竞拍运营后台"
      subtitle="参考 API 管理后台的信息架构：侧边导航、顶部状态、指标卡、数据表、空态和高风险操作区；所有可点击主链路均调用真实后端。"
      navItems={navItems}
      actions={<button className="adminToolbarButton" disabled={busy} onClick={() => load().catch(showError)}><RefreshCw size={16} />刷新</button>}
      userSlot={<div className="adminUserChip"><UserCog size={16} /><span>admin console</span></div>}
    >
      {(notice || error) && <section className={error ? 'notice error' : 'notice'}>{error || notice}</section>}

      <section className="adminStatsGrid" aria-label="状态总览">
        <StatCard label="拍品总数" value={lots.length} hint="room: demo" tone="primary" icon={<ClipboardList size={22} />} />
        <StatCard label="直播中" value={overview.live} hint="可出价 / 可控场" tone="success" icon={<Activity size={22} />} />
        <StatCard label="草稿待开拍" value={overview.drafts} hint="创建后进入主链路" tone="warning" icon={<ShoppingBag size={22} />} />
        <StatCard label="成交 / 取消" value={`${overview.settled} / ${overview.cancelled}`} hint="终态拍品" tone={overview.cancelled ? 'danger' : 'neutral'} icon={<Gavel size={22} />} />
        <StatCard label="成交 GMV" value={formatMoney({ amount: overview.gmv, currency: 'CNY' })} hint="已成交 finalPrice 汇总" tone="primary" icon={<CircleDollarSign size={22} />} />
      </section>

      <section className="adminDashboardGrid">
        <article className="adminPanel createPanel">
          <div className="adminPanelHeader">
            <div>
              <p className="eyebrow">CREATE LOT</p>
              <h2>快速创建拍品</h2>
            </div>
            <StatusBadge label="写接口" tone="info" />
          </div>
          <label className="fieldLabel" htmlFor="lot-title">拍品标题</label>
          <input id="lot-title" value={title} onChange={(e) => setTitle(e.target.value)} />
          <button className="primaryAction fullWidth" disabled={busy || !title.trim()} onClick={create}>{busy ? '处理中...' : '创建草稿'}</button>
          <p className="adminHelpText">POST /api/lots。规则默认：起拍 ¥2,688、加价 ¥100、5 分钟、15 秒防狙击延时。</p>
        </article>

        <article className="adminPanel currentLotPanel">
          <div className="adminPanelHeader">
            <div>
              <p className="eyebrow">CURRENT LOT</p>
              <h2>当前控场拍品</h2>
            </div>
            {selected && <StatusBadge label={STATUS_LABEL[selected.status]} tone={STATUS_TONE[selected.status]} />}
          </div>

          {selected ? (
            <div className="adminCurrentLot">
              <div className="lotPoster"><img src={selected.imageUrl} alt={selected.title} /><span>v{selected.version}</span></div>
              <div className="lotControlBody">
                <h3>{selected.title}</h3>
                <p className="lotDesc">{selected.description}</p>
                <div className="priceBoard">
                  <span>当前价</span>
                  <strong>{formatMoney(selected.currentPrice)}</strong>
                  <small>最低加价 {formatMoney(selected.rule.minIncrement)} · 结束 {formatClock(selected.endsAtUnixMs)}</small>
                </div>
                <div className="operatorActions" aria-label="拍品操作">
                  <button className="primaryAction" disabled={busy || selected.status !== 'LOT_STATUS_DRAFT'} onClick={() => act('开拍', startLot)}>开拍</button>
                  <button className="secondaryAction" disabled={busy || selected.status !== 'LOT_STATUS_LIVE'} onClick={() => act('进入 Duel', startDuel)}>进入 Duel</button>
                  <button className="primaryAction dark" disabled={busy || selected.status !== 'LOT_STATUS_LIVE'} onClick={() => act('落锤成交', settleLot)}>落锤成交</button>
                </div>
                <dl className="lotFacts">
                  <div><dt>领先用户</dt><dd>{selected.leadingNickname || '暂无'}</dd></div>
                  <div><dt>阶段</dt><dd>{selected.playbookStage}</dd></div>
                  <div><dt>信任卡</dt><dd>{trustSummary(selected.trustCards)}</dd></div>
                </dl>
              </div>
            </div>
          ) : <EmptyState title="暂无拍品" description="请先登录主播/运营/admin 账号并创建草稿。" action={<button className="secondaryAction" onClick={create} disabled={busy || !title.trim()}>创建示例拍品</button>} />}
        </article>
      </section>

      <section className="adminSplitGrid">
        <article className="adminPanel tablePanel">
          <div className="adminPanelHeader">
            <div>
              <p className="eyebrow">LOTS TABLE</p>
              <h2>拍品订单列表</h2>
            </div>
            <span className="adminTableMeta">GET /api/lots</span>
          </div>
          <DataTable
            rows={lots}
            rowKey={(lot) => lot.id}
            empty={<EmptyState title="暂无拍品数据" description="列表为空时不构造 mock 数据；请通过真实创建接口新增拍品。" />}
            columns={[
              { key: 'title', label: '拍品', render: (lot) => <button className="adminTextButton" onClick={() => setSelected(lot)}>{lot.title}<small>{lot.id}</small></button> },
              { key: 'status', label: '状态', render: (lot) => <StatusBadge label={STATUS_LABEL[lot.status]} tone={STATUS_TONE[lot.status]} /> },
              { key: 'price', label: '当前价', render: (lot) => <strong>{formatMoney(lot.currentPrice)}</strong> },
              { key: 'leader', label: '领先用户', render: (lot) => lot.leadingNickname || '—' },
              { key: 'trust', label: '信任揭示', render: (lot) => trustSummary(lot.trustCards) },
              { key: 'end', label: '结束时间', render: (lot) => formatClock(lot.endsAtUnixMs) },
              { key: 'actions', label: '操作', render: (lot) => <button className="secondaryAction mini" onClick={() => setSelected(lot)}>控场</button> },
            ]}
          />
        </article>

        <aside className="adminSideStack">
          <AuthPanel mode="host" />

          <article className="adminPanel trustPanel">
            <div className="adminPanelHeader">
              <div>
                <p className="eyebrow">TRUST REVEAL</p>
                <h2>信任揭示</h2>
              </div>
              <BadgeCheck size={20} />
            </div>
            <div className="trustList">
              {selected?.trustCards?.map((card) => (
                <div className={`trustCard ${card.revealed ? 'revealed' : ''}`} key={card.id}>
                  <div>
                    <strong>{card.title}</strong>
                    <span>{card.revealed ? card.content : '待主播口播后揭示'}</span>
                  </div>
                  <button className="secondaryAction mini" disabled={busy || card.revealed || selected.status === 'LOT_STATUS_DRAFT'} onClick={() => reveal(card.id)}>{card.revealed ? '已揭示' : '揭示'}</button>
                </div>
              ))}
              {!selected && <p className="adminHelpText">创建拍品后会显示可揭示卡片。</p>}
            </div>
          </article>

          <article className="adminPanel dangerPanel">
            <div className="adminPanelHeader">
              <div>
                <p className="eyebrow dangerText">RISK OPERATION</p>
                <h2>异常取消</h2>
              </div>
              <ShieldAlert size={20} />
            </div>
            <p className="adminHelpText">仅 LIVE 拍品可触发，原因会提交到后端 CancelLot 契约并进入拍品状态。</p>
            <label className="fieldLabel" htmlFor="cancel-reason">取消原因</label>
            <textarea id="cancel-reason" value={cancelReason} onChange={(e) => setCancelReason(e.target.value)} placeholder="例如：主播网络中断 / 商品状态异常" />
            <button className="dangerAction fullWidth" disabled={busy || selected?.status !== 'LOT_STATUS_LIVE' || !cancelReason.trim()} onClick={cancelSelected}>异常取消竞拍</button>
            <p className="contractNote">POST /api/lots/{'{lot_id}'}/cancel</p>
            {selected?.status === 'LOT_STATUS_CANCELLED' && <p className="cancelReason">已取消原因：{selected.cancelReason || '后端未返回原因'}</p>}
          </article>
        </aside>
      </section>

      <section className="adminRoadmapGrid" aria-label="后台扩展结构">
        <article className="adminPanel mutedPanel"><h3>用户管理</h3><p>沿用后台范式预留：用户表、角色、余额/出价历史、风控标签。当前后端未提供用户列表契约，因此不发请求、不造假数据。</p></article>
        <article className="adminPanel mutedPanel"><h3>订单与支付</h3><p>预留订单统计、退款、支付方式分布、Top 买家等结构。待后端提供 payment/order API 后接入真实数据。</p></article>
        <article className="adminPanel mutedPanel"><h3>运营监控</h3><p>预留延迟、WS 连接、竞拍拒绝原因和异常取消审计。当前主链路以 WebSocket 与快照恢复为准。</p></article>
      </section>
    </AdminLayout>
  );
}
