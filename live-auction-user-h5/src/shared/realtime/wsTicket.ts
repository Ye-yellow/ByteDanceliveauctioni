import { authSession } from '../auth/authSession';
import { apiRequest } from '../api/httpClient';

type WsTicketReply = {
  ticket?: string;
  scope?: string;
  expiresAtUnixMs?: number | string;
};

export async function getPublicWsTicket(roomId: string): Promise<string | null> {
  const token = await authSession.getValidAccessToken();
  if (!token) return null;
  const reply = await apiRequest<WsTicketReply>({
    path: '/api/realtime/ws-ticket',
    method: 'POST',
    body: { roomId, scope: 'public' },
    auth: 'required',
    operation: 'ws-ticket-public',
  });
  return reply.ticket || null;
}
