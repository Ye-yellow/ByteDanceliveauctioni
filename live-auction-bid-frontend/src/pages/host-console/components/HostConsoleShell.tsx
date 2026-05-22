import { type ReactNode } from 'react';
import { Bell, Search, Wifi } from 'lucide-react';
import { currentAuth } from '../../../features/auth/api/authApi';

type HostRoomSummary = { name: string; latency: string };
type TeamAccountSummary = { username: string; role: string };

export type StudioNavItemConfig = {
  label: string;
  href: string;
  icon: ReactNode;
  match?: (pathname: string) => boolean;
};

export type StudioNavGroupConfig = { label: string; items: StudioNavItemConfig[] };

type HostConsoleShellProps = {
  children: ReactNode;
  navGroups: StudioNavGroupConfig[];
  currentHostRoom: HostRoomSummary;
  currentTeamAccount: TeamAccountSummary;
  titleForPath: (pathname: string) => string;
};

export function HostConsoleShell({ children, navGroups, currentHostRoom, currentTeamAccount, titleForPath }: HostConsoleShellProps) {
  const title = titleForPath(location.pathname);
  return <main className="laAdminShell studioShell">
    <a className="studioSkipLink" href="#studio-content">跳到主内容</a>
    <StudioSidebar navGroups={navGroups} />
    <section className="laMain studioMain">
      <StudioTopbar title={title} currentHostRoom={currentHostRoom} currentTeamAccount={currentTeamAccount} />
      <StudioContent title={title} currentHostRoom={currentHostRoom}>{children}</StudioContent>
    </section>
  </main>;
}

function StudioSidebar({ navGroups }: { navGroups: StudioNavGroupConfig[] }) {
  return <aside className="laSidebar studioSidebar">
    <a href="/home" className="laBrand studioBrand"><span>竞</span><div><strong>LiveAuction Studio</strong><small>直播间竞拍工作台</small></div></a>
    <nav className="laNav studioNav">{navGroups.map((group) => <StudioNavGroup key={group.label} group={group} />)}</nav>
  </aside>;
}

function StudioNavGroup({ group }: { group: StudioNavGroupConfig }) {
  return <section className="studioNavGroup"><p>{group.label}</p>{group.items.map((item) => <StudioNavItem key={item.href} item={item} />)}</section>;
}

function StudioNavItem({ item }: { item: StudioNavItemConfig }) {
  const active = item.match ? item.match(location.pathname) : location.pathname === item.href;
  return <a className={active ? 'active' : ''} href={item.href}>{item.icon}<span>{item.label}</span></a>;
}

function StudioTopbar({ title, currentHostRoom, currentTeamAccount }: { title: string; currentHostRoom: HostRoomSummary; currentTeamAccount: TeamAccountSummary }) {
  const user = currentAuth().user;
  const avatarText = user?.nickname?.slice(0, 1) || user?.username?.slice(0, 1) || '主';
  return <header className="laTopBar studioTopbar">
    <div className="studioTopbarTitle"><p>{currentHostRoom.name}</p><h1>{title}</h1><span>{currentTeamAccount.username} · {currentTeamAccount.role}</span></div>
    <label className="laSearch studioSearch"><Search size={16} /><input placeholder="搜索拍品 / 订单 / 出价记录" /></label>
    <div className="laTopActions studioTopActions"><WebSocketStatus /><button type="button" aria-label="通知"><Bell size={16} /></button><span className="laAvatar studioAvatar">{avatarText}</span></div>
  </header>;
}

function WebSocketStatus() {
  return <span className="laWsStatus"><Wifi size={15} /> 实时同步正常 <b>38ms</b></span>;
}

function StudioContent({ title, children }: { title: string; currentHostRoom: HostRoomSummary; children: ReactNode }) {
  return <div id="studio-content" className="laContent studioContent"><div className="studioPage"><StudioPageHeader title={title} /><main className="studioPageBody">{children}</main></div></div>;
}

function StudioPageHeader({ title }: { title: string }) {
  return <header className="studioPageHeader"><div><p>LiveAuction Studio</p><h2>{title}</h2></div></header>;
}
