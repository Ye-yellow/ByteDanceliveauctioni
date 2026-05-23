import { createContext } from 'react';
import { authSession, type AuthSessionSnapshot } from './authSession';

export type AuthSessionContextValue = AuthSessionSnapshot & {
  authMode: ReturnType<typeof authSession.getAuthMode>;
  ensureBuyerSession: typeof authSession.ensureBuyerSession;
  ensureReadyForBid: typeof authSession.ensureReadyForBid;
  loginBuyer: typeof authSession.loginBuyer;
  registerBuyer: typeof authSession.registerBuyer;
  refreshIfNeeded: typeof authSession.refreshIfNeeded;
  logout: typeof authSession.logout;
};

export const AuthSessionContext = createContext<AuthSessionContextValue | null>(null);
