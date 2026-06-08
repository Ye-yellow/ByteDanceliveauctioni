export function getServerOffsetMs(serverTimeUnixMs?: number | string, receivedAtUnixMs = Date.now()): number {
  const serverTime = Number(serverTimeUnixMs || 0);
  if (!Number.isFinite(serverTime) || serverTime <= 0) return 0;
  return serverTime - receivedAtUnixMs;
}

export function getServerNowMs(serverTimeUnixMs?: number | string, receivedAtUnixMs = Date.now()): number {
  return Date.now() + getServerOffsetMs(serverTimeUnixMs, receivedAtUnixMs);
}

export function getLeftMs(
  endsAtUnixMs?: number | string,
  serverTimeUnixMs?: number | string,
  receivedAtUnixMs = Date.now(),
): number {
  const ends = Number(endsAtUnixMs || 0);
  if (!Number.isFinite(ends) || ends <= 0) return 0;
  return Math.max(0, ends - getServerNowMs(serverTimeUnixMs, receivedAtUnixMs));
}

export function getLeftMsWithOffset(endsAtUnixMs?: number | string, serverOffsetMs = 0): number {
  const ends = Number(endsAtUnixMs || 0);
  if (!Number.isFinite(ends) || ends <= 0) return 0;
  return Math.max(0, ends - (Date.now() + serverOffsetMs));
}

export function formatLeftMs(leftMs: number): string {
  if (leftMs <= 0) return '等待落锤';

  const totalSeconds = Math.floor(leftMs / 1000);
  const minutes = Math.floor(totalSeconds / 60);
  const seconds = totalSeconds % 60;

  if (leftMs < 60000) return `${seconds}.${Math.floor((leftMs % 1000) / 100)}`;
  return `${minutes}:${String(seconds).padStart(2, '0')}`;
}

export function formatEventTime(value?: number | string): string {
  const time = Number(value || Date.now());
  return new Date(time).toLocaleTimeString('zh-CN', {
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  });
}
