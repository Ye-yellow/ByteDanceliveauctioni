import { clearAuthState } from '../../features/auth/api/authStore';

let redirecting = false;

export function forceRelogin(message = '登录已过期，请重新登录') {
  clearAuthState();
  if (redirecting) return;
  redirecting = true;
  try {
    sessionStorage.setItem('liveauction.auth.expiredMessage', message);
  } catch {
    // ignore storage errors
  }
  const current = `${location.pathname}${location.search}${location.hash}`;
  const next = current.startsWith('/login') ? '/host' : current;
  location.href = `/login?expired=1&next=${encodeURIComponent(next)}`;
}
