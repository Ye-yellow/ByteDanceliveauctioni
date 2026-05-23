export function resolveLiveSource() { const configured = import.meta.env.VITE_DEMO_LIVE_URL as string | undefined; return configured?.trim() || '/demo-live.mp4'; }
export function isHls(url: string) { return /\.m3u8($|\?)/i.test(url); }
