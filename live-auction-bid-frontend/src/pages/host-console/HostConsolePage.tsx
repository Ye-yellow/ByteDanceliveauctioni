import { useEffect, useMemo, useState, type ReactNode } from 'react';
import { Bell, FileClock, Gavel, LayoutDashboard, ListChecks, PlayCircle, Radio, ReceiptText, Settings, ShieldAlert, Users, Wifi } from 'lucide-react';
import { AdminDashboardPage } from '../../features/auction-manage/AdminDashboardPage';
import { AuctionHistoryPage } from '../../features/auction-manage/AuctionHistoryPage';
import { AuctionManagementPage } from '../../features/auction-manage/AuctionManagementPage';
import { AuctionCreatePage } from '../../features/auction-create/AuctionCreatePage';
import { OrderManagementPage } from '../../features/order-manage/OrderManagementPage';
import { TeamAccountsPage } from '../../features/team-accounts/TeamAccountsPage';
import { BidAuditPage, LiveControlPage, RealtimeDiagnosticsPage } from '../../features/realtime-console/RealtimeConsolePages';
import { listAdminRooms } from '../../features/auction/api/auctionApi';
import { ADMIN_TEAM_ACCOUNT } from '../../shared/config/studio';
import type { Room } from '../../shared/api/types';
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
  if (pathname.includes('/merchants')) return '团队成员';
  if (pathname.includes('/settings')) return '工作台设置';
  if (pathname.includes('/alerts')) return '异常告警';
  return '今日工作台';
}

function AppShell({ children, currentRoom }: { children: ReactNode; currentRoom: Room }) {
  const roomSummary = useMemo(() => ({ name: currentRoom.name || currentRoom.id, latency: currentRoom.platform || 'douyin' }), [currentRoom]);
  return <HostConsoleShell navGroups={navGroups} currentHostRoom={roomSummary} currentTeamAccount={ADMIN_TEAM_ACCOUNT} titleForPath={pathTitle}>{children}</HostConsoleShell>;
}

type ConsoleRoute = { match: (pathname: string) => boolean; render: (room: Room) => ReactNode };

const consoleRoutes: ConsoleRoute[] = [
  { match: (pathname) => pathname === '/admin/realtime' || pathname === '/host/realtime', render: (room) => <RealtimeDiagnosticsPage roomId={room.id} /> },
  { match: (pathname) => pathname.includes('/auctions/create'), render: (room) => <AuctionCreatePage roomId={room.id} roomName={room.name} /> },
  { match: (pathname) => pathname.includes('/auctions/history'), render: (room) => <AuctionHistoryPage roomId={room.id} /> },
  { match: (pathname) => pathname.includes('/control'), render: (room) => <LiveControlPage roomId={room.id} /> },
  { match: (pathname) => pathname.includes('/auctions'), render: (room) => <AuctionManagementPage roomId={room.id} roomName={room.name} /> },
  { match: (pathname) => pathname.includes('/orders'), render: (room) => <OrderManagementPage roomId={room.id} /> },
  { match: (pathname) => pathname.includes('/bids'), render: (room) => <BidAuditPage roomId={room.id} /> },
  { match: (pathname) => pathname.includes('/merchants'), render: () => <TeamAccountsPage /> },
  { match: (pathname) => pathname.includes('/settings'), render: () => <SettingsPage /> },
  { match: (pathname) => pathname.includes('/alerts'), render: () => <AlertsPage /> },
];

function routePage(room: Room, pathname = location.pathname) {
  const route = consoleRoutes.find((item) => item.match(pathname));
  return route ? route.render(room) : <AdminDashboardPage roomId={room.id} roomName={room.name} />;
}

export function HostConsolePage() {
  const [room, setRoom] = useState<Room | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    let alive = true;
    setLoading(true);
    listAdminRooms()
      .then((nextRooms) => {
        if (!alive) return;
        setRoom(nextRooms[0] || null);
        setError('');
      })
      .catch((err) => {
        if (!alive) return;
        setError(err instanceof Error ? err.message : String(err));
      })
      .finally(() => {
        if (alive) setLoading(false);
      });
    return () => { alive = false; };
  }, []);

  if (loading) return <LoadingShell />;
  if (error || !room) return <RoomErrorPage message={error || '当前主账号还没有可用直播间'} />;
  return <AppShell currentRoom={room}>{routePage(room)}</AppShell>;
}

function LoadingShell() {
  return <section className="settingsPage laSettingsGrid"><StudioCard padding="lg"><StudioPageHeader eyebrow="Rooms" title="正在加载直播间" description="正在获取当前主账号的直播间配置。" /></StudioCard></section>;
}

function RoomErrorPage({ message }: { message: string }) {
  return <section className="settingsPage laSettingsGrid"><StudioCard padding="lg"><StudioEmptyState icon={<ShieldAlert size={34} />} title="直播间不可用" description={message} /></StudioCard></section>;
}

function SettingsPage() {
  return <section className="settingsPage laSettingsGrid">
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
  return <StudioCard title="异常告警" subtitle="Alerts" padding="lg" className="alertsPage">
    <StudioEmptyState icon={<ShieldAlert size={34} />} title="告警列表待后端接口" description="P2 不新增 mock 告警数据。竞拍、订单和实时异常已经在对应 feature 页内用真实接口错误展示。" />
  </StudioCard>;
}
