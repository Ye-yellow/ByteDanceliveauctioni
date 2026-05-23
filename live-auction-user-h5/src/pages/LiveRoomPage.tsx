import { LiveRoomView } from '../features/auction-room/components/LiveRoomView';
import { useLiveRoomController } from '../features/auction-room/hooks/useLiveRoomController';

export function LiveRoomPage({ roomId }: { roomId: string }) {
  const controller = useLiveRoomController(roomId);
  return <LiveRoomView controller={controller} />;
}
