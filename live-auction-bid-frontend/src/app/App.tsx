import { currentAuth } from '../features/auth/api/authApi';
import { HomePage } from '../pages/home/HomePage';
import { HostConsolePage } from '../pages/host-console/HostConsolePage';
import { LoginPage } from '../pages/login/LoginPage';
import { LiveRoomPage } from '../pages/live-room/LiveRoomPage';

function isAdminPath(pathname: string) {
  return pathname.startsWith('/host') || pathname.startsWith('/admin');
}

export function App() {
  const { pathname } = location;

  if (pathname.startsWith('/login')) return <LoginPage />;
  if (pathname.startsWith('/room') || pathname.startsWith('/live')) return <LiveRoomPage />;
  if (isAdminPath(pathname)) {
    return currentAuth().user ? <HostConsolePage /> : <LoginPage title="登录后进入管理后台" />;
  }

  return <HomePage />;
}
