import { Suspense, lazy } from 'react';

const HomePage = lazy(() => import('../pages/HomePage').then((module) => ({ default: module.HomePage })));
const LiveRoomPage = lazy(() => import('../pages/LiveRoomPage').then((module) => ({ default: module.LiveRoomPage })));
const ResultPage = lazy(() => import('../pages/ResultPage').then((module) => ({ default: module.ResultPage })));
const HistoryPage = lazy(() => import('../pages/HistoryPage').then((module) => ({ default: module.HistoryPage })));
const ProfilePage = lazy(() => import('../pages/ProfilePage').then((module) => ({ default: module.ProfilePage })));

function routeForPath(path: string) {
  const roomMatch = path.match(/^\/m\/room\/([^/]+)/);
  if (roomMatch) return <LiveRoomPage roomId={decodeURIComponent(roomMatch[1])} />;
  if (path.startsWith('/m/result/')) return <ResultPage />;
  if (path.startsWith('/m/history')) return <HistoryPage />;
  if (path.startsWith('/m/profile')) return <ProfilePage />;
  return <HomePage />;
}

export function Router() {
  return (
    <Suspense fallback={<main className="mobileShell"><section className="emptyState">正在加载页面...</section></main>}>
      {routeForPath(location.pathname)}
    </Suspense>
  );
}
