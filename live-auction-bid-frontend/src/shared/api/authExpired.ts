import { authSession } from '../auth/authSession';

let redirecting = false;

export function forceRelogin(message = '登录已过期，请重新登录') {
  authSession.expire(message);
  if (redirecting) return;
  redirecting = true;
  const current = `${location.pathname}${location.search}${location.hash}`;
  const next = current.startsWith('/login') ? '/host' : current;
  location.href = `/login?expired=1&next=${encodeURIComponent(next)}`;
}
