import { useEffect, useMemo, useState } from 'react';
import { listPublicRooms } from '../features/auction/api/auctionApi';
import { DOUYIN_LIVE_CHANNEL_INDEX, writeHomeReturnState } from '../features/douyin-shell/model/homeReturnState';
import { LiveRoomView } from '../features/live-room/components/LiveRoomView';
import { useLiveRoomController } from '../features/live-room/hooks/useLiveRoomController';
import { clearSearchAIState } from '../features/search/model/searchAIState';
import type { Room } from '../shared/api/types';
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
  const [publicRooms, setPublicRooms] = useState<Room[]>([]);
  const liveRoomIds = useMemo(() => publicRooms.map((room) => room.id).filter(Boolean), [publicRooms]);
  const currentRoomIndex = liveRoomIds.indexOf(roomId);
  const hasSiblingRooms = currentRoomIndex >= 0 && liveRoomIds.length > 1;
  const previousRoom = hasSiblingRooms
    ? publicRooms[(currentRoomIndex - 1 + publicRooms.length) % publicRooms.length]
    : undefined;
  const nextRoom = hasSiblingRooms
    ? publicRooms[(currentRoomIndex + 1) % publicRooms.length]
    : undefined;

  useEffect(() => {
    let disposed = false;
    void listPublicRooms()
      .then((rooms) => {
        if (!disposed) setPublicRooms(rooms);
      })
      .catch(() => {
        if (!disposed) setPublicRooms([]);
      });
    return () => {
      disposed = true;
    };
  }, []);

  const navigateSiblingRoom = (direction: 1 | -1) => {
    if (liveRoomIds.length <= 1) return;
    const currentIndex = liveRoomIds.indexOf(roomId);
    if (currentIndex < 0) return;
    const nextIndex = (currentIndex + direction + liveRoomIds.length) % liveRoomIds.length;
    const nextRoomId = liveRoomIds[nextIndex];
    if (nextRoomId && nextRoomId !== roomId) navigateTo(`/m/room/${encodeURIComponent(nextRoomId)}`, { replace: true });
  };

  const closeLiveRoom = () => {
    clearSearchAIState();
    writeHomeReturnState({
      baseIndex: 1,
      channelIndex: DOUYIN_LIVE_CHANNEL_INDEX,
      itemIndexes: [],
      targetLiveRoomId: roomId,
    });
    navigateTo('/home', { replace: true });
  };

  return (
    <LiveRoomView
      controller={controller}
      hasRoomSwipeTargets={hasSiblingRooms}
      previousRoom={previousRoom ? { id: previousRoom.id, name: previousRoom.name } : undefined}
      nextRoom={nextRoom ? { id: nextRoom.id, name: nextRoom.name } : undefined}
      onSwipeRoom={navigateSiblingRoom}
      onCloseRoom={closeLiveRoom}
    />
  );
}
