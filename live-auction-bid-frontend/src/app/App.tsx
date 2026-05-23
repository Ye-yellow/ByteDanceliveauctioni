import { Suspense, lazy, type ReactNode } from 'react';
import { AuthSessionProvider, ProtectedRoute } from '../shared/auth/AuthSessionProvider';
import { ADMIN_ACCESS_ROLES } from '../shared/api/types';

const HomePage = lazy(() => import('../pages/home/HomePage').then((module) => ({ default: module.HomePage })));
const LoginPage = lazy(() => import('../pages/login/LoginPage').then((module) => ({ default: module.LoginPage })));
const HostConsolePage = lazy(() => import('../pages/host-console/HostConsolePage').then((module) => ({ default: module.HostConsolePage })));

function isAdminPath(pathname: string) {
  return pathname.startsWith('/host') || pathname.startsWith('/admin');
}

export function App() {
  const { pathname } = location;

  if (pathname.startsWith('/login')) return <AuthSessionProvider><RouteSuspense><LoginPage /></RouteSuspense></AuthSessionProvider>;
  if (isAdminPath(pathname)) return <AuthSessionProvider><RouteSuspense><ProtectedRoute requiredRoles={ADMIN_ACCESS_ROLES}><HostConsolePage /></ProtectedRoute></RouteSuspense></AuthSessionProvider>;

  return <AuthSessionProvider><RouteSuspense><HomePage /></RouteSuspense></AuthSessionProvider>;
}

function RouteSuspense({ children }: { children: ReactNode }) {
  return <Suspense fallback={<main className="routeLoading" aria-busy="true"><span>LiveAuction Studio</span><b>正在加载工作台…</b></main>}>
    {children}
  </Suspense>;
}
