export const DOUYIN_LIVE_CHANNEL_INDEX = 3;

export type HomeReturnState = {
  baseIndex: number;
  channelIndex: number;
  itemIndexes: number[];
  targetLiveRoomId?: string;
  at: number;
};

const HOME_RETURN_STATE_KEY = 'douyin-home-return-state';
const HOME_RETURN_STATE_TTL_MS = 5 * 60 * 1000;

export function readHomeReturnState(): HomeReturnState | null {
  try {
    const raw = sessionStorage.getItem(HOME_RETURN_STATE_KEY);
    if (!raw) return null;
    sessionStorage.removeItem(HOME_RETURN_STATE_KEY);
    const parsed = JSON.parse(raw) as Partial<HomeReturnState>;
    if (!parsed || Date.now() - Number(parsed.at || 0) > HOME_RETURN_STATE_TTL_MS) return null;
    if (!Array.isArray(parsed.itemIndexes)) return null;
    return {
      baseIndex: Number(parsed.baseIndex),
      channelIndex: Number(parsed.channelIndex),
      itemIndexes: parsed.itemIndexes.map((value) => Number(value) || 0),
      targetLiveRoomId: typeof parsed.targetLiveRoomId === 'string' ? parsed.targetLiveRoomId : undefined,
      at: Number(parsed.at),
    };
  } catch {
    sessionStorage.removeItem(HOME_RETURN_STATE_KEY);
    return null;
  }
}

export function writeHomeReturnState(state: Omit<HomeReturnState, 'at'>) {
  sessionStorage.setItem(HOME_RETURN_STATE_KEY, JSON.stringify({ ...state, at: Date.now() }));
}

export function liveRoomIdFromHref(href?: string): string {
  if (!href) return '';
  try {
    const url = new URL(href, window.location.origin);
    const match = url.pathname.match(/^\/m\/room\/([^/]+)$/);
    return match ? decodeURIComponent(match[1] || '') : '';
  } catch {
    return '';
  }
}
