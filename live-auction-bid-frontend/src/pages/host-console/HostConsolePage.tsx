import { type ComponentType, type ReactNode } from 'react';
import { Bell, FileClock, Gavel, LayoutDashboard, ListChecks, Package, PlayCircle, Radio, ReceiptText, Settings, ShieldAlert, Users, Wifi } from 'lucide-react';
import { AdminDashboardPage } from '../../features/auction-manage/AdminDashboardPage';
import { AuctionHistoryPage } from '../../features/auction-manage/AuctionHistoryPage';
import { AuctionManagementPage } from '../../features/auction-manage/AuctionManagementPage';
import { ProductLibraryPage } from '../../features/auction-manage/ProductLibraryPage';
import { AuctionCreatePage } from '../../features/auction-create/AuctionCreatePage';
import { OrderManagementPage } from '../../features/order-manage/OrderManagementPage';
import { TeamAccountsPage } from '../../features/team-accounts/TeamAccountsPage';
import { BidAuditPage, LiveControlPage, RealtimeDiagnosticsPage } from '../../features/realtime-console/RealtimeConsolePages';
import { ADMIN_ROOM, ADMIN_TEAM_ACCOUNT } from '../../shared/config/studio';
import { HostConsoleShell, type StudioNavGroupConfig } from './components/HostConsoleShell';
import { StudioCard, StudioEmptyState, StudioMetricCard, StudioPageHeader } from './components/studio-ui';
import './styles/console-round06.css';

const navGroups: StudioNavGroupConfig[] = [
  { label: '今日直播', items: [
    { label: '今日工作台', href: '/admin', icon: <LayoutDashboard size={17} />, match: (pathname) => pathname === '/admin' || pathname === '/host' },
    { label: '直播间中控台', href: '/admin/auctions/current/control', icon: <Radio size={17} />, match: (pathname) => pathname.includes('/control') },
  ] },
  { label: '直播筹备', items: [
    { label: '添加拍品', href: '/admin/auctions/create', icon: <PlayCircle size={17} />, match: (pathname) => pathname.includes('/auctions/create') },
    { label: '本场拍品队列', href: '/admin/auctions', icon: <Gavel size={17} />, match: (pathname) => pathname.includes('/auctions') && !pathname.includes('/auctions/create') && !pathname.includes('/auctions/history') && !pathname.includes('/control') },
    { label: '拍品库', href: '/admin/products', icon: <Package size={17} />, match: (pathname) => pathname.includes('/products') },
  ] },
  { label: '直播后', items: [
    { label: '拍品历史', href: '/admin/auctions/history', icon: <FileClock size={17} />, match: (pathname) => pathname.includes('/auctions/history') },
    { label: '成交处理', href: '/admin/orders', icon: <ReceiptText size={17} />, match: (pathname) => pathname.includes('/orders') },
    { label: '出价明细', href: '/admin/bids', icon: <ListChecks size={17} />, match: (pathname) => pathname.includes('/bids') },
  ] },
  { label: '团队协作', items: [
    { label: '团队成员', href: '/admin/merchants', icon: <Users size={17} />, match: (pathname) => pathname.includes('/merchants') },
    { label: '直播健康', href: '/admin/realtime', icon: <Wifi size={17} />, match: (pathname) => pathname === '/admin/realtime' || pathname === '/host/realtime' },
  ] },
  { label: '系统', items: [
    { label: '工作台设置', href: '/admin/settings', icon: <Settings size={17} />, match: (pathname) => pathname.includes('/settings') },
    { label: '异常告警', href: '/admin/alerts', icon: <ShieldAlert size={17} />, match: (pathname) => pathname.includes('/alerts') },
  ] },
];

function pathTitle(pathname: string) {
  if (pathname === '/admin/realtime' || pathname === '/host/realtime') return '直播健康';
  if (pathname.includes('/auctions/create')) return '添加拍品';
  if (pathname.includes('/auctions/history')) return '拍品历史';
  if (pathname.includes('/control')) return '直播间中控台';
  if (pathname.includes('/auctions')) return '本场拍品队列';
  if (pathname.includes('/orders')) return '成交处理';
  if (pathname.includes('/bids')) return '出价明细';
  if (pathname.includes('/products')) return '拍品库';
  if (pathname.includes('/merchants')) return '团队成员';
  if (pathname.includes('/settings')) return '工作台设置';
  if (pathname.includes('/alerts')) return '异常告警';
  return '今日工作台';
}

function AppShell({ children }: { children: ReactNode }) {
  return <HostConsoleShell navGroups={navGroups} currentHostRoom={ADMIN_ROOM} currentTeamAccount={ADMIN_TEAM_ACCOUNT} titleForPath={pathTitle}>{children}</HostConsoleShell>;
}

type ConsoleRoute = { match: (pathname: string) => boolean; Page: ComponentType };

const consoleRoutes: ConsoleRoute[] = [
  { match: (pathname) => pathname === '/admin/realtime' || pathname === '/host/realtime', Page: RealtimeDiagnosticsPage },
  { match: (pathname) => pathname.includes('/auctions/create'), Page: AuctionCreatePage },
  { match: (pathname) => pathname.includes('/auctions/history'), Page: AuctionHistoryPage },
  { match: (pathname) => pathname.includes('/control'), Page: LiveControlPage },
  { match: (pathname) => pathname.includes('/auctions'), Page: AuctionManagementPage },
  { match: (pathname) => pathname.includes('/orders'), Page: () => <OrderManagementPage roomId={ADMIN_ROOM.id} /> },
  { match: (pathname) => pathname.includes('/bids'), Page: BidAuditPage },
  { match: (pathname) => pathname.includes('/products'), Page: ProductLibraryPage },
  { match: (pathname) => pathname.includes('/merchants'), Page: TeamAccountsPage },
  { match: (pathname) => pathname.includes('/settings'), Page: SettingsPage },
  { match: (pathname) => pathname.includes('/alerts'), Page: AlertsPage },
];

function routePage(pathname = location.pathname) {
  const RoutePage = consoleRoutes.find((route) => route.match(pathname))?.Page || AdminDashboardPage;
  return <RoutePage />;
}

export function HostConsolePage() {
  return <AppShell>{routePage()}</AppShell>;
}

function SettingsPage() {
  return <section className="laSettingsGrid">
    <StudioCard padding="lg" className="laSettingsHero">
      <StudioPageHeader eyebrow="System settings" title="工作台设置" description="P2 只保留后台设置的信息架构入口；涉及风控阈值、默认规则和通知配置的写接口未进入本轮。" />
    </StudioCard>
    <StudioMetricCard icon={<Gavel />} label="默认规则" value="待接口" trend="P3/P4 接配置服务" tone="info" />
    <StudioMetricCard icon={<Wifi />} label="实时链路" value="RoomSocket" trend="统一走 shared/realtime" tone="success" />
    <StudioMetricCard icon={<ReceiptText />} label="成交策略" value="HTTP 查询" trend="订单详情不依赖公开事件" tone="purple" />
    <StudioMetricCard icon={<Bell />} label="通知策略" value="待接口" trend="当前只显示本地提示" tone="warning" />
  </section>;
}

function AlertsPage() {
  return <StudioCard title="异常告警" subtitle="Alerts" padding="lg">
    <StudioEmptyState icon={<ShieldAlert size={34} />} title="告警列表待后端接口" description="P2 不新增 mock 告警数据。竞拍、订单和实时异常已经在对应 feature 页内用真实接口错误展示。" />
  </StudioCard>;
}
