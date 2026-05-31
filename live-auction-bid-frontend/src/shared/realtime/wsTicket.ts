import { apiRequest } from '../api/httpClient';

type WsTicketReply = {
  ticket?: string;
  scope?: string;
  expiresAtUnixMs?: number | string;
};

export async function getWsTicket(input: { roomId: string; scope: 'admin' | 'public' }): Promise<string> {
  const reply = await apiRequest<WsTicketReply>({
    path: '/api/realtime/ws-ticket',
    method: 'POST',
    body: input,
    auth: 'required',
    operation: `ws-ticket-${input.scope}`,
  });
  if (!reply.ticket) throw new Error('websocket ticket missing');
  return reply.ticket;
}
