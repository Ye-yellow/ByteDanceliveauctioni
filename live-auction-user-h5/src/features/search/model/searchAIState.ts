import type { AIBuyerConsultReply } from '../../../shared/api/types';

const SEARCH_AI_STATE_STORAGE_KEY = 'live-auction-h5.search.ai-state.v1';
const SEARCH_AI_RESTORE_STORAGE_KEY = 'live-auction-h5.search.ai-restore-once.v1';

export type SavedSearchAIState = {
  query: string;
  reply: AIBuyerConsultReply | null;
  scrollY: number;
};

const EMPTY_SEARCH_AI_STATE: SavedSearchAIState = { query: '', reply: null, scrollY: 0 };

function storageAvailable() {
  return typeof window !== 'undefined' && Boolean(window.sessionStorage);
}

export function clearSearchAIState() {
  if (!storageAvailable()) return;
  window.sessionStorage.removeItem(SEARCH_AI_STATE_STORAGE_KEY);
  window.sessionStorage.removeItem(SEARCH_AI_RESTORE_STORAGE_KEY);
}

export function readSearchAIStateForRestore(): SavedSearchAIState {
  if (!storageAvailable()) return EMPTY_SEARCH_AI_STATE;
  if (window.sessionStorage.getItem(SEARCH_AI_RESTORE_STORAGE_KEY) !== '1') {
    clearSearchAIState();
    return EMPTY_SEARCH_AI_STATE;
  }

  try {
    const parsed = JSON.parse(window.sessionStorage.getItem(SEARCH_AI_STATE_STORAGE_KEY) || '{}') as Partial<SavedSearchAIState>;
    return {
      query: typeof parsed.query === 'string' ? parsed.query : '',
      reply: parsed.reply && Array.isArray(parsed.reply.results) ? parsed.reply as AIBuyerConsultReply : null,
      scrollY: Number(parsed.scrollY || 0),
    };
  } catch {
    return EMPTY_SEARCH_AI_STATE;
  } finally {
    clearSearchAIState();
  }
}

export function saveSearchAIStateForRoomReturn(query: string, reply: AIBuyerConsultReply | null, scrollY = 0) {
  if (!storageAvailable()) return;
  window.sessionStorage.setItem(SEARCH_AI_STATE_STORAGE_KEY, JSON.stringify({ query, reply, scrollY }));
  window.sessionStorage.setItem(SEARCH_AI_RESTORE_STORAGE_KEY, '1');
}
