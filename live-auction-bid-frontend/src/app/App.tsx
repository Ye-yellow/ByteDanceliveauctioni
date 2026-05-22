import { Suspense, lazy, type ReactNode } from 'react';
import { currentAuth } from '../features/auth/api/authApi';

const HomePage = lazy(() => import('../pages/home/HomePage').then((module) => ({ default: module.HomePage })));
const LoginPage = lazy(() => import('../pages/login/LoginPage').then((module) => ({ default: module.LoginPage })));
const HostConsolePage = lazy(() => import('../pages/host-console/HostConsolePage').then((module) => ({ default: module.HostConsolePage })));

function isAdminPath(pathname: string) {
  return pathname.startsWith('/host') || pathname.startsWith('/admin');
}

export function App() {
  const { pathname } = location;

  if (pathname.startsWith('/login')) return <RouteSuspense><LoginPage /></RouteSuspense>;
  if (isAdminPath(pathname)) {
    return <RouteSuspense>{currentAuth().user ? <HostConsolePage /> : <LoginPage title="登录后进入管理后台" />}</RouteSuspense>;
  }

  return <RouteSuspense><HomePage /></RouteSuspense>;
}

function RouteSuspense({ children }: { children: ReactNode }) {
  return <Suspense fallback={<main className="routeLoading" aria-busy="true"><span>LiveAuction Studio</span><b>正在加载工作台…</b></main>}>
    {children}
  </Suspense>;
}
