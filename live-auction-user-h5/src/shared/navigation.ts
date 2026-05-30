export const SPA_NAVIGATE_EVENT = 'live-auction:navigate';

export function navigateTo(href: string, options: { replace?: boolean } = {}) {
  if (!href) return;

  const url = new URL(href, window.location.origin);
  if (url.origin !== window.location.origin) {
    window.location.assign(href);
    return;
  }

  const next = `${url.pathname}${url.search}${url.hash}`;
  const current = `${window.location.pathname}${window.location.search}${window.location.hash}`;
  if (next !== current) {
    if (options.replace) window.history.replaceState(null, '', next);
    else window.history.pushState(null, '', next);
  }

  window.dispatchEvent(new CustomEvent(SPA_NAVIGATE_EVENT, { detail: { path: url.pathname } }));
}
