import { useEffect } from 'react';
import { listPublicRooms } from '../features/auction/api/auctionApi';
import { LiveRoomView } from '../features/live-room/components/LiveRoomView';
import { useLiveRoomController } from '../features/live-room/hooks/useLiveRoomController';
import { navigateTo } from '../shared/navigation';

export function LiveRoomPage({ roomId }: { roomId: string }) {
  if (roomId === 'demo-room') return <DemoRoomRedirect />;
  return <ResolvedLiveRoom roomId={roomId} />;
}

function DemoRoomRedirect() {
  useEffect(() => {
    let disposed = false;
    void listPublicRooms()
      .then((rooms) => {
        if (disposed) return;
        const firstRoom = rooms[0];
        navigateTo(firstRoom ? `/m/room/${encodeURIComponent(firstRoom.id)}` : '/home/live', { replace: true });
      })
      .catch(() => {
        if (!disposed) navigateTo('/home/live', { replace: true });
      });
    return () => {
      disposed = true;
    };
  }, []);

  return (
    <main className="mobileShell liveRoomPage">
      <section className="emptyState">正在进入直播间</section>
    </main>
  );
}

function ResolvedLiveRoom({ roomId }: { roomId: string }) {
  const controller = useLiveRoomController(roomId);
  return <LiveRoomView controller={controller} />;
}
