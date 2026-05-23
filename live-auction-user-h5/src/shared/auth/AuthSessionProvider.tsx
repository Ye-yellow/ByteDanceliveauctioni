import { useEffect, useMemo, useState, type ReactNode } from 'react';
import { authSession } from './authSession';
import { AuthSessionContext, type AuthSessionContextValue } from './authSessionContext';

export function AuthSessionProvider({ children }: { children: ReactNode }) {
  const [snapshot, setSnapshot] = useState(authSession.getSnapshot());

  useEffect(() => authSession.subscribe(setSnapshot), []);

  useEffect(() => {
    if (authSession.getAuthMode() === 'demo') {
      void authSession.ensureBuyerSession().catch((error) => {
        authSession.expire(error instanceof Error ? error.message : '无法建立 H5 买家登录态');
      });
    } else {
      void authSession.refreshIfNeeded();
    }

    const interval = window.setInterval(() => {
      void authSession.refreshIfNeeded();
    }, 30_000);

    const checkWhenVisible = () => {
      if (document.visibilityState === 'visible') void authSession.refreshIfNeeded();
    };

    window.addEventListener('focus', checkWhenVisible);
    document.addEventListener('visibilitychange', checkWhenVisible);

    return () => {
      window.clearInterval(interval);
      window.removeEventListener('focus', checkWhenVisible);
      document.removeEventListener('visibilitychange', checkWhenVisible);
    };
  }, []);

  const value = useMemo<AuthSessionContextValue>(() => ({
    ...snapshot,
    authMode: authSession.getAuthMode(),
    ensureBuyerSession: () => authSession.ensureBuyerSession(),
    ensureReadyForBid: () => authSession.ensureReadyForBid(),
    loginBuyer: (username: string, password: string) => authSession.loginBuyer(username, password),
    registerBuyer: (username: string, password: string, nickname: string) => authSession.registerBuyer(username, password, nickname),
    refreshIfNeeded: () => authSession.refreshIfNeeded(),
    logout: () => authSession.logout(),
  }), [snapshot]);

  return <AuthSessionContext.Provider value={value}>{children}</AuthSessionContext.Provider>;
}
