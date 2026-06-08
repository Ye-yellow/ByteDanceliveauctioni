const explicitApiBase = import.meta.env.VITE_API_BASE as string | undefined;
const explicitWsBase = import.meta.env.VITE_WS_BASE as string | undefined;

// 默认走同源代理：浏览器只访问前端端口，Vite 再代理到后端 18080。
// 这样能避免 WSL/Windows 下浏览器跨端口访问 18080 失败。
export const API_BASE = explicitApiBase ?? '';
export const WS_BASE = explicitWsBase ?? `${location.protocol === 'https:' ? 'wss' : 'ws'}://${location.host}`;
