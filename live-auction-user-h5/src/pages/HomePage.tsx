import Hls from 'hls.js';
import { useEffect, useMemo, useRef, useState } from 'react';
import { listPublicRooms, listRoomLots, getRoomSnapshot } from '../features/auction/api/auctionApi';
import { isHls, resolveLiveSource } from '../features/live/hooks/useLivePlayer';
import { roomVisualProfileAt } from '../shared/config/demoRooms';
import type { Lot, Room, RoomSnapshot } from '../shared/api/types';

type HomeFeedRoom = {
  roomId: string;
  roomName: string;
  anchorName: string;
  heatText: string;
  summary: string;
  videoUrl: string;
  lot: Lot | null;
};

type RoomPreview = {
  snapshot: RoomSnapshot | null;
  lots: Lot[];
};

const FALLBACK_VIDEO = '/demo-live.mp4';
const MAX_HOME_ROOMS = 12;
const FEED_VIDEO_SOURCES = [
  resolveLiveSource(),
  '/demo-live-02.mp4',
  '/demo-live-03.mp4',
  '/demo-live-04.mp4',
];

function feedVideoAt(index: number): string {
  return FEED_VIDEO_SOURCES[index % FEED_VIDEO_SOURCES.length] || FALLBACK_VIDEO;
}

function HomeTopTabs() {
  return (
    <header className="douyinTopTabs homeTopTabs" aria-label="首页频道">
      <button type="button" aria-label="菜单">☰</button>
      <span>热点</span>
      <span>团购</span>
      <span>关注</span>
      <span>商城</span>
      <span>深圳</span>
      <b>推荐</b>
      <button type="button" aria-label="搜索">⌕</button>
    </header>
  );
}

function HomeLivePreview({
  active,
  lot,
  source,
}: {
  active: boolean;
  lot: Lot | null;
  source: string;
}) {
  const videoRef = useRef<HTMLVideoElement | null>(null);

  useEffect(() => {
    const video = videoRef.current;
    if (!video) return undefined;

    let hls: Hls | null = null;
    const play = () => void video.play().catch(() => undefined);
    const fallback = () => {
      if (video.currentSrc.endsWith(FALLBACK_VIDEO) || video.src.endsWith(FALLBACK_VIDEO)) return;
      hls?.destroy();
      hls = null;
      video.loop = true;
      video.src = FALLBACK_VIDEO;
      play();
    };
    const onVideoError = () => fallback();

    if (!active) {
      video.pause();
      return undefined;
    }

    video.muted = true;
    video.playsInline = true;
    video.addEventListener('error', onVideoError);

    if (isHls(source)) {
      video.loop = false;
      if (video.canPlayType('application/vnd.apple.mpegurl')) {
        video.src = source;
        play();
      } else if (Hls.isSupported()) {
        hls = new Hls({ liveSyncDurationCount: 3 });
        hls.attachMedia(video);
        hls.on(Hls.Events.MEDIA_ATTACHED, () => hls?.loadSource(source));
        hls.on(Hls.Events.MANIFEST_PARSED, play);
        hls.on(Hls.Events.ERROR, (_, data) => {
          if (data.fatal) fallback();
        });
      } else {
        fallback();
      }
    } else {
      video.loop = true;
      video.src = source;
      play();
    }

    return () => {
      video.removeEventListener('error', onVideoError);
      video.pause();
      hls?.destroy();
    };
  }, [active, source]);

  return (
    <section className="homeLivePreview" aria-label="直播预览">
      <video ref={videoRef} className="homeLiveVideo" poster={lot?.imageUrl} autoPlay muted playsInline />
    </section>
  );
}

function createHomeFeedRooms(rooms: Room[], previews: Record<string, RoomPreview>): HomeFeedRoom[] {
  return rooms.map((room, index) => {
    const visual = roomVisualProfileAt(index);
    const preview = previews[room.id];
    const snapshot = preview?.snapshot || null;
    const lots = preview?.lots || [];
    const previewLot = snapshot?.currentLot || lots[0] || null;
    return {
      roomId: room.id,
      roomName: room.name || snapshot?.roomName || visual.roomName,
      anchorName: snapshot?.anchorName || room.name || visual.anchorName,
      heatText: visual.heatText,
      summary: previewLot?.title ? `正在竞拍：${previewLot.title}` : visual.summary,
      videoUrl: feedVideoAt(index),
      lot: previewLot,
    };
  });
}

function BottomNav({ roomId }: { roomId?: string }) {
  const profileHref = roomId ? `/m/profile?roomId=${encodeURIComponent(roomId)}` : '/m/profile';
  return (
    <nav className="homeBottomNav" aria-label="底部导航">
      <b>首页</b>
      <span aria-hidden="true" />
      <button type="button" aria-label="发布">+</button>
      <span aria-hidden="true" />
      <a href={profileHref}>我</a>
    </nav>
  );
}

function HomeFeedSlide({
  active,
  room,
}: {
  active: boolean;
  room: HomeFeedRoom;
}) {
  return (
    <section className={`homeFeedItem${active ? ' active' : ''}`} aria-hidden={!active} aria-label={room.roomName}>
      <HomeLivePreview active={active} lot={room.lot} source={room.videoUrl} />
      <a className="homeEnterLive" href={`/m/room/${encodeURIComponent(room.roomId)}`}>
        点击进入直播间
      </a>
      <section className="homeAnchorIntro">
        <div>
          <b>直播中</b>
          <span>{room.heatText}</span>
        </div>
        <h1>{room.anchorName}</h1>
        <p>{room.summary}</p>
      </section>
    </section>
  );
}

export function HomePage() {
  const initialRoomId = useMemo(() => new URLSearchParams(location.search).get('roomId') || '', []);
  const [publicRooms, setPublicRooms] = useState<Room[]>([]);
  const [previews, setPreviews] = useState<Record<string, RoomPreview>>({});
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [feedIndex, setFeedIndex] = useState(1);
  const [jumping, setJumping] = useState(false);
  const touchStartY = useRef<number | null>(null);
  const transitioningRef = useRef(false);
  const initialRoomAppliedRef = useRef(false);

  useEffect(() => {
    let disposed = false;
    setLoading(true);
    setError('');
    const timer = window.setTimeout(() => {
      void (async () => {
        try {
          const rooms = (await listPublicRooms()).slice(0, MAX_HOME_ROOMS);
          if (disposed) return;
          setPublicRooms(rooms);
          const entries = await Promise.all(rooms.map(async (room) => {
            const [snapshotResult, lotsResult] = await Promise.allSettled([
              getRoomSnapshot(room.id),
              listRoomLots(room.id),
            ]);
            return [
              room.id,
              {
                snapshot: snapshotResult.status === 'fulfilled' ? snapshotResult.value : null,
                lots: lotsResult.status === 'fulfilled' ? lotsResult.value : [],
              },
            ] as const;
          }));
          if (!disposed) setPreviews(Object.fromEntries(entries));
        } catch (e) {
          if (!disposed) {
            setPublicRooms([]);
            setPreviews({});
            setError(e instanceof Error ? e.message : '直播间加载失败');
          }
        } finally {
          if (!disposed) setLoading(false);
        }
      })();
    }, 0);
    return () => {
      disposed = true;
      window.clearTimeout(timer);
    };
  }, []);

  useEffect(() => {
    if (!initialRoomId || initialRoomAppliedRef.current || !publicRooms.length) return;
    const roomIndex = publicRooms.findIndex((room) => room.id === initialRoomId);
    if (roomIndex < 0) return;
    initialRoomAppliedRef.current = true;
    setFeedIndex(roomIndex + 1);
  }, [initialRoomId, publicRooms]);

  const rooms = useMemo(() => createHomeFeedRooms(publicRooms, previews), [previews, publicRooms]);
  const visibleRooms = useMemo(() => rooms.length ? [rooms[rooms.length - 1], ...rooms, rooms[0]] : [], [rooms]);
  const activeRoomIndex = rooms.length ? ((feedIndex - 1) % rooms.length + rooms.length) % rooms.length : 0;
  const activeRoom = rooms[activeRoomIndex];

  const moveFeed = (delta: number) => {
    if (rooms.length < 2) return;
    if (transitioningRef.current) return;
    transitioningRef.current = true;
    setFeedIndex((current) => current + delta);
  };

  const settleLoop = () => {
    if (!rooms.length) {
      transitioningRef.current = false;
      return;
    }
    if (feedIndex === 0) {
      setJumping(true);
      setFeedIndex(rooms.length);
      window.requestAnimationFrame(() => {
        window.requestAnimationFrame(() => {
          setJumping(false);
          transitioningRef.current = false;
        });
      });
      return;
    }
    if (feedIndex === rooms.length + 1) {
      setJumping(true);
      setFeedIndex(1);
      window.requestAnimationFrame(() => {
        window.requestAnimationFrame(() => {
          setJumping(false);
          transitioningRef.current = false;
        });
      });
      return;
    }
    transitioningRef.current = false;
  };

  if (!rooms.length) {
    return (
      <main className="mobileShell homeShell">
        <HomeTopTabs />
        <section className="emptyState">{loading ? '正在加载直播间...' : error || '暂无可进入的直播间'}</section>
        <BottomNav />
      </main>
    );
  }

  return (
    <main
      className="mobileShell homeShell"
      onWheel={(event) => {
        if (Math.abs(event.deltaY) < 20) return;
        event.preventDefault();
        moveFeed(event.deltaY > 0 ? 1 : -1);
      }}
      onTouchStart={(event) => {
        touchStartY.current = event.touches[0]?.clientY ?? null;
      }}
      onTouchEnd={(event) => {
        if (touchStartY.current == null) return;
        const endY = event.changedTouches[0]?.clientY ?? touchStartY.current;
        const deltaY = touchStartY.current - endY;
        touchStartY.current = null;
        if (Math.abs(deltaY) < 48) return;
        moveFeed(deltaY > 0 ? 1 : -1);
      }}
    >
      <HomeTopTabs />
      <section className="homeLivePager" aria-label="直播推荐流">
        <div
          className={`homeLiveTrack${jumping ? ' isJumping' : ''}`}
          style={{ transform: `translate3d(0, -${feedIndex * 100}%, 0)` }}
          onTransitionEnd={(event) => {
            if (event.target === event.currentTarget) settleLoop();
          }}
        >
          {visibleRooms.map((room, index) => (
            <HomeFeedSlide active={index === feedIndex} key={`${room.roomId}-${index}`} room={room} />
          ))}
        </div>
      </section>
      <BottomNav roomId={activeRoom?.roomId} />
    </main>
  );
}
