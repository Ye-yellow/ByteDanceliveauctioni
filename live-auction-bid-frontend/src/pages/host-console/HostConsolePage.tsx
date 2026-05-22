import { useEffect, useMemo, useRef, useState, type ComponentType, type ReactNode } from 'react';
import {
  Activity,
  AlertTriangle,
  BarChart3,
  Bell,
  Boxes,
  CheckCircle2,
  ChevronRight,
  CircleDollarSign,
  ClipboardList,
  Clock3,
  Crown,
  DatabaseZap,
  Gauge,
  Gavel,
  Home,
  LayoutDashboard,
  ListChecks,
  LockKeyhole,
  Megaphone,
  MonitorDot,
  Package,
  PlayCircle,
  Radio,
  ReceiptText,
  RefreshCw,
  Search,
  Settings,
  ShieldAlert,
  ShieldCheck,
  ShoppingBag,
  Sparkles,
  TimerReset,
  TrendingUp,
  Trophy,
  UserCog,
  Users,
  Wifi,
  Zap,
} from 'lucide-react';
import { currentAuth } from '../../features/auth/api/authApi';
import { cancelLot, createDraftLot, deleteUploadedImage, getRoomSnapshot, listLots, patchDraftLot, queueLot, revealTrustCard, settleLot, startDuel, startLot, uploadImage } from '../../features/auction/api/auctionApi';
import { WS_BASE } from '../../shared/config/env';
import { normalizeAuctionEvent, resultMessage } from '../../shared/api/result';
import { formatAuctionLeftMs, getLotLeftMs, getServerOffsetMs } from '../../shared/lib/time';
import type { AuctionEvent, Bid, CreateLotRequest, Lot, Money, RoomSnapshot } from '../../shared/api/types';
import { HostConsoleShell, type StudioNavGroupConfig } from './components/HostConsoleShell';
import { StudioBadge, StudioButton, StudioCard, StudioEmptyState, StudioErrorState, StudioField, StudioMetricCard, StudioPageHeader as StudioSectionHeader, StudioTable, StudioTableSkeleton, StudioToastViewport, type StudioTone, useStudioToast } from './components/studio-ui';
import './styles/console-round06.css';

type Tone = StudioTone;
type AuctionStatus = '草稿' | '未开始' | '预热中' | '进行中' | '延时中' | '已成交' | '已流拍' | '已取消' | '异常';

type AuctionRow = {
  id: string;
  product: string;
  room: string;
  status: AuctionStatus;
  current: string;
  start: string;
  step: string;
  cap: string;
  viewers: number;
  remain: string;
  result: string;
};

type OrderRow = { id: string; product: string; image: string; buyer: string; price: string; time: string; pay: string; fulfill: string; auction: string; issue?: string };
type ProductRow = { name: string; category: string; estimate: string; stock: number; auction: string; created: string; status: string };
type BidRow = { auctionId: string; user: string; amount: string; leading: boolean; time: string; latency: string; result: string; key: string; device: string };
type RoomRow = { id: string; name: string; online: number; status: string; ws: number; latency: string; heartbeat: string };
type AlertRow = { type: string; level: '严重' | '警告' | '普通' | '已处理'; target: string; time: string; detail: string };

const auctions: AuctionRow[] = [
  { id: 'AUC-240521-001', product: 'Vintage Cartier 手镯', room: '小鹿珠宝直播间', status: '进行中', current: '¥8,260', start: '¥0', step: '¥100', cap: '¥12,000', viewers: 386, remain: '00:42', result: '领先中' },
  { id: 'AUC-240521-002', product: '限定 Art Print 版画', room: '小鹿珠宝直播间', status: '延时中', current: '¥1,980', start: '¥0', step: '¥50', cap: '¥3,200', viewers: 214, remain: '00:18', result: '最后出价自动延时 +10s' },
  { id: 'AUC-240521-003', product: 'Designer Bag 复古包', room: '小鹿珠宝直播间', status: '预热中', current: '¥0', start: '¥0', step: '¥200', cap: '¥18,800', viewers: 142, remain: '12:30', result: '未成交' },
  { id: 'AUC-240521-004', product: '手作香氛礼盒', room: '小鹿珠宝直播间', status: '已成交', current: '¥680', start: '¥0', step: '¥20', cap: '¥980', viewers: 96, remain: '结束', result: '成交已生成' },
  { id: 'AUC-240521-005', product: '复古机械键盘', room: '小鹿珠宝直播间', status: '异常', current: '¥1,260', start: '¥0', step: '¥50', cap: '¥2,000', viewers: 188, remain: '暂停', result: '锁冲突待处理' },
];

const products: ProductRow[] = [
  { name: 'Vintage Cartier 手镯', category: '珠宝配饰', estimate: '¥9,800', stock: 1, auction: 'AUC-240521-001', created: '05-21 09:12', status: '竞拍中' },
  { name: '限定 Art Print 版画', category: '艺术收藏', estimate: '¥2,400', stock: 3, auction: 'AUC-240521-002', created: '05-21 09:34', status: '竞拍中' },
  { name: 'Designer Bag 复古包', category: '箱包奢品', estimate: '¥16,800', stock: 1, auction: 'AUC-240521-003', created: '05-20 18:45', status: '已上架' },
  { name: '手作香氛礼盒', category: '生活方式', estimate: '¥699', stock: 12, auction: 'AUC-240521-004', created: '05-20 12:22', status: '已成交' },
];

const orders: OrderRow[] = [
  { id: 'ORD-20260521-8812', product: '手作香氛礼盒', image: '/vite.svg', buyer: 'u_8271', price: '¥680', time: '05-21 10:42', pay: '已支付', fulfill: '待发货', auction: 'AUC-240521-004' },
  { id: 'ORD-20260521-8813', product: '限量盲盒套装', image: '/vite.svg', buyer: 'u_1092', price: '¥1,260', time: '05-21 10:58', pay: '待支付', fulfill: '待履约', auction: 'AUC-240521-006', issue: '支付超时 7 分钟，需客服催付或关闭成交。' },
  { id: 'ORD-20260520-7720', product: '银饰项链', image: '/vite.svg', buyer: 'u_5518', price: '¥920', time: '05-20 21:16', pay: '已支付', fulfill: '已完成', auction: 'AUC-240520-018' },
  { id: 'ORD-20260520-7721', product: '复古机械键盘', image: '/vite.svg', buyer: 'u_0012', price: '¥1,260', time: '05-20 21:44', pay: '异常', fulfill: '待处理', auction: 'AUC-240521-005', issue: '锁冲突后阻断重复成交，等待技术复核。' },
];

const bidRows: BidRow[] = [
  { auctionId: 'AUC-240521-001', user: 'u_2718', amount: '¥8,260', leading: true, time: '12:56:42.184', latency: '42ms', result: '有效', key: 'idem_8fa2', device: '上海 / iOS' },
  { auctionId: 'AUC-240521-001', user: 'u_6621', amount: '¥8,160', leading: false, time: '12:56:41.932', latency: '58ms', result: '有效', key: 'idem_2c91', device: '杭州 / Android' },
  { auctionId: 'AUC-240521-002', user: 'u_3820', amount: '¥1,960', leading: false, time: '12:56:39.508', latency: '96ms', result: '低于当前价', key: 'idem_a981', device: '北京 / Web' },
  { auctionId: 'AUC-240521-005', user: 'u_0012', amount: '¥1,310', leading: false, time: '12:56:38.221', latency: '138ms', result: '锁冲突', key: 'idem_lock', device: '深圳 / Android' },
];



const alerts: AlertRow[] = [
  { type: 'Redis 锁冲突', level: '严重', target: 'AUC-240521-005', time: '12:54:18', detail: '同一拍品出现并发写冲突，已阻断重复成交。' },
  { type: '竞拍已结束仍出价', level: '警告', target: 'AUC-240521-004', time: '12:47:09', detail: '客户端延迟提交，服务端校验拒绝。' },
  { type: '封顶成交触发', level: '普通', target: 'AUC-240521-002', time: '12:41:22', detail: '用户接近封顶价，已推送风险提示。' },
];

const roleRows = [
  { role: '主播主账号', users: 1, scope: '当前主播空间', desc: '拥有拍品、竞拍、控场、成交、风控和团队权限全能力', risk: '高风险', enabled: ['拍品', '竞拍', '控场', '成交', '风控', '系统'] },
  { role: '数据复盘', users: 3, scope: '直播后复盘', desc: '只读数据、拍品表现和复盘报告，不参与控场和成交处理', risk: '普通', enabled: ['数据', '复盘'] },
  { role: '商品助理', users: 4, scope: '直播前筹备', desc: '维护拍品资料、图片、讲解卡；不能开拍和落锤', risk: '普通', enabled: ['拍品', '竞拍', '成交'] },
  { role: '场控', users: 8, scope: '绑定直播间', desc: '进入直播间中控台、推送提醒、查看实时出价和排行榜', risk: '中风险', enabled: ['控场', '出价', '排行'] },
  { role: '订单客服', users: 5, scope: '直播后成交', desc: '处理成交订单、支付和履约；不能修改玩法和控场', risk: '普通', enabled: ['成交'] },
];

const merchantRows = [
  { id: 'HOST-001', name: '小鹿珠宝直播团队', type: '主播主账号', owner: 'host_lulu', role: '主播主账号', room: '小鹿珠宝直播间', products: 128, auctions: 18, gmv: '¥326,800', status: '已认证', risk: '正常', last: '12:58' },
  { id: 'SUB-007', name: 'Ada 控场号', type: '团队子账号', owner: 'team_ada', role: '场控', room: '小鹿珠宝直播间', products: 0, auctions: 32, gmv: '¥512,460', status: '已启用', risk: '正常', last: '12:56' },
  { id: 'SUB-018', name: '奢品商品助理号', type: '团队子账号', owner: 'team_lux', role: '商品助理', room: '小鹿珠宝直播间', products: 76, auctions: 12, gmv: '¥882,100', status: '待复核', risk: '警告', last: '11:42' },
  { id: 'SUB-021', name: 'Kiki 助播号', type: '团队子账号', owner: 'team_kiki', role: '场控', room: '小鹿珠宝直播间', products: 0, auctions: 21, gmv: '¥92,680', status: '已启用', risk: '正常', last: '12:36' },
  { id: 'SUB-031', name: '数据复盘号', type: '团队子账号', owner: 'team_data', role: '数据复盘', room: '小鹿珠宝直播间', products: 52, auctions: 9, gmv: '¥146,900', status: '限制中', risk: '高风险', last: '10:18' },
];

const merchantAuditRows = [
  { time: '12:50:18', subject: '奢品运营号', action: '申请拍品发布权限', operator: 'team_lux', result: '待主账号审核' },
  { time: '12:36:04', subject: 'Ada 控场号', action: '绑定直播间', operator: 'host_lulu', result: '已通过' },
  { time: '11:58:22', subject: '数码拍品运营号', action: '触发异常限制', operator: 'risk-engine', result: '已限制' },
];

const permissionGroups = [
  { group: '直播筹备', items: ['创建竞拍', '编辑未开始规则', '开始竞拍', '强制结束', '异常取消', '封顶成交'] },
  { group: '直播间中控台', items: ['进入控场', '延时 10/30 秒', '推送提醒', '暂停展示', '查看实时出价', '查看排行榜'] },
  { group: '拍品与成交', items: ['添加拍品', '编辑拍品资料', '加入本场队列', '查看成交处理', '处理履约', '查看成交复盘'] },
  { group: '复盘与诊断', items: ['查看数据看板', '处理异常告警', '踢出异常连接', '重发房间状态', '查看消息日志', '导出风控记录'] },
  { group: '系统管理', items: ['团队协作', '配置岗位权限', '修改工作台设置', '查看审计日志'] },
];

const ranking = [
  { rank: 1, user: 'u_2718', price: '¥8,260', state: '🎉 领先' },
  { rank: 2, user: 'u_6621', price: '¥8,160', state: '⚡ 被超越' },
  { rank: 3, user: 'u_9102', price: '¥8,060', state: '追价中' },
  { rank: 4, user: 'u_3021', price: '¥7,960', state: '观望' },
];

const navGroups: StudioNavGroupConfig[] = [
  { label: '今日直播', items: [
    { label: '今日工作台', href: '/admin', icon: <LayoutDashboard size={17} />, match: (pathname) => pathname === '/admin' || pathname === '/host' },
    { label: '直播间中控台', href: '/admin/auctions/AUC-240521-001/control', icon: <Radio size={17} />, match: (pathname) => pathname.includes('/control') },
  ] },
  { label: '直播筹备', items: [
    { label: '添加拍品', href: '/admin/auctions/create', icon: <PlayCircle size={17} />, match: (pathname) => pathname.includes('/auctions/create') },
    { label: '本场拍品队列', href: '/admin/auctions', icon: <Gavel size={17} />, match: (pathname) => pathname.includes('/auctions') && !pathname.includes('/auctions/create') && !pathname.includes('/control') },
    { label: '拍品库', href: '/admin/products', icon: <Package size={17} />, match: (pathname) => pathname.includes('/products') },
  ] },
  { label: '直播后', items: [
    { label: '成交处理', href: '/admin/orders', icon: <ReceiptText size={17} />, match: (pathname) => pathname.includes('/orders') },
    { label: '出价明细', href: '/admin/bids', icon: <ListChecks size={17} />, match: (pathname) => pathname.includes('/bids') },
  ] },
  { label: '团队协作', items: [
    { label: '团队成员', href: '/admin/merchants', icon: <Users size={17} />, match: (pathname) => pathname.includes('/merchants') },
    { label: '实时同步状态', href: '/admin/realtime', icon: <Wifi size={17} />, match: (pathname) => pathname === '/admin/realtime' || pathname === '/host/realtime' },
  ] },
];

function pathTitle(pathname: string) {
  if (pathname === '/admin/realtime' || pathname === '/host/realtime') return '实时同步状态';
  if (pathname.includes('/auctions/create')) return '添加拍品';
  if (pathname.includes('/control')) return '直播间中控台';
  if (pathname.includes('/auctions')) return '本场拍品队列';
  if (pathname.includes('/orders')) return '成交处理';
  if (pathname.includes('/bids')) return '出价明细';
  if (pathname.includes('/merchants')) return '团队成员';
  if (pathname.includes('/settings')) return '工作台设置';
  if (pathname.includes('/products')) return '拍品库';
  return '今日工作台';
}

function toneForStatus(status: string): Tone {
  if (['竞拍中', '进行中', '已成交', '已连接', '有效', '已支付', '已完成'].includes(status)) return 'success';
  if (['待开拍', '延时中', '预热中', '待支付', '警告'].includes(status)) return 'warning';
  if (['异常取消', '异常', '已取消', '锁冲突', '严重'].includes(status)) return 'danger';
  if (['准备中', '草稿', '未开始', '普通'].includes(status)) return 'info';
  if (['已处理'].includes(status)) return 'neutral';
  return 'purple';
}

function StatusBadge({ label, tone = toneForStatus(label) }: { label: string; tone?: Tone }) {
  return <StudioBadge tone={tone}>{label}</StudioBadge>;
}

function AppShell({ children }: { children: ReactNode }) {
  return <HostConsoleShell navGroups={navGroups} currentHostRoom={currentHostRoom} currentTeamAccount={currentTeamAccount} titleForPath={pathTitle}>{children}</HostConsoleShell>;
}

function StatCard({ icon, label, value, hint, tone = 'info' }: { icon: ReactNode; label: string; value: ReactNode; hint: string; tone?: Tone }) {
  return <StudioMetricCard icon={icon} label={label} value={value} trend={hint} tone={tone} />;
}

function DataTable<T>({ columns, rows, rowKey }: { columns: { label: string; render: (row: T, index: number) => ReactNode }[]; rows: T[]; rowKey: (row: T, index: number) => string }) {
  return <StudioTable rows={rows} rowKey={rowKey} columns={columns} filters={<div className="laMiniSearch"><Search size={14} /> 筛选 / 搜索当前列表</div>} header={`共 ${rows.length} 条 · 第 1 / 1 页`} />;
}

function DashboardPage() {
  const liveAuction = auctions.find((item) => item.status === '进行中' || item.status === '延时中');
  const nextAuction = auctions.find((item) => item.status === '预热中' || item.status === '未开始') || auctions[2];
  const todoItems = [
    ['待开拍数量', auctions.filter((item) => ['预热中', '未开始', '草稿'].includes(item.status)).length, '本场队列需要确认开拍顺序'],
    ['讲解卡未完善', 2, '证书、瑕疵、细节卡待补充'],
    ['成交待处理', orders.filter((item) => item.fulfill.includes('待')).length, '支付/履约需要客服跟进'],
    ['异常待处理', alerts.filter((item) => item.level !== '已处理').length, '同步与风控异常需确认'],
  ] as const;
  const metricCards = [
    { icon: <Package />, label: '今日队列', value: auctions.length, hint: '当前固定直播间', tone: 'info' as Tone },
    { icon: <Radio />, label: '正在拍', value: liveAuction ? 1 : 0, hint: liveAuction ? liveAuction.product : '暂无进行中', tone: liveAuction ? 'success' as Tone : 'neutral' as Tone },
    { icon: <ReceiptText />, label: '已成交', value: auctions.filter((item) => item.status === '已成交').length, hint: '已生成成交记录', tone: 'purple' as Tone },
    { icon: <ShieldAlert />, label: '异常/待处理', value: todoItems[3][1], hint: '需要团队处理', tone: 'danger' as Tone },
  ];
  return <>
    <StudioCard padding="lg" className="dashboardWelcomeCard"><StudioSectionHeader eyebrow="今日直播工作台" title="今日直播工作台" description="围绕当前直播间查看本场排品、竞拍状态、成交处理和实时同步情况。" actions={<div className="dashboardWelcomeActions"><a className="studioButton studioButton-primary studioButton-md" href="/admin/auctions/create">添加拍品</a><a className="studioButton studioButton-secondary studioButton-md" href={liveAuction ? `/admin/auctions/${liveAuction.id}/control` : '/admin/auctions/AUC-240521-001/control'}>进入中控台</a><a className="studioButton studioButton-soft studioButton-md" href="/admin/auctions">查看本场队列</a></div>} /></StudioCard>
    <section className="todayRoomStrip"><div><span>当前直播间</span><b>{currentHostRoom.name}</b></div><div><span>主播</span><b>{currentHostRoom.owner}</b></div><div><span>当前账号角色</span><b>{currentTeamAccount.role}</b></div><div><span>在线人数</span><b>{currentHostRoom.online}</b></div><div><span>实时同步状态</span><StatusBadge label="正常 · 38ms" tone="success" /></div><div><span>最近心跳</span><b>刚刚</b></div></section>
    <section className="todayCoreGrid"><TodayLiveAuctionCard auction={liveAuction} /><TodayNextAuctionCard auction={nextAuction} hasLive={Boolean(liveAuction)} /><StudioCard title="今日待办" subtitle="Team Todo" className="todayTodoCard"><div className="todayTodoList">{todoItems.map(([label, value, hint]) => <div key={label}><span>{label}</span><b>{value}</b><small>{hint}</small></div>)}</div></StudioCard></section>
    <section className="todayMetricGrid">{metricCards.map((item) => <StatCard key={item.label} icon={item.icon} label={item.label} value={item.value} hint={item.hint} tone={item.tone} />)}</section>
    <section className="todayLowerGrid"><Panel title="本场拍品队列摘要" action={<a className="studioButton studioButton-ghost studioButton-sm" href="/admin/auctions">查看全部</a>}><TodayQueueSummary /></Panel><Panel title="最近出价 / 操作日志" action={<a className="studioButton studioButton-ghost studioButton-sm" href="/admin/bids">出价明细</a>}><TodayActivityFeed /></Panel></section>
  </>;
}

function TodayLiveAuctionCard({ auction }: { auction?: AuctionRow }) {
  if (!auction) return <StudioCard title="当前竞拍" subtitle="Now Live" className="todayAuctionCard todayAuctionEmpty"><div className="todayEmptyState"><Radio size={34} /><h3>当前暂无进行中竞拍</h3><p>本场直播间没有正在拍的拍品，可以从下一件拍品开始控场。</p><a className="studioButton studioButton-primary studioButton-md" href="/admin/auctions">查看本场队列</a></div></StudioCard>;
  return <StudioCard title="当前竞拍" subtitle="Now Live" className="todayAuctionCard"><div className="todayAuctionMedia"><span><Gavel size={30} /></span><StatusBadge label={auction.status} /></div><div className="todayAuctionBody"><h3>{auction.product}</h3><div className="todayCurrentPrice"><span>当前价</span><b>{auction.current}</b></div><div className="todayAuctionFacts"><span>倒计时 <b>{auction.remain}</b></span><span>领先用户 <b>{ranking[0]?.user || '暂无'}</b></span><span>在线观众 <b>{auction.viewers}</b></span></div><a className="studioButton studioButton-primary studioButton-md" href={`/admin/auctions/${auction.id}/control`}>进入中控台</a></div></StudioCard>;
}

function TodayNextAuctionCard({ auction, hasLive }: { auction: AuctionRow; hasLive: boolean }) {
  return <StudioCard title="下一件拍品" subtitle="Up Next" className="todayNextCard"><div className="todayNextProduct"><span><Package size={28} /></span><div><h3>{auction.product}</h3><small>{auction.id}</small></div></div><div className="todayNextRules"><span>起拍价<b>{auction.start}</b></span><span>加价幅度<b>{auction.step}</b></span><span>预计时长<b>{auction.remain}</b></span></div><div className="todayNextActions"><a className="studioButton studioButton-secondary studioButton-md" href="/admin/auctions/create">设为下一件</a><a className={`studioButton ${hasLive ? 'studioButton-secondary' : 'studioButton-primary'} studioButton-md`} href={`/admin/auctions/${auction.id}/control`}>{hasLive ? '等待当前结束' : '开拍'}</a></div></StudioCard>;
}

function TodayQueueSummary() {
  return <div className="todayQueueList">{auctions.slice(0, 5).map((item) => <div key={item.id}><div><b>{item.product}</b><span>{item.id}</span></div><StatusBadge label={item.status} /><strong>{item.current}</strong></div>)}</div>;
}

function TodayActivityFeed() {
  const logs = [
    ...bidRows.slice(0, 3).map((bid) => ({ key: bid.key, title: `${bid.user} 出价 ${bid.amount}`, meta: `${bid.time} · ${bid.result} · ${bid.latency}` })),
    ...alerts.slice(0, 2).map((alert) => ({ key: `${alert.type}-${alert.time}`, title: alert.type, meta: `${alert.time} · ${alert.target} · ${alert.detail}` })),
  ];
  return <div className="todayActivityList">{logs.map((log) => <div key={log.key}><span /><div><b>{log.title}</b><small>{log.meta}</small></div></div>)}</div>;
}

function Panel({ title, children, action }: { title: string; children: ReactNode; action?: ReactNode }) {
  return <StudioCard title={title} actions={action}>{children}</StudioCard>;
}

function AuctionTable({ rows = auctions, compact = false }: { rows?: AuctionRow[]; compact?: boolean }) {
  return <DataTable rows={rows} rowKey={(r) => r.id} columns={[
    { label: '拍品', render: (r) => <b>{r.product}</b> },
    { label: '直播间', render: (r) => r.room },
    { label: '状态', render: (r) => <StatusBadge label={r.status} /> },
    { label: '当前价', render: (r) => <strong>{r.current}</strong> },
    ...(compact ? [] : [
      { label: '从多少钱开始拍', render: (r: AuctionRow) => r.start },
      { label: '每次至少加多少钱', render: (r: AuctionRow) => r.step },
      { label: '到这个价自动成交', render: (r: AuctionRow) => r.cap },
    ]),
    { label: '参与人数', render: (r) => r.viewers },
    { label: '剩余时间', render: (r) => r.remain },
    { label: '成交结果', render: (r) => r.result },
    { label: '操作', render: (r) => <div className="laRowActions"><a href="/admin/auctions/create">编辑规则</a><a href={`/admin/auctions/${r.id}/control`}>进入控场</a><a href="/admin/bids">查看出价</a></div> },
  ]} />;
}

function CurrentHostRoomStatusCard() {
  return <div className="currentHostRoomStatus"><header><Wifi size={18} /><div><b>{currentHostRoom.name}</b><span>当前主播唯一直播间</span></div><StatusBadge label="实时同步正常" tone="success" /></header><div><span>roomId</span><b>{currentHostRoom.id}</b></div><div><span>主播主账号</span><b>{currentHostRoom.owner}</b></div><div><span>在线观众</span><b>{currentHostRoom.online}</b></div><div><span>平均延迟</span><b>{currentHostRoom.latency}</b></div><p>一个主播空间只绑定一个直播间，本页不提供直播间切换或直播间筛选。</p></div>;
}

function AlertList() {
  return <div className="laAlertList">{alerts.map((a) => <div key={`${a.type}-${a.time}`}><StatusBadge label={a.level} /><b>{a.type}</b><span>{a.detail}</span><small>{a.target} · {a.time}</small></div>)}</div>;
}

type AuctionUiStatus = '今日队列' | '草稿' | '准备中' | '待开拍' | '竞拍中' | '进行中' | '延时中' | '可落锤' | '已成交' | '已取消' | '异常' | '异常取消' | '历史拍品';
type TeamRole = '主播主账号' | '场控' | '商品助理' | '订单客服' | '数据复盘';
type AuctionFilters = { query: string; status: string; operator: string; briefing: string; created: string };
type DetailTab = '概览' | '规则快照' | '实时出价' | '操作日志' | '成交订单';

const auctionStatusTabs: AuctionUiStatus[] = ['今日队列', '准备中', '待开拍', '竞拍中', '已成交', '异常取消', '历史拍品'];
const currentTeamAccount = { username: 'host_lulu', displayName: '小鹿珠宝主账号', role: '主播主账号' as TeamRole };
const emptyFilters: AuctionFilters = { query: '', status: '全部状态', operator: '全部创建人', briefing: '全部讲解卡', created: '全部时间' };
const handledAuctionEventTypes = new Set([
  'AUCTION_EVENT_TYPE_ROOM_SNAPSHOT',
  'AUCTION_EVENT_TYPE_LOT_CREATED',
  'AUCTION_EVENT_TYPE_LOT_STARTED',
  'AUCTION_EVENT_TYPE_LOT_UPDATED',
  'AUCTION_EVENT_TYPE_BID_ACCEPTED',
  'AUCTION_EVENT_TYPE_BID_REJECTED',
  'AUCTION_EVENT_TYPE_RANKING_UPDATED',
  'AUCTION_EVENT_TYPE_LOT_SETTLED',
  'AUCTION_EVENT_TYPE_LOT_CANCELLED',
]);
const handledControlEventTypes = new Set([
  'AUCTION_EVENT_TYPE_ROOM_SNAPSHOT',
  'AUCTION_EVENT_TYPE_LOT_STARTED',
  'AUCTION_EVENT_TYPE_LOT_UPDATED',
  'AUCTION_EVENT_TYPE_BID_ACCEPTED',
  'AUCTION_EVENT_TYPE_BID_REJECTED',
  'AUCTION_EVENT_TYPE_RANKING_UPDATED',
  'AUCTION_EVENT_TYPE_TRUST_REVEALED',
  'AUCTION_EVENT_TYPE_DUEL_STARTED',
  'AUCTION_EVENT_TYPE_DUEL_ENDED',
  'AUCTION_EVENT_TYPE_LOT_SETTLED',
  'AUCTION_EVENT_TYPE_LOT_CANCELLED',
]);

function moneyText(money?: { amount: number | string; currency: string }) {
  if (!money) return '待接入';
  const amount = Number(money.amount || 0);
  return `¥${amount.toLocaleString('zh-CN')}`;
}

function capPriceText(lot: Lot) {
  return moneyText(lot.rule.capPrice);
}

function moneyDeltaText(a?: Money, b?: Money) {
  if (!a || !b) return '待接入';
  const delta = Math.max(0, Number(a.amount || 0) - Number(b.amount || 0));
  return `¥${delta.toLocaleString('zh-CN')}`;
}

function dateTimeText(value?: number | string) {
  const numberValue = Number(value || 0);
  if (!numberValue) return '未设置';
  return new Date(numberValue).toLocaleString('zh-CN', { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' });
}

function secondsLeftText(lot: Lot, serverTimeUnixMs?: number | string) {
  if (!Number(lot.endsAtUnixMs || 0)) return '未开始';
  return formatAuctionLeftMs(getLotLeftMs(lot, serverTimeUnixMs), 'queue');
}

function countdownToneClass(lot: Lot, serverTimeUnixMs?: number | string) {
  const leftMs = getLotLeftMs(lot, serverTimeUnixMs);
  return leftMs > 0 && leftMs < 10000 ? 'danger' : '';
}

function uiStatusOfLot(lot: Lot): AuctionUiStatus {
  if (lot.status === 'LOT_STATUS_QUEUED' || lot.queueStatus === 'LOT_QUEUE_STATUS_QUEUED') return '待开拍';
  if (lot.status === 'LOT_STATUS_READY') return '准备中';
  if (lot.status === 'LOT_STATUS_DRAFT' && lot.playbookStage === 'PLAYBOOK_STAGE_WARM_UP') return '草稿';
  if (lot.status === 'LOT_STATUS_DRAFT') return '准备中';
  if (lot.status === 'LOT_STATUS_LIVE') return '竞拍中';
  if (lot.status === 'LOT_STATUS_SETTLED') return '已成交';
  if (lot.status === 'LOT_STATUS_CANCELLED') return '异常取消';
  return '准备中';
}

function lotLiveStageText(lot: Lot) {
  if (lot.status === 'LOT_STATUS_LIVE' && lot.playbookStage === 'PLAYBOOK_STAGE_SETTLE_READY') return '可落锤';
  if (lot.status === 'LOT_STATUS_LIVE' && lot.duelState?.active) return '延时中';
  if (lot.status === 'LOT_STATUS_LIVE') return '进行中';
  return uiStatusOfLot(lot);
}

function canRole(role: TeamRole, action: 'create' | 'editRule' | 'start' | 'control' | 'cancel' | 'settle' | 'order') {
  if (role === '主播主账号') return true;
  if (role === '场控') return ['start', 'control'].includes(action);
  if (role === '商品助理') return ['create', 'editRule'].includes(action);
  if (role === '订单客服') return action === 'order';
  if (role === '数据复盘') return false;
  return false;
}

function upsertLot(list: Lot[], lot: Lot) {
  const exists = list.some((item) => item.id === lot.id);
  return exists ? list.map((item) => item.id === lot.id ? lot : item) : [lot, ...list];
}

function PermissionGuard({ allowed, children, fallback = null }: { allowed: boolean; children: ReactNode; fallback?: ReactNode }) {
  return allowed ? <>{children}</> : <>{fallback}</>;
}


type LinkStatus = '连接中' | '已连接' | '重连中' | '已断开';
type LinkDiagnosticEvent = { seq: number; time: string; type: string; lotId?: string; detail: string };

function serverOffsetText(snapshot: RoomSnapshot | null) {
  if (!snapshot?.serverTimeUnixMs) return '待同步';
  const offset = getServerOffsetMs(snapshot.serverTimeUnixMs);
  return `${offset >= 0 ? '+' : ''}${offset}ms`;
}

function pushLinkEvent(list: LinkDiagnosticEvent[], event: LinkDiagnosticEvent) {
  return [event, ...list].slice(0, 50);
}

function makeLinkEvent(seq: number, type: string, detail: string, lotId?: string): LinkDiagnosticEvent {
  return { seq, type, detail, lotId, time: new Date().toLocaleTimeString('zh-CN', { hour12: false }) };
}

function AuctionManagementPage() {
  const [lots, setLots] = useState<Lot[]>([]);
  const [snapshot, setSnapshot] = useState<RoomSnapshot | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [wsState, setWsState] = useState<LinkStatus>('连接中');
  const [lastHeartbeat, setLastHeartbeat] = useState('未收到');
  const [reconnectCount, setReconnectCount] = useState(0);
  const [lastEventType, setLastEventType] = useState('暂无');
  const [lastEventSeq, setLastEventSeq] = useState(0);
  const [linkEvents, setLinkEvents] = useState<LinkDiagnosticEvent[]>([]);
  const [linkLogOpen, setLinkLogOpen] = useState(false);
  const [activeTab, setActiveTab] = useState<AuctionUiStatus>('今日队列');
  const [filters, setFilters] = useState<AuctionFilters>(emptyFilters);
  const [selectedLot, setSelectedLot] = useState<Lot | null>(null);
  const [cancelTarget, setCancelTarget] = useState<Lot | null>(null);
  const [actionMessage, setActionMessage] = useState(() => {
    const params = new URLSearchParams(location.search);
    return params.get('queued') === '1' || params.get('created') === '1' ? '拍品已加入本场队列' : '';
  });
  const { toasts, showToast } = useStudioToast();
  const [, setCountdownTick] = useState(0);
  const currentLot = snapshot?.currentLot || lots.find((lot) => uiStatusOfLot(lot) === '竞拍中') || null;
  const nextLot = lots.find((lot) => ['准备中', '待开拍'].includes(uiStatusOfLot(lot))) || null;

  const syncRoom = async () => {
    setLoading(true);
    setError('');
    try {
      const [nextLots, nextSnapshot] = await Promise.all([listLots(currentHostRoom.id), getRoomSnapshot(currentHostRoom.id)]);
      setLots(nextLots);
      setSnapshot(nextSnapshot);
      setLastHeartbeat(new Date().toLocaleTimeString('zh-CN'));
      setLastEventType('ROOM_SNAPSHOT');
      setLastEventSeq((seq) => { const next = seq + 1; setLinkEvents((events) => pushLinkEvent(events, makeLinkEvent(next, 'ROOM_SNAPSHOT', '手动重新同步房间快照'))); return next; });
    } catch (e) {
      showToast({ tone: 'danger', title: '数据加载失败，请稍后重试', description: resultMessage(e) });
      setError(resultMessage(e));
      setLots([]);
      setSnapshot(null);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    void syncRoom();
    const params = new URLSearchParams(location.search);
    if (params.get('queued') === '1' || params.get('created') === '1') {
      showToast({ id: 'lot-created', tone: 'success', title: '拍品已加入本场队列', description: '队列、讲解卡和规则快照已同步到本场拍品工作台。' });
      history.replaceState(null, '', '/admin/auctions');
    }
  }, []);

  useEffect(() => {
    let timer = 0;
    let frame = 0;
    const tick = (time: number) => {
      if (time - timer >= 100) {
        timer = time;
        setCountdownTick((value) => value + 1);
      }
      frame = window.requestAnimationFrame(tick);
    };
    frame = window.requestAnimationFrame(tick);
    return () => window.cancelAnimationFrame(frame);
  }, []);

  useEffect(() => {
    let socket: WebSocket | null = null;
    try {
      socket = new WebSocket(`${WS_BASE}/ws/rooms/${encodeURIComponent(currentHostRoom.id)}`);
      socket.onopen = () => { setWsState('已连接'); setLastHeartbeat(new Date().toLocaleTimeString('zh-CN')); };
      socket.onclose = () => { setWsState('重连中'); setReconnectCount((count) => count + 1); };
      socket.onerror = () => { setWsState('已断开'); setReconnectCount((count) => count + 1); };
      socket.onmessage = (message) => {
        const event = normalizeAuctionEvent(JSON.parse(message.data)) as AuctionEvent;
        if (event.roomId && event.roomId !== currentHostRoom.id) return;
        if (!handledAuctionEventTypes.has(event.type)) return;
        setLastHeartbeat(new Date().toLocaleTimeString('zh-CN'));
        setLastEventType(event.type);
        setLastEventSeq((seq) => { const next = seq + 1; setLinkEvents((events) => pushLinkEvent(events, makeLinkEvent(next, event.type, event.reason || event.lot?.title || event.bid?.nickname || '房间事件', event.lotId))); return next; });
        if (event.snapshot) setSnapshot(event.snapshot);
        if (event.lot) setLots((current) => upsertLot(current, event.lot as Lot));
      };
    } catch {
      setWsState('已断开');
    }
    return () => socket?.close();
  }, []);

  const filteredLots = useMemo(() => lots.filter((lot) => {
    const status = uiStatusOfLot(lot);
    const keyword = filters.query.trim().toLowerCase();
    if (activeTab === '历史拍品' && !['已成交', '异常取消'].includes(status)) return false;
    if (!['今日队列', '历史拍品'].includes(activeTab) && status !== activeTab) return false;
    if (filters.status !== '全部状态' && status !== filters.status) return false;
    if (keyword && !`${lot.title} ${lot.id}`.toLowerCase().includes(keyword)) return false;
    const totalCards = lot.trustCards?.length || 0;
    const revealedCards = lot.trustCards?.filter((card) => card.revealed).length || 0;
    if (filters.briefing === '待补讲解卡' && totalCards > 0) return false;
    if (filters.briefing === '未全部展示' && totalCards > 0 && revealedCards >= totalCards) return false;
    if (filters.briefing === '已全部展示' && (!totalCards || revealedCards < totalCards)) return false;
    return true;
  }), [lots, activeTab, filters]);

  const startAuction = async (lot: Lot) => {
    setActionMessage('');
    setError('');
    try {
      const roomSnapshot = await getRoomSnapshot(currentHostRoom.id);
      setSnapshot(roomSnapshot);
      if (roomSnapshot.currentLot && roomSnapshot.currentLot.id !== lot.id) {
        const message = '当前直播间已有竞拍进行中，请结束后再开拍';
        setActionMessage(message);
        showToast({ id: 'live-conflict', tone: 'warning', title: message });
        return;
      }
      const updated = await startLot(lot.id);
      setLots((current) => upsertLot(current, updated));
      await syncRoom();
    } catch (e) {
      setError(resultMessage(e));
    }
  };

  const settleAuction = async (lot: Lot) => {
    setError('');
    try {
      const updated = await settleLot(lot.id);
      setLots((current) => upsertLot(current, updated));
      await syncRoom();
    } catch (e) {
      setError(resultMessage(e));
    }
  };

  const confirmCancel = async (lot: Lot, reason: string) => {
    setError('');
    try {
      const updated = await cancelLot(lot.id, reason);
      setLots((current) => upsertLot(current, updated));
      setCancelTarget(null);
      await syncRoom();
    } catch (e) {
      setError(resultMessage(e));
    }
  };

  return <section className="auctionMgmtPage">
    <StudioToastViewport toasts={toasts} />
    <AuctionManagementHeader currentLot={currentLot} onSync={syncRoom} syncing={loading} />
    <RealtimeSyncCapsule wsState={wsState} snapshot={snapshot} lastHeartbeat={lastHeartbeat} reconnectCount={reconnectCount} lastEventType={lastEventType} lastEventSeq={lastEventSeq} onSync={syncRoom} onOpenLog={() => setLinkLogOpen(true)} />
    {error ? <div className="auctionMgmtNotice danger"><AlertTriangle size={16} />真实接口暂不可用：{error}</div> : null}
    {actionMessage ? <div className="auctionMgmtNotice"><ShieldAlert size={16} />{actionMessage}</div> : null}
    <AuctionQueueTopCards lots={lots} currentLot={currentLot} nextLot={nextLot} snapshot={snapshot} wsState={wsState} lastHeartbeat={lastHeartbeat} onCancel={setCancelTarget} onStart={startAuction} />
    <AuctionStatusTabs active={activeTab} setActive={setActiveTab} lots={lots} />
    <AuctionFilterBar filters={filters} setFilters={setFilters} lots={lots} />
    <AuctionManagementTable lots={filteredLots} loading={loading} error={error} hasAnyLot={lots.length > 0} currentLot={currentLot} serverTimeUnixMs={snapshot?.serverTimeUnixMs} onDetail={setSelectedLot} onCancel={setCancelTarget} onStart={startAuction} onSettle={settleAuction} onRetry={syncRoom} />
    {selectedLot ? <AuctionDetailDrawer lot={selectedLot} snapshot={snapshot} onClose={() => setSelectedLot(null)} /> : null}
    {cancelTarget ? <CancelAuctionDialog lot={cancelTarget} onClose={() => setCancelTarget(null)} onConfirm={confirmCancel} /> : null}
    {linkLogOpen ? <LinkEventLogDrawer events={linkEvents} onClose={() => setLinkLogOpen(false)} /> : null}
  </section>;
}


function RealtimeSyncCapsule({ wsState, snapshot, lastHeartbeat, reconnectCount, lastEventType, lastEventSeq, onSync, onOpenLog }: { wsState: LinkStatus; snapshot: RoomSnapshot | null; lastHeartbeat: string; reconnectCount: number; lastEventType: string; lastEventSeq: number; onSync: () => void; onOpenLog: () => void }) {
  return <section className="realtimeSyncCapsule"><div><Wifi size={18} /><span>实时同步状态</span><StatusBadge label={wsState} tone={wsState === '已连接' ? 'success' : wsState === '重连中' ? 'warning' : 'danger'} /></div><div className="syncCapsuleMetrics"><span>平均延迟 <b>{currentHostRoom.latency}</b></span><span>最近心跳 <b>{lastHeartbeat}</b></span><span>重连次数 <b>{reconnectCount}</b></span><span>服务器偏移 <b>{serverOffsetText(snapshot)}</b></span><span>最近事件 <b>{lastEventType}</b></span><span>事件序号 <b>#{lastEventSeq}</b></span></div><div><button type="button" onClick={() => void onSync()}>重新同步房间快照</button><button type="button" onClick={onOpenLog}>查看事件日志</button></div></section>;
}

function LinkEventLogDrawer({ events, onClose }: { events: LinkDiagnosticEvent[]; onClose: () => void }) {
  return <aside className="auctionDrawer linkEventDrawer"><div className="drawerMask" onClick={onClose} /><section><header><div><p>当前直播间</p><h3>实时链路事件日志</h3><span>最近 50 条客户端可见 WebSocket / snapshot 事件</span></div><button type="button" onClick={onClose}>关闭</button></header><div className="linkEventList">{events.length ? events.map((event) => <div key={`${event.seq}-${event.time}`}><b>#{event.seq}</b><span>{event.type}</span><small>{event.time} · {event.lotId || currentHostRoom.id}</small><p>{event.detail}</p></div>) : <p>暂无事件。等待 WebSocket 事件或手动同步房间快照。</p>}</div></section></aside>;
}

function AuctionManagementHeader({ currentLot, onSync, syncing }: { currentLot: Lot | null; onSync: () => void; syncing: boolean }) {
  return <section className="auctionMgmtHeader"><StudioSectionHeader eyebrow="当前直播间 / 今日排品" title="本场拍品队列" description="管理当前直播间今日待拍、正在拍、已成交和异常取消的拍品。" actions={<><a className="studioButton studioButton-primary studioButton-md" href="/admin/auctions/create">添加拍品</a><a className={`studioButton studioButton-secondary studioButton-md ${currentLot ? '' : 'disabled'}`} href={currentLot ? `/admin/auctions/${currentLot.id}/control` : '#'}>进入中控台</a><StudioButton type="button" variant="secondary" loading={syncing} onClick={onSync}>{syncing ? '同步中...' : '同步队列'}</StudioButton></>} /></section>;
}

function AuctionQueueTopCards({ lots, currentLot, nextLot, snapshot, wsState, lastHeartbeat, onCancel, onStart }: { lots: Lot[]; currentLot: Lot | null; nextLot: Lot | null; snapshot: RoomSnapshot | null; wsState: string; lastHeartbeat: string; onCancel: (lot: Lot) => void; onStart: (lot: Lot) => void }) {
  const settled = lots.filter((lot) => uiStatusOfLot(lot) === '已成交').length;
  const abnormal = lots.filter((lot) => uiStatusOfLot(lot) === '异常取消').length;
  const waiting = lots.filter((lot) => ['准备中', '待开拍'].includes(uiStatusOfLot(lot))).length;
  return <section className="queueTopCards">
    <article className={`queueTopCard current ${currentLot ? 'isLive' : 'isEmpty'}`}><header><span><Radio size={18} />当前竞拍</span>{currentLot ? <StatusBadge label={lotLiveStageText(currentLot)} /> : <StatusBadge label="空闲" tone="neutral" />}</header>{currentLot ? <><h3>{currentLot.title}</h3><div className="queuePriceLine"><span>当前价</span><b>{moneyText(currentLot.currentPrice)}</b><small>{secondsLeftText(currentLot, snapshot?.serverTimeUnixMs)}</small></div><p>领先用户：{currentLot.leadingNickname || '暂无'} · 出价 {snapshot?.recentBids?.length || 0} 次</p><div className="queueTopActions"><a className="studioButton studioButton-primary studioButton-sm" href={`/admin/auctions/${currentLot.id}/control`}>进入中控台</a><button type="button" onClick={() => onCancel(currentLot)}>异常取消</button></div></> : <><h3>当前没有正在拍</h3><p>可以从下一件拍品开始，或继续完善今日队列。</p><a className="studioButton studioButton-primary studioButton-sm" href="/admin/auctions/create">添加拍品</a></>}</article>
    <article className={`queueTopCard next ${nextLot ? 'hasNext' : 'isEmpty'}`}><header><span><ChevronRight size={18} />下一件拍品</span>{nextLot ? <StatusBadge label={uiStatusOfLot(nextLot)} /> : <StatusBadge label="暂无" tone="neutral" />}</header>{nextLot ? <><h3>{nextLot.title}</h3><div className="queueRulePills"><span>起拍 <b>{moneyText(nextLot.rule.startPrice)}</b></span><span>加价 <b>{moneyText(nextLot.rule.minIncrement)}</b></span><span>封顶 <b>{capPriceText(nextLot)}</b></span></div><p>预计 {formatDuration(nextLot.rule.durationSeconds)} · 讲解卡 {nextLot.trustCards?.length || 0} 张</p><div className="queueTopActions"><button type="button" className="studioButton studioButton-secondary studioButton-sm" disabled={Boolean(currentLot)} onClick={() => void onStart(nextLot)}>{currentLot ? '等待当前结束' : '开始竞拍'}</button><a className="studioButton studioButton-soft studioButton-sm" href="/admin/auctions/create">编辑规则</a></div></> : <><h3>待开拍为空</h3><p>添加拍品后会进入本场队列，主播只需要按顺序控场。</p></>}</article>
    <article className="queueTopCard health"><header><span><ShieldCheck size={18} />队列健康</span><StatusBadge label={wsState} tone={wsState === '已连接' ? 'success' : wsState === '重连中' ? 'warning' : 'danger'} /></header><div className="queueHealthGrid"><span>今日队列<b>{lots.length}</b></span><span>待拍<b>{waiting}</b></span><span>已成交<b>{settled}</b></span><span>异常取消<b>{abnormal}</b></span></div><p>最近心跳 {lastHeartbeat} · 平均延迟 {currentHostRoom.latency} · 单直播间固定队列</p></article>
  </section>;
}

function CurrentRoomOverview({ wsState, lastHeartbeat, snapshot }: { wsState: string; lastHeartbeat: string; snapshot: RoomSnapshot | null }) {
  const user = currentAuth().user;
  return <section className="currentRoomOverview"><header><Radio size={21} /><div><p>当前固定直播间</p><h3>{currentHostRoom.name}</h3><span>一个主播空间只绑定一个直播间，本页不提供直播间切换或直播间筛选。</span></div><StatusBadge label="已绑定" tone="success" /></header><div className="roomMetricGrid"><div><span>roomId</span><b>{currentHostRoom.id}</b></div><div><span>主播主账号</span><b>{currentHostRoom.owner}</b></div><div><span>当前账号</span><b>{user?.username || currentTeamAccount.username}</b><small>{currentTeamAccount.role}</small></div><div><span>在线观众</span><b>{snapshot?.ranking?.length ? `${Math.max(currentHostRoom.online, snapshot.ranking.length)}+` : currentHostRoom.online}</b></div><div><span>WebSocket</span><b>{wsState}</b></div><div><span>平均延迟</span><b>{currentHostRoom.latency}</b></div><div><span>心跳时间</span><b>{lastHeartbeat}</b></div></div></section>;
}

function CurrentLotCard({ lot, snapshot, onCancel }: { lot: Lot | null; snapshot: RoomSnapshot | null; onCancel: (lot: Lot) => void }) {
  if (!lot) return <article className="auctionFocusCard empty"><h3>当前没有正在拍的拍品</h3><p>当前直播间同一时间只能有一件正在竞拍，可从今日队列开始下一件。</p><a className="laPrimaryBtn" href="/admin/auctions/create">添加拍品</a></article>;
  return <article className="auctionFocusCard live"><header><div><p>当前 LIVE 竞拍</p><h3>{lot.title}</h3></div><StatusBadge label={uiStatusOfLot(lot)} /></header><div className="currentLotPrice"><span>当前价</span><b>{moneyText(lot.currentPrice)}</b><small>领先用户：{lot.leadingNickname || '暂无'}</small></div><div className="lotMiniMetrics"><span className={countdownToneClass(lot, snapshot?.serverTimeUnixMs)}>剩余时间 <b>{secondsLeftText(lot, snapshot?.serverTimeUnixMs)}</b></span><span>参与人数 <b>{snapshot?.ranking?.length || 0}</b></span><span>出价次数 <b>{snapshot?.recentBids?.length || 0}</b></span></div><div className="auctionCardActions"><a className="laPrimaryBtn" href={`/admin/auctions/${lot.id}/control`}>进入控场</a><button type="button" onClick={() => window.location.assign('/admin/bids')}>查看出价</button><PermissionGuard allowed={canRole(currentTeamAccount.role, 'cancel')}><button type="button" className="danger" onClick={() => onCancel(lot)}>异常取消</button></PermissionGuard></div></article>;
}

function NextLotCard({ lot, hasLive, onStart }: { lot: Lot | null; hasLive: boolean; onStart: (lot: Lot) => void }) {
  if (!lot) return <article className="auctionFocusCard empty"><h3>今日队列暂无待开拍拍品</h3><p>当前直播间今日待开拍队列为空。</p></article>;
  const disabled = hasLive || !canRole(currentTeamAccount.role, 'start');
  return <article className="auctionFocusCard next"><header><div><p>下一场未开始</p><h3>{lot.title}</h3></div><StatusBadge label={uiStatusOfLot(lot)} /></header><div className="nextRuleGrid"><span>从多少钱开始拍<b>{moneyText(lot.rule.startPrice)}</b></span><span>每次至少加多少钱<b>{moneyText(lot.rule.minIncrement)}</b></span><span>封顶价<b>{capPriceText(lot)}</b></span><span>预计时长<b>{formatDuration(lot.rule.durationSeconds)}</b></span><span>执行账号<b>{currentTeamAccount.username}</b></span></div>{hasLive ? <small className="startBlocked">当前直播间已有正在拍，结束后才能开始下一场。</small> : null}<div className="auctionCardActions"><button type="button" className="laPrimaryBtn" disabled={disabled} onClick={() => void onStart(lot)}>开始竞拍</button><PermissionGuard allowed={canRole(currentTeamAccount.role, 'editRule')}><a href="/admin/auctions/create">编辑规则</a></PermissionGuard><button type="button">调整顺序</button></div></article>;
}

function AuctionStatCards({ lots }: { lots: Lot[]; wsState: string }) {
  const waiting = lots.filter((lot) => ['准备中', '待开拍'].includes(uiStatusOfLot(lot))).length;
  const live = lots.filter((lot) => lot.status === 'LOT_STATUS_LIVE').length;
  const settled = lots.filter((lot) => uiStatusOfLot(lot) === '已成交').length;
  const abnormal = lots.filter((lot) => uiStatusOfLot(lot) === '异常取消').length;
  return <section className="auctionMgmtStats"><StatCard icon={<Clock3 />} label="待开拍" value={waiting} hint="当前固定直播间队列" tone="info" /><StatCard icon={<Radio />} label="进行中" value={live} hint="同一时间最多 1 场" tone="success" /><StatCard icon={<Trophy />} label="今日成交" value={settled} hint="成交状态待真实接口接入" tone="purple" /><StatCard icon={<ShieldAlert />} label="异常竞拍" value={abnormal} hint="需填写原因后取消" tone="danger" /><StatCard icon={<Wifi />} label="WebSocket 延迟" value={currentHostRoom.latency} hint="仅当前直播间" tone="warning" /></section>;
}

function AuctionStatusTabs({ active, setActive, lots }: { active: AuctionUiStatus; setActive: (value: AuctionUiStatus) => void; lots: Lot[] }) {
  const count = (tab: AuctionUiStatus) => {
    if (tab === '今日队列') return lots.length;
    if (tab === '历史拍品') return lots.filter((lot) => ['已成交', '异常取消'].includes(uiStatusOfLot(lot))).length;
    return lots.filter((lot) => uiStatusOfLot(lot) === tab).length;
  };
  return <nav className="auctionStatusTabs" aria-label="队列状态筛选">{auctionStatusTabs.map((tab) => <button key={tab} type="button" className={active === tab ? 'active' : ''} onClick={() => setActive(tab)}>{tab}<b>{count(tab)}</b></button>)}</nav>;
}

function AuctionFilterBar({ filters, setFilters, lots }: { filters: AuctionFilters; setFilters: (filters: AuctionFilters) => void; lots: Lot[] }) {
  const operators = ['全部创建人', ...Array.from(new Set(lots.map(() => currentTeamAccount.username)))];
  const statusOptions = ['全部状态', '准备中', '待开拍', '竞拍中', '已成交', '异常取消'];
  const briefingOptions = ['全部讲解卡', '待补讲解卡', '未全部展示', '已全部展示'];
  return <section className="auctionFilterBar queueFilters" aria-label="队列筛选">
    <label><Search size={15} /><input value={filters.query} onChange={(e) => setFilters({ ...filters, query: e.target.value })} placeholder="搜索拍品名 / 竞拍 ID" /></label>
    <select aria-label="状态" value={filters.status} onChange={(e) => setFilters({ ...filters, status: e.target.value })}>{statusOptions.map((item) => <option key={item}>{item}</option>)}</select>
    <select aria-label="创建人" value={filters.operator} onChange={(e) => setFilters({ ...filters, operator: e.target.value })}>{operators.map((item) => <option key={item}>{item}</option>)}</select>
    <select aria-label="讲解卡状态" value={filters.briefing} onChange={(e) => setFilters({ ...filters, briefing: e.target.value })}>{briefingOptions.map((item) => <option key={item}>{item}</option>)}</select>
    <select aria-label="时间" value={filters.created} onChange={(e) => setFilters({ ...filters, created: e.target.value })}><option>全部时间</option><option>今天</option><option>近 7 天</option></select>
  </section>;
}

function AuctionManagementTable({ lots, loading, error, hasAnyLot, currentLot, serverTimeUnixMs, onDetail, onCancel, onStart, onSettle, onRetry }: { lots: Lot[]; loading: boolean; error: string; hasAnyLot: boolean; currentLot: Lot | null; serverTimeUnixMs?: number | string; onDetail: (lot: Lot) => void; onCancel: (lot: Lot) => void; onStart: (lot: Lot) => void; onSettle: (lot: Lot) => void; onRetry: () => void }) {
  if (loading) return <StudioTableSkeleton className="auctionMgmtSkeleton" rows={6} columns={7} />;
  if (error && !hasAnyLot) return <StudioErrorState className="auctionMgmtEmpty" icon={<AlertTriangle size={40} />} title="本场拍品队列加载失败" description={error} action={<><StudioButton type="button" variant="secondary" icon={<RefreshCw size={15} />} onClick={onRetry}>重试加载</StudioButton><a className="laPrimaryBtn" href="/admin/auctions/create">添加拍品</a></>} />;
  if (!lots.length) return <AuctionQueueEmptyState filtered={hasAnyLot} onRetry={onRetry} />;
  const nextId = lots.find((lot) => ['准备中', '待开拍'].includes(uiStatusOfLot(lot)))?.id;
  return <section className="auctionQueueList" aria-label="本场拍品队列列表">{lots.map((lot, index) => <AuctionQueueRow key={lot.id} lot={lot} index={index} currentLot={currentLot} nextId={nextId} serverTimeUnixMs={serverTimeUnixMs} onDetail={onDetail} onCancel={onCancel} onStart={onStart} onSettle={onSettle} />)}</section>;
}

function AuctionQueueEmptyState({ filtered, onRetry }: { filtered: boolean; onRetry: () => void }) {
  return <section className="queueEmptyWorkbench"><div className="queueEmptyIcon"><Package size={34} /></div><div><p>Queue Setup</p><h3>{filtered ? '没有符合筛选条件的拍品' : '今日队列还没有排品，但工作台已就绪'}</h3><span>{filtered ? '可以调整状态、讲解卡状态或时间筛选，也可以重新同步房间快照。' : '先添加拍品，或从拍品库把已准备好的商品加入本场队列；开拍、讲解卡和成交都会在这里串起来。'}</span></div><div className="queueEmptyActions"><a className="studioButton studioButton-primary studioButton-md" href="/admin/auctions/create">添加拍品</a><a className="studioButton studioButton-secondary studioButton-md" href="/admin/products">从拍品库加入</a><StudioButton type="button" variant="soft" icon={<RefreshCw size={15} />} onClick={onRetry}>同步队列</StudioButton></div><div className="queueEmptyChecklist"><span><CheckCircle2 size={15} /> 固定直播间已绑定</span><span><CheckCircle2 size={15} /> 实时同步链路可检测</span><span><CheckCircle2 size={15} /> 支持待开拍 / 竞拍中 / 成交 / 异常取消流转</span></div></section>;
}

function makeQueueRowPreviewSample(): Partial<Lot>[] {
  return [
    { id: 'sample-ready', title: '示例：待开拍珠宝拍品', status: 'LOT_STATUS_DRAFT' as Lot['status'] },
    { id: 'sample-live', title: '示例：竞拍中拍品', status: 'LOT_STATUS_LIVE' as Lot['status'] },
  ];
}

function AuctionQueueRow({ lot, index, currentLot, nextId, serverTimeUnixMs, onDetail, onCancel, onStart, onSettle }: { lot: Lot; index: number; currentLot: Lot | null; nextId?: string; serverTimeUnixMs?: number | string; onDetail: (lot: Lot) => void; onCancel: (lot: Lot) => void; onStart: (lot: Lot) => void; onSettle: (lot: Lot) => void }) {
  void makeQueueRowPreviewSample;
  const status = uiStatusOfLot(lot);
  const isCurrent = Boolean(currentLot?.id === lot.id || status === '竞拍中');
  const isNext = Boolean(!isCurrent && lot.id === nextId);
  const rowClass = ['queueRowCard', isCurrent ? 'isCurrent' : '', isNext ? 'isNext' : '', status === '异常取消' ? 'isCancelled' : ''].filter(Boolean).join(' ');
  return <article className={rowClass} onClick={() => onDetail(lot)}>
    <div className="queueRowLeft"><span className="queueNo">#{String(index + 1).padStart(2, '0')}</span><img src={lot.imageUrl || '/vite.svg'} alt={lot.title} /><div><h3>{lot.title}</h3><div className="queueTags"><StatusBadge label={status} /><span>{lot.id}</span><span>规则 v{lot.version || 1}</span><span>{briefingReadinessText(lot)}</span></div></div></div>
    <div className="queueRowMiddle"><span><b>状态进度</b>{statusProgressText(lot, serverTimeUnixMs)}</span><span><b>开拍时间</b>{dateTimeText(lot.startedAtUnixMs)}</span><span><b>起拍 / 加价</b>{moneyText(lot.rule.startPrice)} / {moneyText(lot.rule.minIncrement)}</span><span><b>封顶 / 时长</b>{capPriceText(lot)} / {formatDuration(lot.rule.durationSeconds)}</span></div>
    <div className="queueRowRight"><div className="queueAccount"><b>{currentTeamAccount.username}</b><span>{currentTeamAccount.role}</span></div>{orderStateText(lot)}<div onClick={(e) => e.stopPropagation()}><AuctionRowActions lot={lot} hasLive={Boolean(currentLot && currentLot.id !== lot.id)} onDetail={onDetail} onCancel={onCancel} onStart={onStart} onSettle={onSettle} /></div></div>
  </article>;
}

function briefingReadinessText(lot: Lot) {
  const total = lot.trustCards?.length || 0;
  const revealed = lot.trustCards?.filter((card) => card.revealed).length || 0;
  if (!total) return '待补讲解卡';
  return `${revealed}/${total} 已展示`;
}

function statusProgressText(lot: Lot, serverTimeUnixMs?: number | string) {
  const status = uiStatusOfLot(lot);
  if (lot.status === 'LOT_STATUS_LIVE') return `倒计时 ${secondsLeftText(lot, serverTimeUnixMs)}${serverTimeUnixMs ? '' : '（本地 fallback）'}`;
  if (status === '已成交') return `成交时间 ${dateTimeText(lot.settledAtUnixMs)}`;
  if (status === '异常取消') return lot.cancelReason || '已取消';
  return `开拍时间 ${dateTimeText(lot.startedAtUnixMs)}`;
}

function orderStateText(lot: Lot) {
  const status = uiStatusOfLot(lot);
  if (status === '已成交') return <div className="orderState"><b>成交订单待接入</b><span>成交价 {moneyText(lot.finalPrice)}</span></div>;
  if (lotLiveStageText(lot) === '可落锤') return <div className="orderState"><b>等待成交</b><span>可落锤生成成交</span></div>;
  if (status === '异常取消') return <div className="orderState danger"><b>异常原因</b><span>{lot.cancelReason || '已取消'}</span></div>;
  return <span className="mutedText">未成交</span>;
}

function AuctionRowActions({ lot, hasLive, onDetail, onCancel, onStart, onSettle }: { lot: Lot; hasLive: boolean; onDetail: (lot: Lot) => void; onCancel: (lot: Lot) => void; onStart: (lot: Lot) => void; onSettle: (lot: Lot) => void }) {
  const status = uiStatusOfLot(lot);
  const canStartLot = ['准备中', '待开拍'].includes(status);
  const isLive = status === '竞拍中';
  return <div className="auctionRowActions"><button type="button" onClick={() => onDetail(lot)}>详情</button>{canStartLot ? <><PermissionGuard allowed={canRole(currentTeamAccount.role, 'editRule')}><a href="/admin/auctions/create">编辑规则</a></PermissionGuard><PermissionGuard allowed={canRole(currentTeamAccount.role, 'start')}><button type="button" disabled={hasLive} onClick={() => void onStart(lot)}>开始竞拍</button></PermissionGuard></> : null}{isLive ? <><PermissionGuard allowed={canRole(currentTeamAccount.role, 'control')}><a href={`/admin/auctions/${lot.id}/control`}>进入中控台</a></PermissionGuard><PermissionGuard allowed={canRole(currentTeamAccount.role, 'settle')}><button type="button" onClick={() => void onSettle(lot)}>落锤成交</button></PermissionGuard></> : null}{status === '已成交' ? <><button type="button">成交处理</button><a href="/admin/bids">出价</a></> : null}{status === '异常取消' ? <><button type="button">原因</button><a href="/admin/auctions/create">复制重发</a></> : null}<details className="queueMoreActions"><summary>更多</summary><div><button type="button">预览讲解</button><a href="/admin/bids">查看出价</a>{status !== '异常取消' && status !== '已成交' ? <PermissionGuard allowed={canRole(currentTeamAccount.role, 'cancel')}><button type="button" className="danger" onClick={() => onCancel(lot)}>异常取消</button></PermissionGuard> : null}<button type="button">查看日志</button></div></details></div>;
}

function AuctionDetailDrawer({ lot, snapshot, onClose }: { lot: Lot; snapshot: RoomSnapshot | null; onClose: () => void }) {
  const [tab, setTab] = useState<DetailTab>('概览');
  const bids = snapshot?.currentLot?.id === lot.id ? snapshot.recentBids : [];
  return <aside className="auctionDrawer"><div className="drawerMask" onClick={onClose} /><section><header><div><p>竞拍详情</p><h3>{lot.title}</h3><span>{lot.id}</span></div><button type="button" onClick={onClose}>关闭</button></header><nav>{(['概览', '规则快照', '实时出价', '操作日志', '成交订单'] as DetailTab[]).map((item) => <button key={item} type="button" className={tab === item ? 'active' : ''} onClick={() => setTab(item)}>{item}</button>)}</nav>{tab === '概览' ? <div className="drawerOverview"><StatusBadge label={uiStatusOfLot(lot)} /><b>{moneyText(lot.currentPrice)}</b><p>{lot.description || '暂无描述'}</p><span>领先用户：{lot.leadingNickname || '暂无'}</span><span>规则版本：v{lot.version || 1}</span></div> : null}{tab === '规则快照' ? <AuctionRuleSnapshot lot={lot} /> : null}{tab === '实时出价' ? <AuctionRealtimeBidList bids={bids} leadingUserId={lot.leadingUserId} /> : null}{tab === '操作日志' ? <div className="drawerTodo">日志接口待接入，不构造假日志。</div> : null}{tab === '成交订单' ? <div className="drawerTodo">{uiStatusOfLot(lot) === '已成交' ? '成交订单待接入' : '当前竞拍未成交，暂无成交订单。'}</div> : null}</section></aside>;
}

function AuctionRuleSnapshot({ lot }: { lot: Lot }) {
  const rule = lot.rule;
  const items = [['从多少钱开始拍', moneyText(rule.startPrice)], ['每次至少加多少钱', moneyText(rule.minIncrement)], ['竞拍时长', formatDuration(rule.durationSeconds)], ['封顶价', capPriceText(lot)], ['最后出价自动延时窗口', `${rule.antiSnipeWindowSeconds}s`], ['每次延长时长', `${rule.antiSnipeExtendSeconds}s`], ['最大延时次数', `${rule.maxExtendCount}`], ['规则版本', `v${lot.version || 1}`], ['创建人', '待接入'], ['最后修改人', '待接入']];
  return <div className="ruleSnapshotGrid">{items.map(([label, value]) => <div key={label}><span>{label}</span><b>{value}</b></div>)}</div>;
}

function AuctionRealtimeBidList({ bids, leadingUserId }: { bids: Bid[]; leadingUserId: string }) {
  if (!bids.length) return <div className="drawerTodo">实时出价列表等待 WebSocket / snapshot 返回。</div>;
  return <div className="drawerBidList">{bids.map((bid) => <div key={bid.id}><span>{bid.nickname || bid.userId}</span><b>{moneyText(bid.amount)}</b><small>{bid.userId === leadingUserId ? '领先' : '非领先'} · {dateTimeText(bid.createdAtUnixMs)} · 客户端延迟待接入 · 服务端校验有效 · 幂等 Key 待接入</small></div>)}</div>;
}

function CancelAuctionDialog({ lot, onClose, onConfirm }: { lot: Lot; onClose: () => void; onConfirm: (lot: Lot, reason: string) => Promise<void> }) {
  const [reason, setReason] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const submit = async () => {
    if (!reason.trim()) return;
    setSubmitting(true);
    try { await onConfirm(lot, reason.trim()); } finally { setSubmitting(false); }
  };
  return <div className="cancelDialog"><div onClick={onClose} /><section><header><AlertTriangle size={22} /><div><h3>异常取消竞拍</h3><p>必须填写原因，取消后会写入审计并广播当前直播间。</p></div></header><b>{lot.title}</b><textarea value={reason} onChange={(e) => setReason(e.target.value)} rows={4} placeholder="请输入异常取消原因，例如拍品信息异常、库存问题、系统风控拦截等" /><footer><button type="button" onClick={onClose}>返回</button><button type="button" className="danger" disabled={!reason.trim() || submitting} onClick={() => void submit()}>{submitting ? '提交中...' : '确认异常取消'}</button></footer></section></div>;
}


type AuctionLifecycleState = 'DRAFT' | 'READY' | 'SCHEDULED' | 'PREHEATING' | 'LIVE' | 'EXTENDED' | 'SOLD' | 'UNSOLD' | 'CANCELLED' | 'ABNORMAL';
type PreviewMode = '默认态' | '领先态' | '被超越态' | '竞拍延时态' | '成交态';
type AuctionCreateForm = {
  productSource: '选择已有拍品' | '添加拍品';
  productName: string;
  mainImageName: string;
  mainImageUrl: string;
  mainImageAssetId: string;
  carouselImages: string[];
  category: '珠宝' | '艺术品' | '奢侈品' | '潮玩' | '其他';
  description: string;
  flawDescription: string;
  certificateInfo: string;
  detailNotes: string;
  priceReference: string;
  serviceNotes: string;
  estimate: number;
  stock: number;
  tags: string[];
  afterSale: string;
  startPrice: number;
  bidStep: number;
  durationSeconds: number;
  capPrice: number | '';
  autoExtend: boolean;
  extendTriggerSeconds: number;
  extendSeconds: number;
  maxExtendTimes: number;
  depositEnabled: boolean;
  depositAmount: number | '';
  bidCooldownMs: number;
  cancelPermission: '主播主账号' | '授权场控';
  roomId: string;
  responsibleAccount: string;
  startTime: string;
  warmupEnabled: boolean;
  warmupMinutes: number;
  visibility: '全部观众' | '指定粉丝等级' | '白名单';
  wsTopic: string;
  lifecycle: AuctionLifecycleState;
  draftLotId: string;
};

type AutoSaveStatus = 'idle' | 'dirty' | 'saving' | 'saved' | 'failed';
type ValidationIssue = { level: 'error' | 'warning'; text: string };
type AuctionTip = { tone: 'success' | 'warning' | 'error'; text: string };

const createSteps = ['拍品资料', '竞拍玩法', '主播讲解', '加入队列'];
const productTags = ['稀缺', '限量', '保真', '福利场', '秒杀场'];
const lifecycleStates: AuctionLifecycleState[] = ['DRAFT', 'READY', 'SCHEDULED', 'PREHEATING', 'LIVE', 'EXTENDED', 'SOLD', 'UNSOLD', 'CANCELLED', 'ABNORMAL'];
const previewModes: PreviewMode[] = ['默认态', '领先态', '被超越态', '竞拍延时态', '成交态'];
const currentHostRoom = { id: 'room-jewel-01', name: '小鹿珠宝直播间', owner: '小鹿珠宝直播团队', anchor: '主播主账号', online: 386, latency: '38ms' };
const responsibleAccounts = ['当前账号（主播主账号）', 'team_ada（场控）', 'team_lux（场控）', 'team_order（订单客服）'];

const initialAuctionCreateForm: AuctionCreateForm = {
  productSource: '添加拍品',
  productName: '',
  mainImageName: '',
  mainImageUrl: '',
  mainImageAssetId: '',
  carouselImages: [],
  category: '其他',
  description: '',
  flawDescription: '',
  certificateInfo: '',
  detailNotes: '',
  priceReference: '',
  serviceNotes: '',
  estimate: 0,
  stock: 1,
  tags: [],
  afterSale: '',
  startPrice: 0,
  bidStep: 50,
  durationSeconds: 600,
  capPrice: '',
  autoExtend: true,
  extendTriggerSeconds: 10,
  extendSeconds: 15,
  maxExtendTimes: 5,
  depositEnabled: false,
  depositAmount: '',
  bidCooldownMs: 800,
  cancelPermission: '主播主账号',
  roomId: currentHostRoom.id,
  responsibleAccount: '当前账号（主播主账号）',
  startTime: '',
  warmupEnabled: false,
  warmupMinutes: 15,
  visibility: '全部观众',
  wsTopic: `auction.room.${currentHostRoom.id}.AUC-DRAFT`,
  lifecycle: 'DRAFT',
  draftLotId: '',
};

const AUCTION_CREATE_DRAFT_KEY = `liveAuction.createLotDraft.${currentHostRoom.id}.v2`;

function loadAuctionCreateDraft(): AuctionCreateForm {
  try {
    const raw = localStorage.getItem(AUCTION_CREATE_DRAFT_KEY);
    if (!raw) return initialAuctionCreateForm;
    const parsed = JSON.parse(raw) as Partial<AuctionCreateForm>;
    return { ...initialAuctionCreateForm, ...parsed, roomId: currentHostRoom.id };
  } catch {
    return initialAuctionCreateForm;
  }
}

function saveAuctionCreateDraft(form: AuctionCreateForm) {
  localStorage.setItem(AUCTION_CREATE_DRAFT_KEY, JSON.stringify(form));
}

function clearAuctionCreateDraft() {
  localStorage.removeItem(AUCTION_CREATE_DRAFT_KEY);
}

function formatMoney(value: number | '') {
  if (value === '') return '未设置';
  return `¥${Number(value).toLocaleString('zh-CN')}`;
}

function formatDuration(seconds: number) {
  if (!Number.isFinite(seconds)) return '-';
  if (seconds >= 60) return `${Math.floor(seconds / 60)} 分钟`;
  return `${seconds} 秒`;
}


function auctionMoney(amount: number | ''): Money {
  return { amount: Number(amount || 0), currency: 'CNY' };
}

function trustCardContent(value: string, fallback = '待补充') {
  const content = value.trim();
  return content || fallback;
}

function buildTrustCards(form: AuctionCreateForm): CreateLotRequest['trustCards'] {
  return [
    { id: 'certificate-card', type: 'TRUST_CARD_TYPE_CERTIFICATE', title: '证书卡', content: trustCardContent(form.certificateInfo, '证书信息待补充') },
    { id: 'flaw-card', type: 'TRUST_CARD_TYPE_FLAW', title: '瑕疵说明卡', content: trustCardContent(form.flawDescription, '瑕疵说明待补充') },
    { id: 'detail-card', type: 'TRUST_CARD_TYPE_DETAIL', title: '细节图卡', content: trustCardContent(form.detailNotes, '细节说明待补充'), imageUrl: form.mainImageUrl || undefined },
    { id: 'service-card', type: 'TRUST_CARD_TYPE_SERVICE', title: '售后说明卡', content: trustCardContent(form.serviceNotes, '售后说明待补充') },
    { id: 'price-ref-card', type: 'TRUST_CARD_TYPE_PRICE_REF', title: '价格参考卡', content: trustCardContent(form.priceReference, '价格参考待补充') },
  ];
}

function trustCardPrepared(value: string) {
  const content = value.trim();
  return Boolean(content) && !content.includes('待补充');
}

function isStableImageUrl(url: string) {
  const normalized = url.trim();
  return Boolean(normalized) && !normalized.startsWith('blob:') && !normalized.startsWith('data:');
}


function fromFormToCreateLotRequest(form: AuctionCreateForm): CreateLotRequest {
  return {
    roomId: currentHostRoom.id,
    title: form.productName.trim(),
    description: form.description.trim(),
    imageUrl: isStableImageUrl(form.mainImageUrl) ? form.mainImageUrl : '',
    rule: {
      startPrice: auctionMoney(form.startPrice),
      minIncrement: auctionMoney(form.bidStep),
      ...(form.capPrice !== '' ? { capPrice: auctionMoney(form.capPrice) } : {}),
      durationSeconds: form.durationSeconds,
      antiSnipeWindowSeconds: form.autoExtend ? form.extendTriggerSeconds : 0,
      antiSnipeExtendSeconds: form.autoExtend ? form.extendSeconds : 0,
      maxExtendCount: form.autoExtend ? form.maxExtendTimes : 0,
    },
    trustCards: buildTrustCards(form),
  };
}

function validateAuctionCreateForm(form: AuctionCreateForm): ValidationIssue[] {
  const issues: ValidationIssue[] = [];
  if (!form.productName.trim()) issues.push({ level: 'error', text: '拍品名称不能为空' });
  if (!form.mainImageName || !isStableImageUrl(form.mainImageUrl)) issues.push({ level: 'error', text: '需要上传稳定可用的拍品主图' });
  if (!form.description.trim()) issues.push({ level: 'warning', text: '拍品介绍较弱，建议补充材质、成色和竞拍亮点' });
  if (form.estimate <= 0) issues.push({ level: 'warning', text: '拍品估值未设置或过低' });
  if (form.stock < 1) issues.push({ level: 'error', text: '库存数量至少为 1' });
  if (form.startPrice < 0) issues.push({ level: 'error', text: '从多少钱开始拍必须大于等于 0' });
  if (form.bidStep <= 0) issues.push({ level: 'error', text: '每次至少加多少钱必须大于 0' });
  if (form.capPrice !== '' && form.capPrice <= form.startPrice) issues.push({ level: 'error', text: '到这个价自动成交必须大于从多少钱开始拍' });
  if (form.durationSeconds < 60) issues.push({ level: 'error', text: '竞拍时长不能小于 60 秒' });
  if (form.autoExtend && (form.extendSeconds < 10 || form.extendSeconds > 30)) issues.push({ level: 'error', text: '每次延长时长必须在 10-30 秒' });
  if (form.autoExtend && form.extendTriggerSeconds > form.durationSeconds) issues.push({ level: 'error', text: '延时触发窗口不能大于竞拍时长' });
  if (form.depositEnabled && form.depositAmount !== '' && form.capPrice !== '' && form.depositAmount > form.capPrice) issues.push({ level: 'error', text: '保证金不能大于到这个价自动成交' });
  if (form.capPrice !== '' && form.capPrice < form.estimate * 0.5) issues.push({ level: 'warning', text: '到这个价自动成交低于估值 50%，发布时建议二次确认' });
  if (form.durationSeconds < 180) issues.push({ level: 'warning', text: '竞拍时长偏短，可能影响充分出价' });
  if (form.roomId !== currentHostRoom.id) issues.push({ level: 'error', text: '当前主播空间只能绑定唯一直播间' });
  if (!form.wsTopic.trim()) issues.push({ level: 'error', text: 'WebSocket 房间 topic 未生成' });
  return issues;
}


function stepIssueKeywords(step: number) {
  if (step === 0) return ['拍品名称', '拍品主图', '拍品介绍', '拍品估值', '库存数量'];
  if (step === 1) return ['从多少钱开始拍', '每次至少加多少钱', '到这个价自动成交', '竞拍时长', '延长时长', '触发窗口', '保证金'];
  if (step === 2) return ['直播间', 'WebSocket', '开拍时间'];
  return [];
}

function issuesForStep(issues: ValidationIssue[], step: number) {
  const keywords = stepIssueKeywords(step);
  if (!keywords.length) return issues;
  return issues.filter((issue) => keywords.some((keyword) => issue.text.includes(keyword)));
}

function canEnterPublishStep(issues: ValidationIssue[], targetStep: number) {
  for (let index = 0; index < targetStep; index += 1) {
    if (issuesForStep(issues, index).some((issue) => issue.level === 'error')) return false;
  }
  return true;
}

function AutoSaveIndicator({ status, savedAt, onRetry }: { status: AutoSaveStatus; savedAt: string; onRetry: () => void }) {
  const content: Record<AutoSaveStatus, { label: string; tone: string }> = {
    idle: { label: '未保存', tone: 'idle' },
    dirty: { label: '未保存', tone: 'dirty' },
    saving: { label: '保存中...', tone: 'saving' },
    saved: { label: savedAt ? `已自动保存 ${savedAt}` : '已自动保存', tone: 'saved' },
    failed: { label: '保存失败，点击重试', tone: 'failed' },
  };
  const next = content[status];
  return <button type="button" className={`autoSaveIndicator ${next.tone}`} onClick={status === 'failed' ? onRetry : undefined} disabled={status !== 'failed'}><span />{next.label}</button>;
}

function stepStatusLabel(stepIssues: ValidationIssue[], index: number, currentStep: number) {
  if (stepIssues.some((issue) => issue.level === 'error')) return '有风险';
  if (stepIssues.some((issue) => issue.level === 'warning')) return '有风险';
  if (index < currentStep) return '已完成';
  return '未完成';
}

function PublishStepper({ step, setStep, canEnter, issues }: { step: number; setStep: (step: number) => void; canEnter: (step: number) => boolean; issues: ValidationIssue[] }) {
  return <nav className="publishStepper" aria-label="添加拍品步骤">{createSteps.map((label, index) => {
    const locked = !canEnter(index);
    const stepIssues = issuesForStep(issues, index);
    const status = stepStatusLabel(stepIssues, index, step);
    return <button key={label} type="button" disabled={locked} className={`${index === step ? 'active' : index < step ? 'done' : ''} ${locked ? 'locked' : ''} ${status === '有风险' ? 'risk' : ''}`} onClick={() => { if (!locked) setStep(index); }}><b>{index + 1}</b><span>{label}</span><small>{status}</small></button>;
  })}</nav>;
}

function Field({ label, children, hint, error, className = '' }: { label: string; children: ReactNode; hint?: string; error?: string; className?: string }) {
  return <StudioField className={`auctionField ${className}`.trim()} label={label} help={hint} error={error}>{children}</StudioField>;
}

function Segmented<T extends string>({ value, options, onChange }: { value: T; options: T[]; onChange: (value: T) => void }) {
  return <div className="auctionSegmented">{options.map((option) => <button key={option} type="button" className={value === option ? 'active' : ''} onClick={() => onChange(option)}>{option}</button>)}</div>;
}

function ProductInfoStep({ form, update, generateDescription, mainImageUploading, mainImageError, onMainImageSelect, onRemoveMainImage }: { form: AuctionCreateForm; update: (patch: Partial<AuctionCreateForm>) => void; generateDescription: () => void; mainImageUploading: boolean; mainImageError: string; onMainImageSelect: (file: File) => void; onRemoveMainImage: () => void }) {
  const uploadMain = (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    event.target.value = '';
    if (!file) return;
    onMainImageSelect(file);
  };
  const uploadCarousel = (event: React.ChangeEvent<HTMLInputElement>) => {
    const files = Array.from(event.target.files || []).slice(0, 6);
    if (!files.length) return;
    update({ carouselImages: files.map((file) => file.name).slice(0, 6) });
  };
  return <section className="auctionStepCard productInfoWorkbench"><header><div><p>Step 1</p><h3>拍品资料</h3></div><span>先把拍品素材和基础交易信息准备好，讲解卡细节可以在第三步继续完善。</span></header><div className="productInfoRedesign">
    <div className="productMediaColumn"><StudioCard title="拍品素材" subtitle="Media" padding="md"><Field className="fieldMainImage" label="拍品主图" hint="先上传到 OSS 临时目录；保存拍品后绑定，未保存或删除会清理 OSS 临时文件。" error={mainImageError}><div className={`auctionUpload mainImageUpload ${mainImageUploading ? 'isUploading' : ''} ${form.mainImageUrl ? 'hasImage' : ''}`}>{form.mainImageUrl ? <img src={form.mainImageUrl} alt="拍品主图回显" /> : <ShoppingBag size={34} />}<b>{mainImageUploading ? '图片上传中...' : form.mainImageName || '点击上传主图'}</b>{form.mainImageUrl ? <span>点击可重新上传</span> : null}<input type="file" accept="image/*" disabled={mainImageUploading} onChange={uploadMain} /></div>{form.mainImageUrl ? <div className="mainImageControl"><span>{form.mainImageAssetId ? 'OSS 临时图片，保存后自动绑定' : (isStableImageUrl(form.mainImageUrl) ? '主图已上传，URL 稳定' : '主图地址不可用于提交')}</span><button type="button" onClick={onRemoveMainImage} disabled={mainImageUploading}>删除主图</button></div> : null}</Field><Field className="fieldCarousel" label="拍品轮播图" hint="最多 6 张"><div className="auctionUpload carouselUpload"><Package size={22} /><b>{form.carouselImages.length ? `${form.carouselImages.length} 张已选择` : '上传轮播图'}</b><input type="file" accept="image/*" multiple onChange={uploadCarousel} /></div></Field></StudioCard></div>
    <div className="productBaseColumn"><StudioCard title="基础信息" subtitle="Basic Info" padding="md"><div className="productBaseGrid"><Field className="fieldSource" label="拍品来源"><Segmented value={form.productSource} options={['选择已有拍品', '添加拍品']} onChange={(value) => update({ productSource: value })} /></Field><Field className="fieldTitle" label="拍品名称"><input value={form.productName} onChange={(e) => update({ productName: e.target.value })} placeholder="请输入竞拍拍品名称" /></Field><Field label="拍品分类"><select value={form.category} onChange={(e) => update({ category: e.target.value as AuctionCreateForm['category'] })}>{['珠宝', '艺术品', '奢侈品', '潮玩', '其他'].map((item) => <option key={item}>{item}</option>)}</select></Field><Field label="拍品估值"><input type="number" value={form.estimate} onChange={(e) => update({ estimate: Number(e.target.value) })} /></Field><Field label="库存数量"><input type="number" min={1} value={form.stock} onChange={(e) => update({ stock: Number(e.target.value) })} /></Field><Field className="fieldTags" label="拍品标签"><div className="auctionTagPicker">{productTags.map((tag) => <button key={tag} type="button" className={form.tags.includes(tag) ? 'active' : ''} onClick={() => update({ tags: form.tags.includes(tag) ? form.tags.filter((item) => item !== tag) : [...form.tags, tag] })}>{tag}</button>)}</div></Field></div></StudioCard></div>
    <div className="productDetailRow"><StudioCard title="拍品介绍与保障说明" subtitle="Story & Service" padding="md"><div className="productDetailGrid"><Field className="fieldDescription" label="拍品介绍" hint="AI 生成内容需用户确认后填入"><textarea value={form.description} onChange={(e) => update({ description: e.target.value })} rows={5} placeholder="描述拍品材质、成色、亮点和竞拍价值" /><button type="button" className="auctionInlineAi" onClick={generateDescription}><Sparkles size={15} /> AI 生成拍品介绍</button></Field><Field label="瑕疵说明"><textarea value={form.flawDescription} onChange={(e) => update({ flawDescription: e.target.value })} rows={4} placeholder="如实记录磨损、划痕、缺件或使用痕迹" /></Field><Field label="证书信息"><textarea value={form.certificateInfo} onChange={(e) => update({ certificateInfo: e.target.value })} rows={4} placeholder="证书编号、鉴定机构、材质证明等" /></Field><Field label="售后说明"><textarea value={form.serviceNotes} onChange={(e) => update({ serviceNotes: e.target.value, afterSale: e.target.value })} rows={4} placeholder="退换、支付、保价发货、客服承诺等" /></Field></div></StudioCard></div>
  </div></section>;
}

function RuleSimulator({ form }: { form: AuctionCreateForm }) {
  const path = Array.from({ length: 4 }, (_, index) => form.startPrice + form.bidStep * index);
  const capPrice = form.capPrice;
  const hitCap = capPrice !== '' && path.some((price) => price >= capPrice);
  return <div className="ruleSimulator"><header><p>Rule Simulator</p><h4>规则模拟器</h4></header><div className="rulePath">{path.map((price, index) => <span key={price}>{formatMoney(price)}{index < path.length - 1 ? <ChevronRight size={14} /> : null}</span>)}</div><div className="ruleSimulationNotes"><span>{form.capPrice !== '' ? `到达 ${formatMoney(form.capPrice)} 自动成交` : '未设置到这个价自动成交，竞拍按倒计时结束成交'}</span><span>{form.autoExtend ? `最后 ${form.extendTriggerSeconds} 秒出价会延长 ${form.extendSeconds} 秒` : '最后出价自动延时关闭，倒计时结束即结算'}</span><span>{hitCap ? '当前路径已触发封顶成交' : '路径未触发到这个价自动成交'}</span></div></div>;
}

function AuctionRuleStep({ form, update, issues }: { form: AuctionCreateForm; update: (patch: Partial<AuctionCreateForm>) => void; issues: ValidationIssue[] }) {
  const issueText = (keyword: string) => issues.find((issue) => issue.level === 'error' && issue.text.includes(keyword))?.text;
  return <section className="auctionStepCard ruleWorkbench"><header><p>Step 2</p><h3>竞拍玩法</h3><span>用三组规则配置直播间竞拍节奏：价格、时间延时和适合本场的玩法模板。</span></header><div className="ruleCardGrid">
    <StudioCard title="价格规则" subtitle="Price" padding="md"><div className="ruleFieldGrid"><Field label="起拍价" error={issueText('从多少钱开始拍')}><input type="number" value={form.startPrice} onChange={(e) => update({ startPrice: Number(e.target.value) })} /></Field><Field label="加价幅度" error={issueText('每次至少加多少钱')}><input type="number" value={form.bidStep} onChange={(e) => update({ bidStep: Number(e.target.value) })} /></Field><Field label="封顶价"><input type="number" value={form.capPrice} placeholder="可选" onChange={(e) => update({ capPrice: e.target.value === '' ? '' : Number(e.target.value) })} /></Field></div></StudioCard>
    <StudioCard title="时间规则" subtitle="Timing" padding="md"><div className="ruleFieldGrid"><Field label="竞拍时长（秒）" error={issueText('竞拍时长')}><input type="number" value={form.durationSeconds} onChange={(e) => update({ durationSeconds: Number(e.target.value) })} /></Field><Field label="延时窗口（秒）" error={issueText('触发窗口')}><input type="number" value={form.extendTriggerSeconds} onChange={(e) => update({ extendTriggerSeconds: Number(e.target.value) })} /></Field><Field label="每次延长（秒）" error={issueText('延长时长')}><input type="number" min={10} max={30} value={form.extendSeconds} onChange={(e) => update({ extendSeconds: Number(e.target.value) })} /></Field><Field label="最大延时次数"><input type="number" value={form.maxExtendTimes} onChange={(e) => update({ maxExtendTimes: Number(e.target.value) })} /></Field><Field className="fieldWide" label="最后出价自动延时"><Segmented value={form.autoExtend ? '开启' : '关闭'} options={['开启', '关闭']} onChange={(value) => update({ autoExtend: value === '开启' })} /></Field></div></StudioCard>
    <StudioCard title="玩法模板" subtitle="Template" padding="md"><div className="playTemplateList"><button type="button" onClick={() => update({ startPrice: 0, bidStep: 50, durationSeconds: 180 })}><b>福利快闪</b><span>短时高频，适合福利款快速成交</span></button><button type="button" onClick={() => update({ startPrice: 0, bidStep: 200, durationSeconds: 600 })}><b>高价值慢拍</b><span>更长讲解时间，适合珠宝/奢品</span></button><button type="button" onClick={() => update({ startPrice: 0, bidStep: 20, durationSeconds: 240 })}><b>0元起拍引流</b><span>从 0 元启动，提升观众参与</span></button></div></StudioCard>
  </div><RuleSimulator form={form} /></section>;
}

function RoomHealthCard({ form }: { form: AuctionCreateForm }) {
  return <div className="roomHealthCard"><header><Radio size={19} /><div><b>Room Health</b><span>{currentHostRoom.id}</span></div></header><div><span>直播间状态</span><StatusBadge label="在线" /></div><div><span>WebSocket</span><StatusBadge label="可用" /></div><div><span>当前在线人数</span><strong>{currentHostRoom.online}</strong></div><div><span>平均延迟</span><strong>{currentHostRoom.latency}</strong></div><div><span>房间隔离</span><StatusBadge label="已开启" /></div><div><span>心跳保活</span><StatusBadge label="已开启" /></div></div>;
}

function FixedRoomCard() {
  return <div className="fixedRoomCard"><Radio size={22} /><div><span>当前主播唯一直播间</span><b>{currentHostRoom.name}</b><small>{currentHostRoom.owner} · {currentHostRoom.id}</small></div><StatusBadge label="已绑定" tone="success" /></div>;
}

function LiveRoomStep({ form, update }: { form: AuctionCreateForm; update: (patch: Partial<AuctionCreateForm>) => void }) {
  const nextTopic = `auction.room.${currentHostRoom.id}.${form.productName ? form.productName.replace(/\s+/g, '-').slice(0, 16) : 'AUC-DRAFT'}`;
  return <section className="auctionStepCard liveBriefingWorkbench"><header><p>Step 3</p><h3>主播讲解</h3><span>准备证书、瑕疵说明、细节图、价格参考和售后说明卡，直播中可一键展示给观众。</span></header><div className="briefingCardGrid">{buildTrustCards(form).map((card) => {
    const pending = card.content.includes('待补充');
    return <article key={card.id} className={`briefingCard ${pending ? 'pending' : 'ready'}`}><div><b>{card.title}</b><StatusBadge label={pending ? '待补充' : '已准备'} tone={pending ? 'warning' : 'success'} /></div><p>{card.content}</p><button type="button">编辑</button></article>;
  })}</div><div className="briefingConfigGrid"><StudioCard title="加入队列配置" subtitle="Queue" padding="md"><div className="auctionFormGrid twoCols">
    <Field label="固定直播间" hint="一个主播主账号只对应一个直播间，不在发布页切换直播间。"><FixedRoomCard /></Field>
    <Field label="本场负责人" hint="可选择当前账号或已授权的团队子账号，不再选择“主播”。"><select value={form.responsibleAccount} onChange={(e) => update({ responsibleAccount: e.target.value })}>{responsibleAccounts.map((account) => <option key={account}>{account}</option>)}</select></Field>
    <Field label="队列说明" hint="开拍动作不在添加拍品页执行，加入队列后到本场拍品队列页调整顺序或开始竞拍。"><div className="queueOnlyNotice"><CheckCircle2 size={16} /><span>本页只负责自动保存草稿与最终加入本场队列</span></div></Field>
    <Field label="预热展示"><Segmented value={form.warmupEnabled ? '开启' : '关闭'} options={['开启', '关闭']} onChange={(value) => update({ warmupEnabled: value === '开启' })} /></Field>
    <Field label="预热时间（分钟）"><input type="number" value={form.warmupMinutes} onChange={(e) => update({ warmupMinutes: Number(e.target.value) })} /></Field>
    <Field label="观众可见范围"><select value={form.visibility} onChange={(e) => update({ visibility: e.target.value as AuctionCreateForm['visibility'] })}><option>全部观众</option><option>指定粉丝等级</option><option>白名单</option></select></Field>
    <Field label="WebSocket 房间 topic 自动生成" hint="随直播间和竞拍活动生成，支持房间隔离"><div className="topicGenerator"><input value={form.wsTopic} onChange={(e) => update({ wsTopic: e.target.value })} /><button type="button" onClick={() => update({ wsTopic: nextTopic })}>重新生成</button></div></Field>
  </div></StudioCard><RoomHealthCard form={form} /></div></section>;
}

function SummaryBlock({ title, items }: { title: string; items: Array<[string, ReactNode]> }) {
  return <article className="publishSummaryBlock"><h4>{title}</h4>{items.map(([label, value]) => <div key={label}><span>{label}</span><b>{value}</b></div>)}</article>;
}

function PublishReviewStep({ form, issues }: { form: AuctionCreateForm; issues: ValidationIssue[] }) {
  return <section className="auctionStepCard"><header><p>Step 4</p><h3>加入队列</h3><span>确认拍品、规则、直播间、成交生成和异常处理策略。严重错误会阻断发布，风险提示需二次确认。</span></header><div className="publishReviewGrid">
    <SummaryBlock title="拍品信息摘要" items={[["拍品", form.productName], ["分类", form.category], ["估值", formatMoney(form.estimate)], ["库存", form.stock], ["标签", form.tags.join(' / ') || '未设置']]} />
    <SummaryBlock title="竞拍规则摘要" items={[["从多少钱开始拍", formatMoney(form.startPrice)], ["每次至少加多少钱", formatMoney(form.bidStep)], ["到这个价自动成交", formatMoney(form.capPrice)], ["竞拍时长", formatDuration(form.durationSeconds)], ["最后出价自动延时", form.autoExtend ? `结束前 ${form.extendTriggerSeconds}s 出价 +${form.extendSeconds}s` : '关闭']]} />
    <SummaryBlock title="加入队列摘要" items={[["直播间", currentHostRoom.name], ["本场负责人", form.responsibleAccount], ["队列动作", '加入本场队列'], ["可见范围", form.visibility], ["Topic", form.wsTopic]]} />
    <SummaryBlock title="成交订单生成策略" items={[["成交后自动生成订单", '生成成交订单'], ["规则快照", '写入成交订单'], ["支付方式", '模拟支付'], ["履约", '待支付后发货']]} />
    <SummaryBlock title="异常处理策略" items={[["异常取消", form.cancelPermission], ["锁冲突", '阻断重复成交'], ["断线恢复", '快照恢复'], ["延时广播", form.autoExtend ? '开启' : '关闭']]} />
    <article className="publishIssueBox"><h4>校验结果</h4>{issues.length ? issues.map((issue) => <div key={issue.text} className={issue.level}><AlertTriangle size={15} /><span>{issue.text}</span></div>) : <div className="success"><CheckCircle2 size={15} /><span>所有核心配置已通过校验</span></div>}</article>
  </div></section>;
}

function MobileAuctionPreview({ form, mode, setMode }: { form: AuctionCreateForm; mode: PreviewMode; setMode: (mode: PreviewMode) => void }) {
  const currentPrice = mode === '成交态' ? (form.capPrice || form.startPrice + form.bidStep * 8) : mode === '领先态' ? form.startPrice + form.bidStep * 4 : mode === '被超越态' ? form.startPrice + form.bidStep * 5 : form.startPrice + form.bidStep * 3;
  const feedback = mode === '领先态' ? '🎉 你正在领先' : mode === '被超越态' ? '⚡ 你已被超越' : mode === '竞拍延时态' ? `⏱ 最后出价自动延时 +${form.extendSeconds}s` : mode === '成交态' ? '🏆 竞拍成交，成交已生成' : '实时竞拍即将开始';
  return <div className="mobilePreviewWrap"><div className="previewModeTabs">{previewModes.map((item) => <button key={item} type="button" className={mode === item ? 'active' : ''} onClick={() => setMode(item)}>{item}</button>)}</div><div className="mobileAuctionPhone"><div className="phoneTop"><span>LiveAuction</span><b>{mode}</b></div><div className="phoneImage">{form.mainImageUrl ? <img src={form.mainImageUrl} alt="拍品主图预览" /> : <ShoppingBag size={58} />}</div><h4>{form.productName || '未命名拍品'}</h4><p>{form.tags.join(' · ') || '直播竞拍拍品'}</p><div className="phonePrice"><span>当前价</span><strong>{formatMoney(currentPrice)}</strong></div><div className="phoneRules"><span>起拍 {formatMoney(form.startPrice)}</span><span>加价 {formatMoney(form.bidStep)}</span><span>封顶 {formatMoney(form.capPrice)}</span></div><div className={`phoneFeedback ${mode === '被超越态' ? 'danger' : mode === '成交态' ? 'success' : ''}`}>{feedback}</div><button>立即出价</button><div className="phoneRanking"><span>#1 u_2718 {formatMoney(currentPrice)}</span><span>#2 u_6621 {formatMoney(Number(currentPrice) - form.bidStep)}</span></div></div></div>;
}

function RuleSummaryCard({ form }: { form: AuctionCreateForm }) {
  return <article className="stickyMiniCard"><h3>规则摘要</h3><div><span>从多少钱开始拍</span><b>{formatMoney(form.startPrice)}</b></div><div><span>每次至少加多少钱</span><b>{formatMoney(form.bidStep)}</b></div><div><span>竞拍时长</span><b>{formatDuration(form.durationSeconds)}</b></div><div><span>到这个价自动成交</span><b>{formatMoney(form.capPrice)}</b></div><div><span>最后出价自动延时</span><b>{form.autoExtend ? `${form.extendTriggerSeconds}s / +${form.extendSeconds}s` : '关闭'}</b></div></article>;
}

function PublishHealthCard({ form, issues }: { form: AuctionCreateForm; issues: ValidationIssue[] }) {
  const errors = issues.filter((issue) => issue.level === 'error');
  const warnings = issues.filter((issue) => issue.level === 'warning');
  const checks = [
    ['拍品信息完整', !errors.length],
    ['竞拍规则合法', !errors.length],
    ['直播间已绑定', form.roomId === currentHostRoom.id],
    ['WebSocket 房间已生成', Boolean(form.wsTopic.trim())],
    ['证书卡是否已准备', trustCardPrepared(form.certificateInfo)],
    ['瑕疵说明是否已准备', trustCardPrepared(form.flawDescription)],
    ['售后说明是否已准备', trustCardPrepared(form.serviceNotes)],
    ['成交后自动生成订单策略已开启', true],
  ] as const;
  return <article className="stickyMiniCard publishHealthCard"><h3>开拍检查</h3>{checks.map(([check, ok]) => <div key={check}><span>{check}</span><StatusBadge label={ok ? '通过' : '待补充'} tone={ok ? 'success' : 'warning'} /></div>)}{warnings.map((issue) => <p key={issue.text}><AlertTriangle size={14} />{issue.text}</p>)}</article>;
}

function ActionHintCard({ step, issues }: { step: number; issues: ValidationIssue[] }) {
  const errorCount = issues.filter((issue) => issue.level === 'error').length;
  const texts = ['先完成主图和基础信息，下一步配置竞拍玩法。', '确认价格和延时策略，右侧摘要会同步更新。', '检查主播讲解卡，待补充内容会在健康检查中提示。', '确认无严重错误后加入本场队列。'];
  return <article className="stickyMiniCard actionHintCard"><h3>当前动作提示</h3><p>{texts[step]}</p><StatusBadge label={errorCount ? `${errorCount} 个严重错误` : '可以继续'} tone={errorCount ? 'danger' : 'success'} /></article>;
}

function StickyPreviewPanel({ form, issues, previewMode, setPreviewMode, step }: { form: AuctionCreateForm; issues: ValidationIssue[]; previewMode: PreviewMode; setPreviewMode: (mode: PreviewMode) => void; step: number }) {
  return <aside className="stickyPreviewPanel"><MobileAuctionPreview form={form} mode={previewMode} setMode={setPreviewMode} /><RuleSummaryCard form={form} /><PublishHealthCard form={form} issues={issues} /><ActionHintCard step={step} issues={issues} /></aside>;
}

function AuctionCreatePage() {
  const [step, setStep] = useState(0);
  const [form, setForm] = useState<AuctionCreateForm>(() => loadAuctionCreateDraft());
  const [previewMode, setPreviewMode] = useState<PreviewMode>('默认态');
  const [autoSaveStatus, setAutoSaveStatus] = useState<AutoSaveStatus>(() => loadAuctionCreateDraft().draftLotId ? 'saved' : 'idle');
  const [autoSavedAt, setAutoSavedAt] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [submitError, setSubmitError] = useState('');
  const [mainImageUploading, setMainImageUploading] = useState(false);
  const [mainImageError, setMainImageError] = useState('');
  const [, setStepNotice] = useState('');
  const [tip, setTip] = useState<AuctionTip | null>(null);
  const savedAssetIds = useRef(new Set<string>());
  const autoSaveSeq = useRef(0);
  const { toasts, showToast } = useStudioToast(3000);

  const showStepNotice = (message: string) => {
    setStepNotice(message);
    showToast({ id: 'auction-step-notice', tone: 'warning', title: message });
  };
  const update = (patch: Partial<AuctionCreateForm>) => {
    setForm((current) => ({ ...current, ...patch }));
    setAutoSaveStatus('dirty');
  };
  const showTip = (nextTip: AuctionTip) => {
    setTip(nextTip);
    showToast({ tone: nextTip.tone === 'error' ? 'danger' : nextTip.tone, title: nextTip.text });
  };
  const cleanupTemporaryAsset = async (assetId: string, options?: { keepalive?: boolean; silent?: boolean }) => {
    if (!assetId || savedAssetIds.current.has(assetId)) return;
    try {
      await deleteUploadedImage(assetId, options);
    } catch (error) {
      if (!options?.silent) throw error;
    }
  };

  const saveCurrentDraft = async (input: AuctionCreateForm, options?: { silent?: boolean }) => {
    const seq = ++autoSaveSeq.current;
    setAutoSaveStatus('saving');
    try {
      const payload = fromFormToCreateLotRequest(input);
      let draftId = input.draftLotId;
      if (!draftId) {
        const draft = await createDraftLot({ roomId: currentHostRoom.id });
        draftId = draft.id;
      }
      const saved = await patchDraftLot(draftId, payload);
      const savedAt = new Date().toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit', hour12: false });
      savedAssetIds.current = new Set([input.mainImageAssetId].filter(Boolean));
      setForm((current) => {
        const next = { ...current, draftLotId: saved.id || draftId };
        saveAuctionCreateDraft(next);
        return next;
      });
      if (seq === autoSaveSeq.current) {
        setAutoSavedAt(savedAt);
        setAutoSaveStatus('saved');
      }
      if (!options?.silent) showToast({ id: 'draft-saved', tone: 'success', title: '草稿已自动保存' });
      return saved;
    } catch (error) {
      if (seq === autoSaveSeq.current) setAutoSaveStatus('failed');
      if (!options?.silent) showToast({ id: 'draft-save-failed', tone: 'danger', title: '保存失败，点击重试', description: resultMessage(error) });
      throw error;
    }
  };

  const uploadMainImage = async (file: File) => {
    const allowedTypes = ['image/jpeg', 'image/png', 'image/webp'];
    const allowedExt = /\.(jpe?g|png|webp)$/i;
    if (file.type && !allowedTypes.includes(file.type)) {
      const message = '请选择 JPG、PNG 或 WebP 图片';
      setMainImageError(message);
      showTip({ tone: 'error', text: message });
      return;
    }
    if (!file.type && !allowedExt.test(file.name)) {
      const message = '请选择 JPG、PNG 或 WebP 图片';
      setMainImageError(message);
      showTip({ tone: 'error', text: message });
      return;
    }
    if (file.size > 5 * 1024 * 1024) {
      const message = '图片不能超过 5MB';
      setMainImageError(message);
      showTip({ tone: 'error', text: message });
      return;
    }
    setMainImageUploading(true);
    setMainImageError('');
    showTip({ tone: 'warning', text: '正在上传拍品主图...' });
    const previousAssetId = form.mainImageAssetId;
    update({ mainImageName: file.name, mainImageUrl: '', mainImageAssetId: '' });
    try {
      if (previousAssetId) await cleanupTemporaryAsset(previousAssetId, { silent: true });
      const asset = await uploadImage(file, { roomId: currentHostRoom.id, bizType: 'lot_image' });
      update({ mainImageName: file.name, mainImageUrl: asset.imageUrl, mainImageAssetId: asset.id });
      showTip({ tone: 'success', text: '主图已上传到 OSS 临时区，自动保存后会绑定草稿' });
    } catch (e) {
      const message = resultMessage(e);
      setMainImageError(message);
      showToast({ id: 'image-upload-failed', tone: 'danger', title: '图片上传失败，请重试', description: message });
      console.error('[uploadImage] failed', e);
      update({ mainImageName: '', mainImageUrl: '', mainImageAssetId: '' });
    } finally {
      setMainImageUploading(false);
    }
  };
  const removeMainImage = () => {
    const assetId = form.mainImageAssetId;
    setMainImageError('');
    update({ mainImageName: '', mainImageUrl: '', mainImageAssetId: '' });
    if (assetId) {
      void cleanupTemporaryAsset(assetId, { silent: true }).then(() => showToast({ id: 'image-temp-deleted', tone: 'success', title: '临时图片已从 OSS 删除' }));
    }
  };
  const issues = useMemo(() => validateAuctionCreateForm(form), [form]);
  const currentStepErrors = issuesForStep(issues, step).filter((issue) => issue.level === 'error');
  const canEnter = (targetStep: number) => canEnterPublishStep(issues, targetStep);

  useEffect(() => {
    saveAuctionCreateDraft(form);
  }, [form]);

  useEffect(() => {
    if (autoSaveStatus !== 'dirty') return;
    const timer = window.setTimeout(() => {
      void saveCurrentDraft(form, { silent: true });
    }, 800);
    return () => window.clearTimeout(timer);
  }, [form, autoSaveStatus]);

  useEffect(() => {
    if (!tip) return;
    const timer = window.setTimeout(() => setTip(null), tip.tone === 'warning' ? 1800 : 4200);
    return () => window.clearTimeout(timer);
  }, [tip]);

  const retryAutoSave = () => {
    void saveCurrentDraft(form).catch(() => undefined);
  };

  const queueCurrentLot = async () => {
    if (mainImageUploading) {
      setStep(0);
      showStepNotice('拍品主图仍在处理，请等待完成后再加入队列。');
      return;
    }
    const nextIssues = validateAuctionCreateForm(form);
    if (nextIssues.some((issue) => issue.level === 'error')) {
      const firstInvalid = [0, 1, 2].find((index) => issuesForStep(nextIssues, index).some((issue) => issue.level === 'error')) ?? 3;
      setStep(firstInvalid);
      showStepNotice('存在未完成步骤，请修正当前步骤后再加入队列。');
      return;
    }
    setSubmitting(true);
    setSubmitError('');
    try {
      const saved = await saveCurrentDraft(form, { silent: true });
      const queued = await queueLot(saved.id);
      if (form.mainImageAssetId) savedAssetIds.current.add(form.mainImageAssetId);
      clearAuctionCreateDraft();
      showToast({ id: 'lot-queued', tone: 'success', title: '拍品已加入本场队列', description: `队列位置 #${queued.queuePosition || queued.lot.queuePosition || '-'}，正在前往本场拍品队列。` });
      window.setTimeout(() => { location.href = '/admin/auctions?queued=1'; }, 350);
    } catch (e) {
      const message = resultMessage(e);
      setSubmitError(message);
      showToast({ id: 'queue-failed', tone: 'danger', title: '加入本场队列失败', description: message });
      setSubmitting(false);
    }
  };

  const generateDescription = () => {
    const next = `${form.productName || '该拍品'}适合直播竞拍场景，具备${form.tags.join('、') || '稀缺'}等卖点。建议在开拍前强调估值 ${formatMoney(form.estimate)}、固定加价 ${formatMoney(form.bidStep)} 与封顶成交规则，提升观众决策效率。`;
    if (window.confirm('AI 已生成拍品介绍，是否确认填入？')) update({ description: next });
  };

  return <section className="auctionCreatePage">{tip ? <span className="auctionUploadTipBridge" aria-hidden="true" /> : null}<StudioToastViewport toasts={toasts} className="auctionCreateToastViewport" />
    <div className="auctionCreateTitleBar"><div><p>当前直播间 / 添加拍品</p><h2>添加拍品</h2><span>草稿静默自动保存，资料完成后只执行“加入本场队列”。</span></div><AutoSaveIndicator status={autoSaveStatus} savedAt={autoSavedAt} onRetry={retryAutoSave} /></div>
    <PublishStepper step={step} setStep={(next) => { if (canEnter(next)) { setStep(next); setStepNotice(''); } }} canEnter={canEnter} issues={issues} />
    <div className="auctionCreateLayout"><main className="auctionCreateMain">{submitError ? <div className="auctionMgmtNotice danger"><AlertTriangle size={16} />{submitError}</div> : null}{step === 0 && <ProductInfoStep form={form} update={update} generateDescription={generateDescription} mainImageUploading={mainImageUploading} mainImageError={mainImageError} onMainImageSelect={(file) => void uploadMainImage(file)} onRemoveMainImage={removeMainImage} />}{step === 1 && <AuctionRuleStep form={form} update={update} issues={issues} />}{step === 2 && <LiveRoomStep form={form} update={update} />}{step === 3 && <PublishReviewStep form={form} issues={issues} />}<div className="auctionStepNav"><StudioButton type="button" variant="secondary" disabled={step === 0 || submitting} onClick={() => setStep(Math.max(0, step - 1))}>上一步</StudioButton><StudioButton type="button" variant="primary" onClick={() => { if (step < 3) { if (currentStepErrors.length) { showStepNotice('请先完成当前步骤必填项和严重校验。'); return; } setStepNotice(''); setStep(step + 1); } else { void queueCurrentLot(); } }} disabled={submitting}>{step < 3 ? '下一步' : (submitting ? '正在加入本场队列...' : '加入本场队列')}</StudioButton></div></main><StickyPreviewPanel form={form} issues={issues} previewMode={previewMode} setPreviewMode={setPreviewMode} step={step} /></div></section>;
}


type ControlLog = { id: string; time: string; type: string; detail: string; level?: 'info' | 'warning' | 'danger' | 'success' };
type ControlDialog = 'cancel' | 'settle' | 'force-end' | null;

const playbookStages = [
  ['PLAYBOOK_STAGE_WARM_UP', '暖场'],
  ['PLAYBOOK_STAGE_TRUST_BLOCKED', '讲解卡展示'],
  ['PLAYBOOK_STAGE_BIDDING_ACTIVE', '正常出价'],
  ['PLAYBOOK_STAGE_DUEL_READY', '决胜准备'],
  ['PLAYBOOK_STAGE_DUEL_MODE', '决胜延时'],
  ['PLAYBOOK_STAGE_SETTLE_READY', '可落锤'],
] as const;

function playbookStageLabel(stage?: string) {
  return playbookStages.find(([key]) => key === stage)?.[1] || '暖场';
}

function controlNow() {
  return new Date().toLocaleTimeString('zh-CN', { hour12: false });
}

function logFromEvent(event: AuctionEvent): ControlLog {
  const map: Record<string, string> = {
    AUCTION_EVENT_TYPE_ROOM_SNAPSHOT: '房间快照同步',
    AUCTION_EVENT_TYPE_LOT_STARTED: '竞拍开始',
    AUCTION_EVENT_TYPE_LOT_UPDATED: '竞拍状态更新',
    AUCTION_EVENT_TYPE_BID_ACCEPTED: '出价接受',
    AUCTION_EVENT_TYPE_BID_REJECTED: '出价拒绝',
    AUCTION_EVENT_TYPE_RANKING_UPDATED: '排名更新',
    AUCTION_EVENT_TYPE_TRUST_REVEALED: '展示讲解卡',
    AUCTION_EVENT_TYPE_DUEL_STARTED: '进入决胜',
    AUCTION_EVENT_TYPE_DUEL_ENDED: '决胜结束',
    AUCTION_EVENT_TYPE_LOT_SETTLED: '落锤成交',
    AUCTION_EVENT_TYPE_LOT_CANCELLED: '异常取消',
  };
  const danger = event.type === 'AUCTION_EVENT_TYPE_LOT_CANCELLED' || event.type === 'AUCTION_EVENT_TYPE_BID_REJECTED';
  const success = event.type === 'AUCTION_EVENT_TYPE_LOT_SETTLED' || event.type === 'AUCTION_EVENT_TYPE_BID_ACCEPTED';
  return { id: event.id || `${event.type}-${Date.now()}`, time: controlNow(), type: map[event.type] || event.type, detail: event.bid ? `${event.bid.nickname || event.bid.userId} ${moneyText(event.bid.amount)}` : event.reason || event.lot?.title || currentHostRoom.name, level: danger ? 'danger' : success ? 'success' : 'info' };
}

function LiveControlPage() {
  const [snapshot, setSnapshot] = useState<RoomSnapshot | null>(null);
  const [lot, setLot] = useState<Lot | null>(null);
  const [lots, setLots] = useState<Lot[]>([]);
  const [wsState, setWsState] = useState<LinkStatus>('连接中');
  const [lastHeartbeat, setLastHeartbeat] = useState('未收到');
  const [reconnectCount, setReconnectCount] = useState(0);
  const [lastEventType, setLastEventType] = useState('暂无');
  const [lastEventSeq, setLastEventSeq] = useState(0);
  const [error, setError] = useState('');
  const [logs, setLogs] = useState<ControlLog[]>([]);
  const [feedback, setFeedback] = useState<string[]>([]);
  const [dialog, setDialog] = useState<ControlDialog>(null);

  const appendLog = (entry: Omit<ControlLog, 'id' | 'time'>) => setLogs((current) => [{ ...entry, id: `${Date.now()}-${Math.random()}`, time: controlNow() }, ...current].slice(0, 28));
  const pushFeedback = (text: string) => {
    setFeedback((current) => [text, ...current].slice(0, 4));
    window.setTimeout(() => setFeedback((current) => current.filter((item) => item !== text)), 3200);
  };

  const syncRoom = async () => {
    setError('');
    try {
      const [nextSnapshot, nextLots] = await Promise.all([getRoomSnapshot(currentHostRoom.id), listLots(currentHostRoom.id)]);
      setSnapshot(nextSnapshot);
      setLot(nextSnapshot.currentLot || null);
      setLots(nextLots);
      setLastHeartbeat(controlNow());
      setLastEventType('ROOM_SNAPSHOT');
      setLastEventSeq((seq) => seq + 1);
      appendLog({ type: '房间快照同步', detail: '已同步 currentLot / ranking / recentBids', level: 'success' });
    } catch (e) {
      setError(resultMessage(e));
      setSnapshot(null);
      setLot(null);
      setLots([]);
    }
  };

  useEffect(() => {
    void syncRoom();
    if (new URLSearchParams(location.search).get('created') === '1') {
      history.replaceState(null, '', '/admin/auctions');
    }
  }, []);

  useEffect(() => {
    let socket: WebSocket | null = null;
    try {
      socket = new WebSocket(`${WS_BASE}/ws/rooms/${encodeURIComponent(currentHostRoom.id)}`);
      socket.onopen = () => { setWsState('已连接'); setLastHeartbeat(controlNow()); appendLog({ type: '实时链路连接', detail: '当前固定直播间已连接', level: 'success' }); void syncRoom(); };
      socket.onclose = () => { setWsState('重连中'); setReconnectCount((count) => count + 1); appendLog({ type: '实时链路断开', detail: '进入 reconnecting，重连成功后自动恢复房间快照', level: 'warning' }); };
      socket.onerror = () => { setWsState('已断开'); setReconnectCount((count) => count + 1); };
      socket.onmessage = (message) => {
        const event = normalizeAuctionEvent(JSON.parse(message.data)) as AuctionEvent;
        if (event.roomId && event.roomId !== currentHostRoom.id) return;
        if (!handledControlEventTypes.has(event.type)) return;
        setLastHeartbeat(controlNow());
        setLastEventType(event.type);
        setLastEventSeq((seq) => seq + 1);
        if (event.snapshot) { setSnapshot(event.snapshot); setLot(event.snapshot.currentLot || null); }
        if (event.lot) setLot(event.lot);
        setLogs((current) => [logFromEvent(event), ...current].slice(0, 28));
        if (event.type === 'AUCTION_EVENT_TYPE_BID_ACCEPTED') pushFeedback('🎉 当前领先');
        if (event.type === 'AUCTION_EVENT_TYPE_RANKING_UPDATED') pushFeedback('⚡ 排名更新');
        if (event.type === 'AUCTION_EVENT_TYPE_LOT_UPDATED') pushFeedback('⏱ 最后出价自动延时 +15s');
        if (event.type === 'AUCTION_EVENT_TYPE_LOT_SETTLED') pushFeedback('🏆 竞拍成交');
        if (event.type === 'AUCTION_EVENT_TYPE_LOT_CANCELLED') pushFeedback('⚠ 异常取消');
      };
    } catch {
      setWsState('已断开');
    }
    return () => socket?.close();
  }, []);

  const revealNextTrustCard = async (cardId?: string) => {
    if (!lot) return;
    const card = cardId ? lot.trustCards?.find((item) => item.id === cardId) : lot.trustCards?.find((item) => !item.revealed);
    if (!card) { appendLog({ type: '展示讲解卡', detail: '没有未展示的讲解卡', level: 'warning' }); return; }
    try {
      const reply = await revealTrustCard(lot.id, card.id);
      setLot(reply.lot);
      appendLog({ type: '展示讲解卡', detail: card.title, level: 'success' });
    } catch (e) { appendLog({ type: '展示讲解卡失败', detail: resultMessage(e), level: 'danger' }); }
  };

  const enterDuel = async () => {
    if (!lot) return;
    try { const updated = await startDuel(lot.id); setLot(updated); appendLog({ type: '进入决胜', detail: '已请求进入决胜模式', level: 'success' }); }
    catch (e) { appendLog({ type: '进入决胜失败', detail: resultMessage(e), level: 'danger' }); }
  };

  const settleCurrentLot = async () => {
    if (!lot) return;
    try { const updated = await settleLot(lot.id); setLot(updated); setDialog(null); pushFeedback('🏆 竞拍成交'); appendLog({ type: '落锤成交', detail: `${lot.leadingNickname || '中标用户待同步'} · ${moneyText(lot.currentPrice)}`, level: 'success' }); }
    catch (e) { appendLog({ type: '落锤失败', detail: resultMessage(e), level: 'danger' }); }
  };

  const cancelCurrentLot = async (reason: string) => {
    if (!lot) return;
    try { const updated = await cancelLot(lot.id, reason); setLot(updated); setDialog(null); pushFeedback('⚠ 异常取消'); appendLog({ type: '异常取消', detail: reason, level: 'danger' }); }
    catch (e) { appendLog({ type: '异常取消失败', detail: resultMessage(e), level: 'danger' }); }
  };

  const nextLot = lots.find((item) => ['准备中', '待开拍'].includes(uiStatusOfLot(item)) && item.id !== lot?.id) || null;

  if (!lot) return <PreparedControlStage nextLot={nextLot} wsState={wsState} lastHeartbeat={lastHeartbeat} snapshot={snapshot} reconnectCount={reconnectCount} lastEventType={lastEventType} lastEventSeq={lastEventSeq} logs={logs} error={error} onSync={syncRoom} />;

  const status = uiStatusOfLot(lot);
  return <section className="liveControlPage">
    <ControlTopBar lot={lot} wsState={wsState} lastHeartbeat={lastHeartbeat} serverTime={snapshot?.serverTimeUnixMs} onSync={syncRoom} />
    {error ? <div className="auctionMgmtNotice danger"><AlertTriangle size={16} />{error}</div> : null}
    <div className="controlRoomGrid">
      <HostBriefingRail lot={lot} snapshot={snapshot} wsState={wsState} onReveal={revealNextTrustCard} />
      <main className="controlCenterRail"><PlaybookStageBar lot={lot} /><section className="commandArea"><PriceCommandBoard lot={lot} snapshot={snapshot} /><EmotionFeedbackLayer items={feedback} /><ControlActionDeck lot={lot} status={status} onReveal={revealNextTrustCard} onDuel={enterDuel} onCancel={() => setDialog('cancel')} onSettle={() => setDialog('settle')} onLog={appendLog} /></section></main>
      <aside className="controlRightRail"><RealtimeBidFeedPanel bids={snapshot?.recentBids || []} /><LiveRankingBoard ranking={snapshot?.ranking || []} leadingUserId={lot.leadingUserId} /><SystemHealthCard wsState={wsState} lastHeartbeat={lastHeartbeat} snapshot={snapshot} reconnectCount={reconnectCount} lastEventType={lastEventType} lastEventSeq={lastEventSeq} /></aside>
    </div>
    <div className="controlBottomGrid"><NextLotQueue lot={nextLot} currentStatus={status} /><ControlEventLog logs={logs} /></div>
    {dialog === 'cancel' ? <CancelLotDialog lot={lot} onClose={() => setDialog(null)} onConfirm={cancelCurrentLot} /> : null}
    {dialog === 'settle' ? <SettleLotDialog lot={lot} onClose={() => setDialog(null)} onConfirm={settleCurrentLot} /> : null}
  </section>;
}

function makePreparedDemoLot(): Lot {
  const now = Date.now();
  return {
    id: 'AUC-PREPARE-DEMO',
    roomId: currentHostRoom.id,
    title: '下一件演示拍品 · Vintage Cartier 手镯',
    description: '无 LIVE 时展示的预备态拍品卡：用于主播提前讲解卖点、确认规则、等待开拍。',
    imageUrl: '/vite.svg',
    status: 'LOT_STATUS_DRAFT',
    rule: { startPrice: { amount: 0, currency: 'CNY' }, minIncrement: { amount: 100, currency: 'CNY' }, capPrice: { amount: 12000, currency: 'CNY' }, durationSeconds: 90, antiSnipeWindowSeconds: 10, antiSnipeExtendSeconds: 10, maxExtendCount: 6 },
    currentPrice: { amount: 0, currency: 'CNY' },
    leadingUserId: '',
    leadingNickname: '',
    startedAtUnixMs: 0,
    endsAtUnixMs: now + 90_000,
    settledAtUnixMs: 0,
    winnerUserId: '',
    winnerNickname: '',
    finalPrice: { amount: 0, currency: 'CNY' },
    version: 1,
    trustCards: [
      { id: 'prep-card-cert', lotId: 'AUC-PREPARE-DEMO', type: 'TRUST_CARD_TYPE_CERTIFICATE', title: '证书与来源', content: '开拍前先展示证书、来源和保真承诺，降低观众决策成本。', revealed: false, revealedAtUnixMs: 0 },
      { id: 'prep-card-flaw', lotId: 'AUC-PREPARE-DEMO', type: 'TRUST_CARD_TYPE_FLAW', title: '瑕疵说明', content: '提前说明自然使用痕迹和售后规则，避免成交后争议。', revealed: false, revealedAtUnixMs: 0 },
    ],
    duelState: { active: false, lotId: 'AUC-PREPARE-DEMO', userAId: '', userANickname: '', userBId: '', userBNickname: '', startedAtUnixMs: 0, endsAtUnixMs: 0, extendCount: 0, maxExtendCount: 0 },
    playbookStage: 'PLAYBOOK_STAGE_WARM_UP',
  };
}

function PreparedControlStage({ nextLot, wsState, lastHeartbeat, snapshot, reconnectCount, lastEventType, lastEventSeq, logs, error, onSync }: { nextLot: Lot | null; wsState: LinkStatus; lastHeartbeat: string; snapshot: RoomSnapshot | null; reconnectCount: number; lastEventType: string; lastEventSeq: number; logs: ControlLog[]; error: string; onSync: () => void }) {
  const preparedLot = nextLot || makePreparedDemoLot();
  return <section className="liveControlPage preparedControlStage">
    <ControlTopBar lot={preparedLot} wsState={wsState} lastHeartbeat={lastHeartbeat} serverTime={snapshot?.serverTimeUnixMs} onSync={onSync} />
    {error ? <div className="auctionMgmtNotice danger"><AlertTriangle size={16} />当前无 LIVE，且同步失败：{error}<button type="button" onClick={() => void onSync()}>重试同步</button></div> : <div className="auctionMgmtNotice"><Clock3 size={16} />当前无 LIVE，已进入预备控场态：可检查下一件拍品、讲解卡、同步状态和空出价流。</div>}
    <div className="controlRoomGrid">
      <aside className="controlLeftRail hostBriefingRail"><header className="briefingRailTitle"><span>主播预备区</span><b>下一件拍品 / 提词器预备卡</b></header><RoomLivePreview lot={preparedLot} snapshot={snapshot} wsState={wsState} /><CurrentLotInfoCard lot={preparedLot} /><TeleprompterCard lot={preparedLot} /><TrustCardPanel lot={preparedLot} onReveal={() => undefined} /></aside>
      <main className="controlCenterRail"><PlaybookStageBar lot={preparedLot} /><section className="commandArea"><PreparedCommandCenter lot={preparedLot} snapshot={snapshot} /><div className="emotionFeedbackLayer prepared"><span>等待主播开拍</span><span>观众端将收到开拍提醒</span></div><PreparedActionDeck onSync={onSync} /></section></main>
      <aside className="controlRightRail"><RealtimeBidFeedPanel bids={[]} /><LiveRankingBoard ranking={[]} leadingUserId="" /><SystemHealthCard wsState={wsState} lastHeartbeat={lastHeartbeat} snapshot={snapshot} reconnectCount={reconnectCount} lastEventType={lastEventType} lastEventSeq={lastEventSeq} /></aside>
    </div>
    <div className="controlBottomGrid"><NextLotQueue lot={nextLot} currentStatus="准备中" /><ControlEventLog logs={logs} /></div>
  </section>;
}

function PreparedCommandCenter({ lot, snapshot }: { lot: Lot; snapshot: RoomSnapshot | null }) {
  return <section className="priceCommandBoard auctionCommandCenter preparedCommandCenter"><div className="priceCommandEyebrow"><span>Command Area · Ready</span><StatusBadge label="等待开拍" tone="warning" /></div><div className="currentPriceFocus"><p>当前价占位</p><strong className="livePriceBig">{moneyText(lot.currentPrice)}</strong><small>下一件：{lot.title}</small></div><span className="serverCountdown prepared"><Clock3 size={22} />等待开拍</span><div className="priceCommandMetrics"><span>起拍价<b>{moneyText(lot.rule.startPrice)}</b></span><span>最小加价<b>{moneyText(lot.rule.minIncrement)}</b></span><span>封顶价<b>{capPriceText(lot)}</b></span><span>同步状态<b>{snapshot?.serverTimeUnixMs ? '快照已同步' : '等待 snapshot'}</b></span></div></section>;
}

function PreparedActionDeck({ onSync }: { onSync: () => void }) {
  return <section className="controlActionDeck preparedActionDeck"><header><h3>预备操作</h3><StatusBadge label="无 LIVE" tone="warning" /></header><div className="controlActionsGrid"><a href="/admin/auctions" className="controlActionLink">查看本场队列</a><a href="/admin/auctions/create" className="controlActionLink">添加拍品</a><button type="button" onClick={() => void onSync()}>同步房间状态</button></div><div className="dangerActionStrip"><button type="button" disabled>异常取消</button><button type="button" disabled className="settleButton">落锤成交</button><button type="button" disabled>强制结束</button></div></section>;
}

function ControlTopBar({ lot, wsState, lastHeartbeat, serverTime, onSync }: { lot: Lot | null; wsState: string; lastHeartbeat: string; serverTime?: number | string; onSync: () => void }) {
  const user = currentAuth().user;
  const offsetText = serverTime ? `${getServerOffsetMs(serverTime)}ms` : '待同步';
  return <header className="controlTopBar controlStatusBar">
    <a className="controlBackLink" href="/admin/auctions">← 返回本场队列</a>
    <div className="controlTopIdentity"><p>直播间中控台 · Control Room</p><h2>{currentHostRoom.name}</h2><span>当前房间 {currentHostRoom.id} · {currentHostRoom.owner}</span></div>
    <div className="controlTopMetrics">
      <span>当前账号 / 角色 <b>{user?.username || currentTeamAccount.username} · {currentTeamAccount.role}</b></span>
      <span>当前竞拍 ID <b>{lot?.id || '无正在拍'}</b></span>
      <span>WebSocket <StatusBadge label={wsState} tone={wsState === '已连接' ? 'success' : wsState === '重连中' ? 'warning' : 'danger'} /></span>
      <span>Server offset <b>{offsetText}</b></span>
      <span>心跳 <b>{lastHeartbeat}</b></span>
    </div>
    <div className="controlTopActions"><button type="button">观众端预览</button><button type="button" className="controlPrimary" onClick={() => void onSync()}><RefreshCw size={15} /> 立即同步</button></div>
  </header>;
}

function HostBriefingRail({ lot, snapshot, wsState, onReveal }: { lot: Lot; snapshot: RoomSnapshot | null; wsState: string; onReveal: (cardId?: string) => void }) {
  return <aside className="controlLeftRail hostBriefingRail"><header className="briefingRailTitle"><span>主播讲解区</span><b>说什么、何时推、推给谁</b></header><RoomLivePreview lot={lot} snapshot={snapshot} wsState={wsState} /><CurrentLotInfoCard lot={lot} /><TeleprompterCard lot={lot} /><TrustCardPanel lot={lot} onReveal={onReveal} /></aside>;
}

function TeleprompterCard({ lot }: { lot: Lot }) {
  const nextPrice = { ...lot.currentPrice, amount: Number(lot.currentPrice?.amount || 0) + Number(lot.rule.minIncrement?.amount || 0) };
  return <section className="controlCard teleprompterCard"><h3>主播提词器</h3><p>“现在这件 {lot.title} 已来到 <b>{moneyText(lot.currentPrice)}</b>，下一口只要 <b>{moneyText(nextPrice)}</b>。注意看证书卡和瑕疵说明，喜欢的朋友别等最后一秒。”</p><div><span>强调卖点</span><span>封顶成交</span><span>最后出价自动延时</span></div></section>;
}

function RoomLivePreview({ lot, snapshot, wsState }: { lot: Lot; snapshot: RoomSnapshot | null; wsState: string }) {
  const live = lot.status === 'LOT_STATUS_LIVE';
  return <section className="roomLivePreview"><div className={`liveFrame ${live ? '' : 'prepared'}`}><img src={lot.imageUrl || '/vite.svg'} alt="" /><span>{live ? 'LIVE' : 'READY'}</span><b>{live ? moneyText(lot.currentPrice) : '等待开拍'}</b></div><div><span>在线 {Math.max(currentHostRoom.online, snapshot?.ranking?.length || 0)}</span><span>同步 {wsState}</span><span>房间隔离 ON</span></div></section>;
}

function CurrentLotInfoCard({ lot }: { lot: Lot }) {
  return <section className="controlCard"><h3>当前竞拍拍品</h3><img className="controlLotImg" src={lot.imageUrl || '/vite.svg'} alt="" /><b>{lot.title}</b><p>{lot.description || '拍品描述待接入'}</p><div className="controlTags"><span>当前直播间</span><span>{uiStatusOfLot(lot)}</span><span>v{lot.version || 1}</span></div></section>;
}

function RuleSnapshotCard({ lot }: { lot: Lot }) {
  const rule = lot.rule;
  return <section className="controlCard"><h3>规则快照</h3><div className="controlRuleGrid"><span>从多少钱开始拍<b>{moneyText(rule.startPrice)}</b></span><span>每次至少加多少钱<b>{moneyText(rule.minIncrement)}</b></span><span>封顶价<b>{capPriceText(lot)}</b></span><span>竞拍时长<b>{formatDuration(rule.durationSeconds)}</b></span><span>最后出价自动延时<b>{rule.antiSnipeWindowSeconds}s / +{rule.antiSnipeExtendSeconds}s</b></span><span>最大延时<b>{rule.maxExtendCount}</b></span><span>规则版本<b>v{lot.version || 1}</b></span></div></section>;
}

function TrustCardPanel({ lot, onReveal }: { lot: Lot; onReveal: (cardId?: string) => void }) {
  const cards = lot.trustCards || [];
  return <section className="controlCard"><header><h3>讲解卡 / 信任信息</h3><button type="button" onClick={() => void onReveal()}>展示下一张</button></header><div className="trustCardList">{cards.length ? cards.map((card) => <div key={card.id} className={card.revealed ? 'revealed' : ''}><b>{card.title}</b><span>{card.type.replace('TRUST_CARD_TYPE_', '')}</span><p>{card.content}</p><small>{card.revealed ? '已展示给观众' : '未展示'}</small><button type="button" disabled={card.revealed} onClick={() => void onReveal(card.id)}>{card.revealed ? '已展示' : '展示给观众'}</button></div>) : <p>讲解卡待接入。</p>}</div></section>;
}

function PlaybookStageBar({ lot }: { lot: Lot }) {
  const current = playbookStageLabel(lot.playbookStage);
  return <section className="playbookStageBar">{playbookStages.map(([, label]) => <div key={label} className={label === current ? 'active' : ''}><i />{label}</div>)}</section>;
}

function ServerSyncedCountdown({ lot, serverTime }: { lot: Lot; serverTime?: number | string }) {
  const [leftMs, setLeftMs] = useState(() => getLotLeftMs(lot, serverTime));
  useEffect(() => {
    let frame = 0;
    const tick = () => {
      setLeftMs(getLotLeftMs(lot, serverTime));
      frame = window.requestAnimationFrame(tick);
    };
    tick();
    return () => window.cancelAnimationFrame(frame);
  }, [lot.endsAtUnixMs, serverTime]);
  const text = formatAuctionLeftMs(leftMs, 'control');
  return <span className={`serverCountdown ${leftMs > 0 && leftMs < 10000 ? 'urgent danger' : ''}`}><Clock3 size={22} />{text}{serverTime ? null : <small>本地 fallback</small>}</span>;
}

function PriceCommandBoard({ lot, snapshot }: { lot: Lot; snapshot: RoomSnapshot | null }) {
  const nextPrice = { ...lot.currentPrice, amount: Number(lot.currentPrice?.amount || 0) + Number(lot.rule.minIncrement?.amount || 0) };
  const topRank = snapshot?.ranking?.[0];
  return <section className="priceCommandBoard auctionCommandCenter"><div className="priceCommandEyebrow"><span>Command Area</span><StatusBadge label={uiStatusOfLot(lot)} /></div><div className="currentPriceFocus"><p>当前最高价</p><strong className="livePriceBig">{moneyText(lot.currentPrice)}</strong><small>领先用户：{lot.leadingNickname || topRank?.nickname || topRank?.userId || '暂无'}</small></div><ServerSyncedCountdown lot={lot} serverTime={snapshot?.serverTimeUnixMs} /><div className="priceCommandMetrics"><span>下一口价<b>{moneyText(nextPrice)}</b></span><span>距离封顶<b>{moneyDeltaText(lot.rule.capPrice, lot.currentPrice)}</b></span><span>参与人数<b>{snapshot?.ranking?.length || 0}</b></span><span>出价次数<b>{snapshot?.recentBids?.length || 0}</b></span></div></section>;
}

function EmotionFeedbackLayer({ items }: { items: string[] }) {
  return <div className="emotionFeedbackLayer">{items.map((item) => <span key={item}>{item}</span>)}</div>;
}

function ControlActionDeck({ lot, status, onReveal, onDuel, onCancel, onSettle, onLog }: { lot: Lot; status: AuctionUiStatus; onReveal: () => void; onDuel: () => void; onCancel: () => void; onSettle: () => void; onLog: (entry: Omit<ControlLog, 'id' | 'time'>) => void }) {
  const canControl = canRole(currentTeamAccount.role, 'control');
  const canHighRisk = canRole(currentTeamAccount.role, 'cancel') && canRole(currentTeamAccount.role, 'settle');
  const settled = lot.status === 'LOT_STATUS_SETTLED';
  const cancelled = lot.status === 'LOT_STATUS_CANCELLED';
  return <section className="controlActionDeck"><header><h3>控场操作</h3><StatusBadge label={status} /></header><div className="controlActionsGrid"><button type="button" disabled={!canControl || settled || cancelled} onClick={() => onLog({ type: '推送提醒', detail: '观众端提醒接口待接入', level: 'info' })}>推送提醒</button><button type="button" disabled={!canControl || settled || cancelled} onClick={() => void onReveal()}>展示讲解卡</button><button type="button" disabled={!canControl || settled || cancelled} onClick={() => void onDuel()}>进入决胜</button><button type="button" disabled={!canControl} onClick={() => onLog({ type: '同步观众端状态', detail: '已请求重新广播 room snapshot', level: 'success' })}>同步观众端</button><button type="button" disabled={!canControl || settled || cancelled} onClick={() => onLog({ type: '延时 10 秒', detail: '延时接口待接入', level: 'warning' })}>延时 +10s</button><button type="button" disabled={!canControl || settled || cancelled} onClick={() => onLog({ type: '延时 30 秒', detail: '延时接口待接入', level: 'warning' })}>延时 +30s</button></div><div className="dangerActionStrip"><button type="button" disabled={!canHighRisk || settled || cancelled} onClick={onCancel}>异常取消</button><button type="button" className="settleButton" disabled={!canHighRisk || settled || cancelled} onClick={onSettle}>落锤成交</button><button type="button" disabled={!canHighRisk || settled || cancelled} onClick={() => onLog({ type: '强制结束', detail: '强制结束接口待接入，需二次确认', level: 'danger' })}>强制结束</button></div></section>;
}

function RealtimeBidFeedPanel({ bids }: { bids: Bid[] }) {
  return <section className="controlSideCard"><h3>实时出价流</h3><div className="controlBidFeed">{bids.length ? bids.slice(0, 12).map((bid) => <div key={bid.id}><span>{bid.nickname || bid.userId}</span><b>{moneyText(bid.amount)}</b><small>有效 · 延迟待接入 · {dateTimeText(bid.createdAtUnixMs)}</small></div>) : <StudioEmptyState compact icon={<ListChecks size={24} />} title="等待实时出价事件" description="开拍后这里会显示服务端接受的最新出价。" action={<a className="studioButton studioButton-soft studioButton-sm" href="/admin/bids">查看历史明细</a>} />}</div></section>;
}

function LiveRankingBoard({ ranking, leadingUserId }: { ranking: RoomSnapshot['ranking']; leadingUserId: string }) {
  const top = ranking[0];
  const topAmount = Number(top?.amount?.amount || 0);
  return <section className="controlSideCard rankingPanel"><h3>实时排行榜 TOP 5 / TOP 10</h3><div className="controlRanking">{ranking.length ? ranking.slice(0, 10).map((item) => <div key={item.userId}><b>#{item.rank}</b><span>{item.nickname || item.userId}</span><strong>{moneyText(item.amount)}</strong><small>{item.userId === leadingUserId ? '领先' : `差距 ¥${Math.max(0, topAmount - Number(item.amount.amount || 0)).toLocaleString('zh-CN')}`}</small></div>) : <StudioEmptyState compact icon={<Trophy size={24} />} title="排行榜等待 snapshot" description="房间快照恢复后展示 TOP 10 排名。" action={<button type="button" onClick={() => window.location.reload()}>刷新页面</button>} />}</div></section>;
}

function SystemHealthCard({ wsState, lastHeartbeat, snapshot, reconnectCount, lastEventType, lastEventSeq }: { wsState: LinkStatus; lastHeartbeat: string; snapshot: RoomSnapshot | null; reconnectCount: number; lastEventType: string; lastEventSeq: number }) {
  return <section className="controlSideCard syncStateCard"><h3>同步状态</h3><div className="systemHealthGrid"><span>连接状态<b>{wsState}</b></span><span>平均延迟<b>{currentHostRoom.latency}</b></span><span>最近心跳<b>{lastHeartbeat}</b></span><span>重连次数<b>{reconnectCount}</b></span><span>服务器偏移<b>{serverOffsetText(snapshot)}</b></span><span>最近事件<b>{lastEventType}</b></span><span>事件序号<b>#{lastEventSeq}</b></span><span>快照版本<b>{snapshot?.currentLot?.version || '待同步'}</b></span></div></section>;
}

function NextLotQueue({ lot, currentStatus }: { lot: Lot | null; currentStatus: AuctionUiStatus }) {
  return <section className="controlBottomCard"><h3>下一场待开拍</h3>{lot ? <div className="nextQueueItem"><b>{lot.title}</b><span>起拍 {moneyText(lot.rule.startPrice)} · 加价 {moneyText(lot.rule.minIncrement)} · 封顶价待接入 · {formatDuration(lot.rule.durationSeconds)}</span><button type="button" disabled={!['已成交', '已取消', '异常'].includes(currentStatus)}>开始下一场</button></div> : <StudioEmptyState compact icon={<Package size={24} />} title="暂无下一场待开拍" description="可以回到本场队列调整顺序，或添加新拍品。" action={<><a className="studioButton studioButton-secondary studioButton-sm" href="/admin/auctions">查看队列</a><a className="studioButton studioButton-primary studioButton-sm" href="/admin/auctions/create">添加拍品</a></>} />}</section>;
}

function ControlEventLog({ logs }: { logs: ControlLog[] }) {
  return <section className="controlBottomCard"><h3>控场事件日志</h3><div className="controlEventLog">{logs.length ? logs.map((log) => <div key={log.id} className={log.level}><span>{log.time}</span><b>{log.type}</b><small>{log.detail}</small></div>) : <StudioEmptyState compact icon={<MonitorDot size={24} />} title="等待控场操作和系统事件" description="同步、开拍、讲解卡展示、落锤等动作会记录在这里。" />}</div></section>;
}

function CancelLotDialog({ lot, onClose, onConfirm }: { lot: Lot; onClose: () => void; onConfirm: (reason: string) => Promise<void> }) {
  const reasons = ['拍品信息异常', '主播误操作', '出价异常', '系统延迟异常', '风控拦截', '其他'];
  const [reason, setReason] = useState(reasons[0]);
  const [detail, setDetail] = useState('');
  return <div className="controlDialog"><div onClick={onClose} /><section><h3>确认取消异常竞拍？</h3><p>取消后当前竞拍将立即终止，观众端会收到“竞拍异常取消”提醒，当前最高价不会生成成交订单，操作会写入审计日志。</p><div className="reasonGrid">{reasons.map((item) => <button key={item} type="button" className={reason === item ? 'active' : ''} onClick={() => setReason(item)}>{item}</button>)}</div><textarea value={detail} onChange={(e) => setDetail(e.target.value)} placeholder="补充说明，可选" /><footer><button type="button" onClick={onClose}>返回</button><button type="button" className="danger" onClick={() => void onConfirm(detail ? `${reason}：${detail}` : reason)}>确认取消</button></footer></section></div>;
}

function SettleLotDialog({ lot, onClose, onConfirm }: { lot: Lot; onClose: () => void; onConfirm: () => Promise<void> }) {
  return <div className="controlDialog"><div onClick={onClose} /><section><h3>确认落锤成交？</h3><p>确认后将按当前最高价生成成交结果；订单接口未接入时页面显示“成交订单待接入”。</p><div className="settleSummary"><span>拍品名<b>{lot.title}</b></span><span>成交价<b>{moneyText(lot.currentPrice)}</b></span><span>中标用户<b>{lot.leadingNickname || '暂无'}</b></span><span>规则版本<b>v{lot.version || 1}</b></span></div><footer><button type="button" onClick={onClose}>返回</button><button type="button" className="controlPrimary" onClick={() => void onConfirm()}>确认落锤</button></footer></section></div>;
}

function ControlPage() { return <LiveControlPage />; }

function ProductsPage() {
  const totalStock = products.reduce((sum, item) => sum + item.stock, 0);
  const readyCount = products.filter((item) => ['已上架', '竞拍中'].includes(item.status)).length;
  const settledCount = products.filter((item) => item.status === '已成交').length;
  return <section className="productLibraryPage">
    <section className="productLibraryHero">
      <div><p>当前直播间 / 拍品库</p><h2>拍品库</h2><span>沉淀可复拍、可排队、可复盘的拍品资产，商品助理在这里完成上架前准备。</span></div>
      <div className="productLibraryHeroActions"><a className="studioButton studioButton-secondary studioButton-lg" href="/admin/auctions">本场队列</a><a className="studioButton studioButton-primary studioButton-lg" href="/admin/auctions/create">添加拍品</a></div>
    </section>
    <section className="productLibraryStats"><div><span>库内拍品</span><b>{products.length}</b><small>当前直播间资产</small></div><div><span>可排队</span><b>{readyCount}</b><small>已上架 / 竞拍中</small></div><div><span>总库存</span><b>{totalStock}</b><small>可用于复拍</small></div><div><span>已成交</span><b>{settledCount}</b><small>可复制重拍</small></div></section>
    <section className="productLibraryFilters"><label><Search size={17} /><input placeholder="搜索拍品名 / 竞拍 ID / 分类" /></label><select defaultValue="全部分类" aria-label="分类"><option>全部分类</option><option>珠宝配饰</option><option>艺术收藏</option><option>箱包奢品</option><option>生活方式</option></select><select defaultValue="全部状态" aria-label="状态"><option>全部状态</option><option>已上架</option><option>竞拍中</option><option>已成交</option></select><select defaultValue="最近创建" aria-label="排序"><option>最近创建</option><option>估值最高</option><option>库存最多</option></select></section>
    <section className="productLibraryList" aria-label="拍品库列表">
      {products.map((product, index) => <article className="productLibraryRow" key={product.name}>
        <div className="productIdentity"><span className="productLibraryNo">#{String(index + 1).padStart(2, '0')}</span><div className="productLibraryThumb"><ShoppingBag size={28} /></div><div><h3>{product.name}</h3><div className="productLibraryTags"><StatusBadge label={product.status} /><span>{product.category}</span><span>{product.auction}</span></div></div></div>
        <div className="productLibraryMetrics"><span><b>估值</b>{product.estimate}</span><span><b>库存</b>{product.stock}</span><span><b>创建时间</b>{product.created}</span><span><b>队列绑定</b>{product.auction}</span></div>
        <div className="productLibraryActions"><a href="/admin/auctions/create">编辑资料</a><a className="primary" href="/admin/auctions">加入本场队列</a><button type="button">复制重拍</button><button type="button">下架</button></div>
      </article>)}
    </section>
  </section>;
}
function OrdersPage() {
  const [activeTab, setActiveTab] = useState('全部');
  const [detail, setDetail] = useState<OrderRow | null>(orders[0] || null);
  const tabs = ['全部', '待支付', '待发货', '异常订单'];
  const visibleOrders = orders.filter((order) => activeTab === '全部' || (activeTab === '异常订单' ? order.issue || order.pay === '异常' || order.fulfill === '待处理' : order.pay === activeTab || order.fulfill === activeTab));
  const metrics = [
    { icon: <CircleDollarSign />, label: '今日成交', value: orders.length, hint: '直播后自动生成成交单', tone: 'success' as Tone },
    { icon: <Clock3 />, label: '待支付', value: orders.filter((item) => item.pay === '待支付').length, hint: '建议 15 分钟内催付', tone: 'warning' as Tone },
    { icon: <Package />, label: '待发货', value: orders.filter((item) => item.fulfill === '待发货').length, hint: '支付后进入履约队列', tone: 'info' as Tone },
    { icon: <ShieldAlert />, label: '异常订单', value: orders.filter((item) => item.issue || item.pay === '异常').length, hint: '支付超时 / 锁冲突待复核', tone: 'danger' as Tone },
  ];
  return <section className="postLivePage orderReviewPage">
    <StudioCard padding="lg" className="postLiveHeader"><StudioSectionHeader eyebrow="Post-live settlement" title="成交处理" description="直播结束后核对成交生成、支付进度、履约状态和异常订单；业务客服与技术复核在同一张表里完成闭环。" actions={<><StudioButton variant="secondary" icon={<RefreshCw size={15} />}>同步成交</StudioButton><StudioButton variant="primary" icon={<ClipboardList size={15} />}>导出复盘</StudioButton></>} /></StudioCard>
    <section className="postLiveMetricGrid">{metrics.map((item) => <StatCard key={item.label} icon={item.icon} label={item.label} value={item.value} hint={item.hint} tone={item.tone} />)}</section>
    <section className="postLiveWorkbench" aria-label="直播后处理工作台"><article><b>1. 核对成交</b><span>确认最高价、规则快照和中标用户一致。</span></article><article><b>2. 跟进支付</b><span>待支付订单优先催付，超时进入异常。</span></article><article><b>3. 履约发货</b><span>已支付订单进入客服发货与售后闭环。</span></article><article className="danger"><b>异常优先</b><span>支付超时、锁冲突、重复成交先处理。</span></article></section>
    <div className="postLiveTabs" role="tablist">{tabs.map((tab) => <button key={tab} type="button" className={activeTab === tab ? 'active' : ''} onClick={() => setActiveTab(tab)}>{tab}<b>{tab === '全部' ? orders.length : orders.filter((order) => tab === '异常订单' ? order.issue || order.pay === '异常' || order.fulfill === '待处理' : order.pay === tab || order.fulfill === tab).length}</b></button>)}</div>
    <div className="postLiveGrid"><StudioTable className="orderReviewTable" rows={visibleOrders} rowKey={(r) => r.id} header={`共 ${visibleOrders.length} 条成交 · 第 1 / 1 页`} filters={<div className="postLiveFilters"><span>支付状态</span><span>履约状态</span><span>成交时间</span><span>异常优先</span></div>} empty={<StudioEmptyState icon={<ReceiptText size={34} />} title="当前筛选下暂无成交" description="换一个状态筛选，或同步最新成交记录。" action={<><StudioButton type="button" variant="secondary" icon={<RefreshCw size={15} />}>同步成交</StudioButton><StudioButton type="button" variant="primary" onClick={() => setActiveTab('全部')}>查看全部成交</StudioButton></>} compact />} columns={[{ label: '拍品 / 成交号', render: (r) => <div className="orderProductCell"><img src={r.image} alt={r.product} /><div><b>{r.product}</b><span>{r.id}</span><small>{r.auction}</small></div></div> }, { label: '成交价', render: (r) => <strong className="moneyText">{r.price}</strong> }, { label: '中标用户', render: (r) => <b>{r.buyer}</b> }, { label: '成交时间', render: (r) => r.time }, { label: '支付', render: (r) => <StatusBadge label={r.pay} /> }, { label: '履约', render: (r) => <StatusBadge label={r.fulfill} /> }, { label: '操作', render: (r) => <div className="laRowActions"><button type="button" onClick={() => setDetail(r)}>查看详情</button><button type="button">催付</button><button type="button">发货</button></div> }]} />
    {detail ? <OrderDetailDrawer order={detail} onClose={() => setDetail(null)} /> : null}</div>
  </section>;
}

function OrderDetailDrawer({ order, onClose }: { order: OrderRow; onClose: () => void }) {
  return <aside className="orderDetailDrawer"><StudioCard title="成交详情" subtitle={order.id} actions={<StudioButton size="sm" variant="ghost" onClick={onClose}>关闭</StudioButton>}><div className="drawerOrderHero"><img src={order.image} alt={order.product} /><div><h3>{order.product}</h3><strong>{order.price}</strong><span>{order.buyer} · {order.time}</span></div></div><div className="drawerInfoGrid"><span>支付状态<b>{order.pay}</b></span><span>履约状态<b>{order.fulfill}</b></span><span>关联竞拍<b>{order.auction}</b></span><span>规则快照<b>v1 · 固定加价</b></span></div>{order.issue ? <StudioErrorState compact icon={<AlertTriangle size={22} />} title="异常需要复核" description={order.issue} /> : <StudioEmptyState compact tone="success" icon={<CheckCircle2 size={22} />} title="暂无异常记录" description="支付、履约与成交规则快照一致。" />}<div className="drawerActions"><StudioButton variant="secondary">查看出价历史</StudioButton><StudioButton variant="primary">处理履约</StudioButton></div></StudioCard></aside>;
}

function BidsPage() {
  const valid = bidRows.filter((bid) => bid.result === '有效').length;
  const rejected = bidRows.length - valid;
  const duplicate = bidRows.filter((bid) => bid.key.includes('lock') || bid.result.includes('锁')).length;
  const metrics = [
    { icon: <ListChecks />, label: '今日出价次数', value: bidRows.length, hint: '按服务端接收时间排序', tone: 'info' as Tone },
    { icon: <ShieldCheck />, label: '有效', value: valid, hint: '通过价格与时效校验', tone: 'success' as Tone },
    { icon: <ShieldAlert />, label: '拒绝', value: rejected, hint: '低于当前价 / 锁冲突', tone: 'danger' as Tone },
    { icon: <Gauge />, label: '平均延迟', value: '84ms', hint: '客户端提交到服务端接收', tone: 'warning' as Tone },
    { icon: <DatabaseZap />, label: '重复提交', value: duplicate, hint: '幂等 Key 命中', tone: 'purple' as Tone },
  ];
  return <section className="postLivePage bidAuditPage">
    <StudioCard padding="lg" className="postLiveHeader"><StudioSectionHeader eyebrow="Technical bid audit" title="出价明细" description="面向直播后技术复核：核对每一次出价的金额、领先状态、服务端校验结果、延迟和幂等 Key，快速定位拒绝或重复提交。" actions={<><StudioButton variant="secondary" icon={<Search size={15} />}>保存筛选</StudioButton><StudioButton variant="primary" icon={<ClipboardList size={15} />}>导出明细</StudioButton></>} /></StudioCard>
    <section className="postLiveMetricGrid bidMetricGrid">{metrics.map((item) => <StatCard key={item.label} icon={item.icon} label={item.label} value={item.value} hint={item.hint} tone={item.tone} />)}</section>
    <section className="techWorkbench" aria-label="技术排查台"><StudioCard title="排查路径" subtitle="Investigation"><ol><li><b>校验拒绝</b><span>先看服务端结果与当前价，判断是否低价或过期。</span></li><li><b>延迟异常</b><span>超过 120ms 的出价优先核对客户端网络与 WebSocket 重连。</span></li><li><b>幂等冲突</b><span>同一 Key 重复提交时核对是否重复点击或锁冲突。</span></li></ol></StudioCard><StudioCard title="高风险线索" subtitle="Signals"><div className="riskSignalList"><span>锁冲突 <b>{duplicate}</b></span><span>拒绝出价 <b>{rejected}</b></span><span>延迟峰值 <b>138ms</b></span><span>最近异常 <b>idem_lock</b></span></div></StudioCard></section>
    <StudioTable className="bidAuditTable" rows={bidRows} rowKey={(r) => r.key} rowClassName={() => 'bidAuditRow'} header={`共 ${bidRows.length} 条出价 · 延迟 40-150ms`} filters={<div className="bidFilterBar"><label>拍品<input defaultValue="全部拍品" /></label><label>用户<input placeholder="用户 ID" /></label><label>校验结果<select defaultValue="全部"><option>全部</option><option>有效</option><option>低于当前价</option><option>锁冲突</option></select></label><label>时间<input defaultValue="今天" /></label><label>延迟范围<input defaultValue="0-150ms" /></label></div>} empty={<StudioEmptyState icon={<ListChecks size={34} />} title="暂无出价明细" description="当前筛选条件下没有服务端出价记录。" action={<><StudioButton type="button" variant="secondary" icon={<RefreshCw size={15} />}>重试拉取</StudioButton><a className="studioButton studioButton-primary studioButton-md" href="/admin/auctions">回到本场队列</a></>} compact />} columns={[{ label: '拍品 / 用户', render: (r) => <div className="bidIdentityCell"><b>{r.auctionId}</b><span>{r.user}</span><small>{r.device}</small></div> }, { label: '出价金额', render: (r) => <strong className="moneyText">{r.amount}</strong> }, { label: '领先', render: (r) => <StatusBadge label={r.leading ? '领先' : '未领先'} tone={r.leading ? 'success' : 'neutral'} /> }, { label: '校验结果', render: (r) => <StatusBadge label={r.result} /> }, { label: '延迟', render: (r) => <span className={Number.parseInt(r.latency, 10) > 120 ? 'latencyDanger' : 'latencyText'}>{r.latency}</span> }, { label: '幂等 Key', render: (r) => <code>{r.key}</code> }, { label: '出价时间', render: (r) => r.time }]} />
  </section>;
}
function RealtimePage() {
  const [snapshot, setSnapshot] = useState<RoomSnapshot | null>(null);
  const [wsState, setWsState] = useState<LinkStatus>('连接中');
  const [lastHeartbeat, setLastHeartbeat] = useState('未收到');
  const [reconnectCount, setReconnectCount] = useState(0);
  const [lastEventType, setLastEventType] = useState('暂无');
  const [lastEventSeq, setLastEventSeq] = useState(0);
  const [events, setEvents] = useState<LinkDiagnosticEvent[]>([]);
  const [error, setError] = useState('');
  const [logOpen, setLogOpen] = useState(false);

  const syncSnapshot = async () => {
    setError('');
    try {
      const next = await getRoomSnapshot(currentHostRoom.id);
      setSnapshot(next);
      setLastHeartbeat(controlNow());
      setLastEventType('ROOM_SNAPSHOT');
      setLastEventSeq((seq) => { const nextSeq = seq + 1; setEvents((list) => pushLinkEvent(list, makeLinkEvent(nextSeq, 'ROOM_SNAPSHOT', '手动重新同步房间快照'))); return nextSeq; });
    } catch (e) {
      setError(resultMessage(e));
    }
  };

  useEffect(() => { void syncSnapshot(); }, []);

  useEffect(() => {
    let socket: WebSocket | null = null;
    try {
      socket = new WebSocket(`${WS_BASE}/ws/rooms/${encodeURIComponent(currentHostRoom.id)}`);
      socket.onopen = () => { setWsState('已连接'); setLastHeartbeat(controlNow()); void syncSnapshot(); };
      socket.onclose = () => { setWsState('重连中'); setReconnectCount((count) => count + 1); };
      socket.onerror = () => { setWsState('已断开'); setReconnectCount((count) => count + 1); };
      socket.onmessage = (message) => {
        const event = normalizeAuctionEvent(JSON.parse(message.data)) as AuctionEvent;
        if (event.roomId && event.roomId !== currentHostRoom.id) return;
        if (!handledControlEventTypes.has(event.type)) return;
        setLastHeartbeat(controlNow());
        setLastEventType(event.type);
        setLastEventSeq((seq) => { const nextSeq = seq + 1; setEvents((list) => pushLinkEvent(list, makeLinkEvent(nextSeq, event.type, event.reason || event.lot?.title || event.bid?.nickname || '当前直播间事件', event.lotId))); return nextSeq; });
        if (event.snapshot) setSnapshot(event.snapshot);
      };
    } catch { setWsState('已断开'); }
    return () => socket?.close();
  }, []);

  const heartbeatTimeout = lastHeartbeat === '未收到';
  const offset = snapshot?.serverTimeUnixMs ? Math.abs(getServerOffsetMs(snapshot.serverTimeUnixMs)) : 0;
  const diagnostics = [
    ['心跳超时', heartbeatTimeout ? '待同步' : '正常'],
    ['排名延迟', snapshot?.ranking?.length ? '正常' : '等待排名事件'],
    ['倒计时偏移', offset > 1000 ? `${offset}ms，需关注` : serverOffsetText(snapshot)],
    ['重连频繁', reconnectCount >= 3 ? `${reconnectCount} 次，需关注` : '正常'],
    ['快照一致性', snapshot?.currentLot ? '已恢复 currentLot' : '等待 currentLot'],
  ];

  return <section className="realtimeDiagPage">
    <section className="realtimeDiagHero"><div><p>当前固定直播间</p><h2>实时链路诊断</h2><span>只诊断当前主播空间唯一直播间的 WebSocket、快照恢复、服务端时间偏移与事件流。</span></div><button type="button" onClick={() => void syncSnapshot()}>重新同步房间快照</button></section>
    {error ? <div className="auctionMgmtNotice danger"><AlertTriangle size={16} />{error}</div> : null}
    <RealtimeSyncCapsule wsState={wsState} snapshot={snapshot} lastHeartbeat={lastHeartbeat} reconnectCount={reconnectCount} lastEventType={lastEventType} lastEventSeq={lastEventSeq} onSync={syncSnapshot} onOpenLog={() => setLogOpen(true)} />
    <section className="realtimeDiagGrid"><CurrentRoomOverview wsState={wsState} lastHeartbeat={lastHeartbeat} snapshot={snapshot} /><section className="controlSideCard"><h3>客户端可计算指标</h3><div className="systemHealthGrid"><span>快照版本<b>{snapshot?.currentLot?.version || '待同步'}</b></span><span>排行榜版本<b>事件 #{lastEventSeq}</b></span><span>服务器偏移<b>{serverOffsetText(snapshot)}</b></span><span>最近事件<b>{lastEventType}</b></span></div></section></section>
    <section className="realtimeDiagGrid"><section className="controlBottomCard"><h3>异常诊断</h3><div className="linkDiagnosisList">{diagnostics.map(([label, value]) => <div key={label}><span>{label}</span><b>{value}</b></div>)}</div></section><section className="controlBottomCard"><h3>最近事件流</h3><div className="linkEventList inline">{events.length ? events.map((event) => <div key={`${event.seq}-${event.time}`}><b>#{event.seq}</b><span>{event.type}</span><small>{event.time} · {event.lotId || currentHostRoom.id}</small><p>{event.detail}</p></div>) : <p>暂无事件。等待 WebSocket 事件或手动同步房间快照。</p>}</div></section></section>
    {logOpen ? <LinkEventLogDrawer events={events} onClose={() => setLogOpen(false)} /> : null}
  </section>;
}

function AlertsPage() { return <Panel title="异常告警"><DataTable rows={alerts} rowKey={(r) => `${r.type}-${r.time}`} columns={[{ label: '告警类型', render: (r) => <b>{r.type}</b> }, { label: '告警等级', render: (r) => <StatusBadge label={r.level} /> }, { label: '关联对象', render: (r) => r.target }, { label: '时间', render: (r) => r.time }, { label: '详情', render: (r) => r.detail }, { label: '操作', render: () => <div className="laRowActions"><a>处理</a><a>查看日志</a></div> }]} /></Panel>; }

function SettingSection({ icon, title, desc, children }: { icon: ReactNode; title: string; desc: string; children: ReactNode }) {
  return <article className="laSettingSection"><div className="laSettingHead"><span>{icon}</span><div><h3>{title}</h3><p>{desc}</p></div></div><div className="laSettingBody">{children}</div></article>;
}

function SettingToggle({ label, desc, checked = true }: { label: string; desc: string; checked?: boolean }) {
  return <label className="laSettingToggle"><span><b>{label}</b><small>{desc}</small></span><input type="checkbox" defaultChecked={checked} /><i /></label>;
}

function TeamAccountsPage() {
  return <>
    <section className="laSettingsHero laMerchantsHero"><div><p>Host Workspace Team</p><h2>团队协作</h2><span>一个主播主账号对应一个后台空间。主账号可以创建多个团队子账号，分别负责场控、商品助理、订单客服、数据复盘等协作任务。</span></div><div className="laWelcomeActions"><a>新增子账号</a><a>分配权限</a><a>批量导入</a></div></section>
    <section className="laStatsGrid"><StatCard icon={<ShoppingBag />} label="主播空间" value="1" hint="主账号已认证" tone="info" /><StatCard icon={<Radio />} label="团队子账号" value="18" hint="14 个已绑定直播间" tone="purple" /><StatCard icon={<Package />} label="绑定拍品" value="1,286" hint="竞拍中 42 件" tone="success" /><StatCard icon={<ShieldAlert />} label="风险主体" value="2" hint="1 个高风险限制中" tone="danger" /></section>
    <section className="laGrid laGrid-2-1"><Panel title="主体列表" action={<a>导出名单</a>}><DataTable rows={merchantRows} rowKey={(r) => r.id} columns={[{ label: '主体 ID', render: (r) => <b>{r.id}</b> }, { label: '名称', render: (r) => r.name }, { label: '类型', render: (r) => <StatusBadge label={r.type} /> }, { label: '账号', render: (r) => r.owner }, { label: '角色', render: (r) => r.role }, { label: '绑定直播间', render: (r) => r.room }, { label: '拍品数', render: (r) => r.products }, { label: '竞拍场次', render: (r) => r.auctions }, { label: 'GMV', render: (r) => <strong>{r.gmv}</strong> }, { label: '状态', render: (r) => <StatusBadge label={r.status} /> }, { label: '风控', render: (r) => <StatusBadge label={r.risk} /> }, { label: '操作', render: () => <div className="laRowActions"><a>编辑资料</a><a>绑定角色</a><a>查看竞拍</a><a>风控记录</a></div> }]} /></Panel><Panel title="团队准入与绑定"><div className="laMerchantFlow"><div><CheckCircle2 size={20} /><b>资质认证</b><span>主播实名、经营资质、拍品类目授权和直播间归属。</span></div><div><Radio size={20} /><b>固定直播间绑定</b><span>主账号固定绑定一个直播间，子账号权限都围绕该直播间协作。</span></div><div><ShieldCheck size={20} /><b>角色授权</b><span>子账号按岗位分配拍品、成交、控场、数据和风控权限。</span></div><div><AlertTriangle size={20} /><b>风控限制</b><span>异常出价、成交失败、投诉或锁冲突可限制发布和控场。</span></div></div></Panel></section>
    <section className="laGrid laGrid-1-1"><Panel title="主播空间画像"><div className="laMerchantProfile"><div><span>小鹿珠宝直播团队</span><strong>¥326,800</strong><small>近 30 日 GMV · 18 场竞拍 · 成交率 76.4%</small></div><div className="laRulePreview"><span>珠宝配饰</span><span>已认证</span><span>WebSocket 稳定</span><span>无高危告警</span></div></div></Panel><Panel title="最近团队账号审计"><DataTable rows={merchantAuditRows} rowKey={(r) => `${r.subject}-${r.time}`} columns={[{ label: '时间', render: (r) => r.time }, { label: '账号/空间', render: (r) => r.subject }, { label: '操作', render: (r) => r.action }, { label: '操作人', render: (r) => r.operator }, { label: '结果', render: (r) => <StatusBadge label={r.result} /> }]} /></Panel></section>
  </>;
}

function RolesPage() {
  return <>
    <section className="laSettingsHero laRolesHero"><div><p>RBAC Permission</p><h2>岗位权限</h2><span>围绕直播竞拍核心链路配置权限：拍品上架、规则配置、主播控场、成交结算、实时链路诊断与异常风控。高风险操作需要主播主账号授权和审计记录。</span></div><button className="laPrimaryBtn">新建角色</button></section>
    <section className="laStatsGrid"><StatCard icon={<UserCog />} label="角色数量" value="5" hint="主账号 / 场控 / 商品助理 / 订单客服 / 数据复盘" /><StatCard icon={<Users />} label="绑定用户" value="59" hint="当前工作台账号" tone="info" /><StatCard icon={<ShieldCheck />} label="高风险权限" value="8" hint="强制结束、异常取消等" tone="warning" /><StatCard icon={<ShieldAlert />} label="待审计操作" value="3" hint="近 24 小时" tone="danger" /></section>
    <section className="laGrid laGrid-2-1"><Panel title="角色列表" action={<a>同步账号</a>}><DataTable rows={roleRows} rowKey={(r) => r.role} columns={[{ label: '角色', render: (r) => <b>{r.role}</b> }, { label: '用户数', render: (r) => r.users }, { label: '数据范围', render: (r) => r.scope }, { label: '说明', render: (r) => r.desc }, { label: '风险等级', render: (r) => <StatusBadge label={r.risk} /> }, { label: '操作', render: () => <div className="laRowActions"><a>编辑权限</a><a>复制角色</a><a>审计日志</a></div> }]} /></Panel><Panel title="权限策略摘要"><div className="laRoleSummary"><div><Crown size={22} /><b>最小权限原则</b><span>子账号只拥有岗位所需权限，控场账号只进入授权直播间。</span></div><div><ShieldAlert size={22} /><b>高风险二次确认</b><span>异常取消、强制结束、踢出连接需要记录原因。</span></div><div><DatabaseZap size={22} /><b>审计留痕</b><span>规则修改、控场动作、风控处置进入审计流。</span></div></div></Panel></section>
    <Panel title="权限矩阵"><div className="laPermissionMatrix">{permissionGroups.map((group) => <section key={group.group}><h3>{group.group}</h3>{group.items.map((item, index) => <label key={item}><span>{item}</span><input type="checkbox" defaultChecked={index < 4} /><i /></label>)}</section>)}</div></Panel>
    <Panel title="最近权限审计"><DataTable rows={[{ user: 'host_lulu', action: '修改竞拍规则权限', role: '主播主账号', time: '12:46:20', result: '已通过' }, { user: 'team_ada', action: '申请异常取消权限', role: '场控', time: '12:32:11', result: '待复核' }, { user: 'team_lux', action: '绑定拍品库权限', role: '商品助理', time: '11:58:09', result: '已通过' }]} rowKey={(r) => `${r.user}-${r.time}`} columns={[{ label: '账号', render: (r) => r.user }, { label: '操作', render: (r) => r.action }, { label: '角色', render: (r) => r.role }, { label: '时间', render: (r) => r.time }, { label: '结果', render: (r) => <StatusBadge label={r.result} /> }]} /></Panel>
  </>;
}

function SettingsPage() {
  return <>
    <section className="laSettingsHero"><div><p>System Settings</p><h2>工作台设置</h2><span>统一管理直播竞拍默认规则、WebSocket 实时链路、风控阈值、成交生成和通知策略。当前为前端配置原型，后续对接 Kratos 配置接口。</span></div><button className="laPrimaryBtn">保存设置</button></section>
    <section className="laSettingsGrid">
      <SettingSection icon={<Gavel size={20} />} title="竞拍默认规则" desc="新建竞拍时默认带出的业务规则，服务 0 元起拍、固定加价、封顶成交和最后出价自动延时。">
        <div className="laFormGrid"><label>默认从多少钱开始拍<input defaultValue="0" /></label><label>默认每次至少加多少钱<input defaultValue="100" /></label><label>默认竞拍时长<input defaultValue="300 秒" /></label><label>默认到这个价自动成交<input defaultValue="12000" /></label><label>结束前触发延时<input defaultValue="15 秒" /></label><label>每次延长时间<input defaultValue="10-30 秒" /></label></div>
        <div className="laRulePreview"><span>0 元起拍</span><span>固定加价</span><span>封顶成交</span><span>最后出价自动延时</span><span>异常取消</span></div>
      </SettingSection>
      <SettingSection icon={<Wifi size={20} />} title="实时链路诊断" desc="控制房间广播、心跳保活、断线快照恢复和重连策略。">
        <SettingToggle label="启用心跳保活" desc="客户端每 10 秒上报心跳，服务端记录最后活跃时间。" />
        <SettingToggle label="断线自动快照恢复" desc="重连后主动下发价格、排名、倒计时和当前状态。" />
        <SettingToggle label="房间级广播节流" desc="高频出价时合并非关键 UI 消息，保证排名和价格优先。" />
        <div className="laFormGrid"><label>心跳间隔<input defaultValue="10 秒" /></label><label>重连最大次数<input defaultValue="5 次" /></label><label>广播 P95 告警<input defaultValue="100ms" /></label><label>快照缓存 TTL<input defaultValue="30 秒" /></label></div>
      </SettingSection>
      <SettingSection icon={<ShieldAlert size={20} />} title="高并发风控" desc="处理重复提交、每次至少加多少钱错误、竞拍结束后出价、Redis 锁冲突等风险。">
        <SettingToggle label="启用幂等 Key 校验" desc="同一用户同一请求只允许成功处理一次。" />
        <SettingToggle label="锁冲突自动告警" desc="Redis 锁冲突超过阈值时推送严重告警。" />
        <SettingToggle label="客户端延迟校验" desc="客户端延迟异常时优先以服务端竞拍状态为准。" />
        <div className="laFormGrid"><label>重复提交窗口<input defaultValue="3 秒" /></label><label>锁等待超时<input defaultValue="200ms" /></label><label>异常 IP 阈值<input defaultValue="20 次/分钟" /></label><label>倒计时偏移阈值<input defaultValue="500ms" /></label></div>
      </SettingSection>
      <SettingSection icon={<ReceiptText size={20} />} title="成交与支付" desc="竞拍结束后自动生成成交，支持模拟支付和异常成交追踪。">
        <SettingToggle label="成交后自动生成成交" desc="竞拍已成交时写入成交订单和规则快照。" />
        <SettingToggle label="开启模拟支付" desc="本地演示环境允许使用模拟支付流。" />
        <SettingToggle label="成交异常记录" desc="成交生成失败、支付超时和履约异常进入告警列表。" />
        <div className="laFormGrid"><label>支付超时时间<input defaultValue="15 分钟" /></label><label>成交关闭策略<input defaultValue="超时自动关闭" /></label></div>
      </SettingSection>
      <SettingSection icon={<Bell size={20} />} title="通知与提醒" desc="控制领先、被超越、延时、成交和异常取消等实时提醒。">
        <SettingToggle label="领先提醒" desc="用户成为第一名时推送 🎉 领先反馈。" />
        <SettingToggle label="被超越提醒" desc="排名被超过时推送 ⚡ 被超越反馈。" />
        <SettingToggle label="竞拍延时提醒" desc="触发最后出价自动延时时向房间广播 ⏱ 延时消息。" />
        <SettingToggle label="异常取消提醒" desc="主播异常取消时通知参与用户和运营。" />
      </SettingSection>
      <SettingSection icon={<Settings size={20} />} title="系统运行参数" desc="本地演示、服务注册发现和数据源状态。">
        <div className="laSettingInfo"><span>注册中心</span><b>Consul planned</b></div><div className="laSettingInfo"><span>数据存储</span><b>MySQL + Redis</b></div><div className="laSettingInfo"><span>当前环境</span><b>Local Dev</b></div><div className="laSettingInfo"><span>默认主题</span><b>浅蓝粉玻璃拟态</b></div>
      </SettingSection>
    </section>
  </>;
}

function PlaceholderPage({ title }: { title: string }) { return <Panel title={title}><StudioEmptyState className="laPlaceholder" icon={<Sparkles size={42} />} title={`${title} 已按 LiveAuction Console 信息架构预留`} description="后续接入真实 Kratos API / WebSocket / Redis 指标后补充图表和操作流。" /></Panel>; }

type ConsoleRoute = { match: (pathname: string) => boolean; Page: ComponentType };

const consoleRoutes: ConsoleRoute[] = [
  { match: (pathname) => pathname === '/admin/realtime' || pathname === '/host/realtime', Page: RealtimePage },
  { match: (pathname) => pathname.includes('/auctions/create'), Page: AuctionCreatePage },
  { match: (pathname) => pathname.includes('/control'), Page: ControlPage },
  { match: (pathname) => pathname.includes('/auctions'), Page: AuctionManagementPage },
  { match: (pathname) => pathname.includes('/orders'), Page: OrdersPage },
  { match: (pathname) => pathname.includes('/bids'), Page: BidsPage },
  { match: (pathname) => pathname.includes('/products'), Page: ProductsPage },
  { match: (pathname) => pathname.includes('/settings'), Page: SettingsPage },
  { match: (pathname) => pathname.includes('/merchants'), Page: TeamAccountsPage },
];

function routePage(pathname = location.pathname) {
  const RoutePage = consoleRoutes.find((route) => route.match(pathname))?.Page || DashboardPage;
  return <RoutePage />;
}

export function HostConsolePage() {
  return <AppShell>{routePage()}</AppShell>;
}
