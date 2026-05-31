import { useEffect, useState } from 'react';
import { formatLeftMs, getLeftMsWithOffset, getServerOffsetMs } from '../../../shared/lib/time';

export function useServerCountdown(
  endsAtUnixMs?: number | string,
  serverTimeUnixMs?: number | string,
  serverTimeReceivedAtUnixMs?: number,
) {
  const [leftMs, setLeftMs] = useState(0);

  useEffect(() => {
    let frame = 0;
    const offsetMs = getServerOffsetMs(serverTimeUnixMs, serverTimeReceivedAtUnixMs ?? Date.now());

    const tick = () => {
      setLeftMs(getLeftMsWithOffset(endsAtUnixMs, offsetMs));
      frame = window.requestAnimationFrame(tick);
    };

    frame = window.requestAnimationFrame(tick);
    return () => window.cancelAnimationFrame(frame);
  }, [endsAtUnixMs, serverTimeReceivedAtUnixMs, serverTimeUnixMs]);

  return {
    leftMs,
    text: formatLeftMs(leftMs),
    danger: leftMs > 0 && leftMs < 10000,
    ended: leftMs <= 0,
    fallback: !serverTimeUnixMs,
  };
}
