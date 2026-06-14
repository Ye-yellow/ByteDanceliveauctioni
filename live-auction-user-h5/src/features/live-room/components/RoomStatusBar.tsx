import type { AuctionRoomState } from '../../../shared/api/types';

function roomSourceText(source: AuctionRoomState['eventState']['source']): string {
  if (source === 'snapshot') return '快照已同步';
  if (source === 'websocket') return '实时事件同步';
  if (source === 'local') return '本地待确认';
  return '初始化中';
}

export function RoomStatusBar({ source }: { roomId: string; source: AuctionRoomState['eventState']['source'] }) {
  return (
    <nav className="mobileQuickNav">
      <a href="/shop/orders?from=room">我的订单</a>
      <span>{roomSourceText(source)}</span>
    </nav>
  );
}
