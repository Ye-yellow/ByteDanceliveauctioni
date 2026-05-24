import Hls from 'hls.js';
import { useEffect, useMemo, useRef, useState } from 'react';
import { listRoomLots, getRoomSnapshot } from '../features/auction/api/auctionApi';
import { isHls, resolveLiveSource } from '../features/live/hooks/useLivePlayer';
import { DEMO_ROOM_PROFILES } from '../shared/config/demoRooms';
import type { Lot, RoomSnapshot } from '../shared/api/types';

type HomeFeedRoom = {
  roomId: string;
  roomName: string;
  anchorName: string;
  heatText: string;
  summary: string;
  videoUrl: string;
  lot: Lot | null;
};

const FALLBACK_VIDEO = '/demo-live.mp4';

const DEMO_FEED_TEMPLATES = DEMO_ROOM_PROFILES.map((profile, index) => ({
  ...profile,
  videoUrl: [
    resolveLiveSource(),
    '/demo-live-02.mp4',
    '/demo-live-03.mp4',
    '/demo-live-04.mp4',
  ][index] || FALLBACK_VIDEO,
}));

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

function createHomeFeedRooms(roomId: string, snapshot: RoomSnapshot | null, lots: Lot[]): HomeFeedRoom[] {
  const previewLot = snapshot?.currentLot || lots[0] || null;
  return DEMO_FEED_TEMPLATES.map((template, index) => {
    if (index === 0) {
      return {
        ...template,
        roomId,
        roomName: template.roomName,
        summary: previewLot?.title ? `今晚主拍：${previewLot.title}` : template.summary,
        lot: previewLot,
      };
    }
    return {
      ...template,
      lot: lots[index] || previewLot,
    };
  });
}

function BottomNav({ roomId }: { roomId: string }) {
  return (
    <nav className="homeBottomNav" aria-label="底部导航">
      <b>首页</b>
      <span aria-hidden="true" />
      <button type="button" aria-label="发布">+</button>
      <span aria-hidden="true" />
      <a href={`/m/profile?roomId=${encodeURIComponent(roomId)}`}>我</a>
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

export function HomePage({ roomId }: { roomId: string }) {
  const [snapshot, setSnapshot] = useState<RoomSnapshot | null>(null);
  const [lots, setLots] = useState<Lot[]>([]);
  const [feedIndex, setFeedIndex] = useState(1);
  const [jumping, setJumping] = useState(false);
  const touchStartY = useRef<number | null>(null);
  const transitioningRef = useRef(false);

  useEffect(() => {
    let disposed = false;
    const timer = window.setTimeout(() => {
      void Promise.allSettled([getRoomSnapshot(roomId), listRoomLots(roomId)]).then(([snapshotResult, lotsResult]) => {
        if (disposed) return;
        if (snapshotResult.status === 'fulfilled') setSnapshot(snapshotResult.value);
        if (lotsResult.status === 'fulfilled') setLots(lotsResult.value);
      });
    }, 0);
    return () => {
      disposed = true;
      window.clearTimeout(timer);
    };
  }, [roomId]);

  const rooms = useMemo(() => createHomeFeedRooms(roomId, snapshot, lots), [lots, roomId, snapshot]);
  const visibleRooms = useMemo(() => [rooms[rooms.length - 1], ...rooms, rooms[0]], [rooms]);
  const activeRoomIndex = ((feedIndex - 1) % rooms.length + rooms.length) % rooms.length;
  const activeRoom = rooms[activeRoomIndex];

  const moveFeed = (delta: number) => {
    if (transitioningRef.current) return;
    transitioningRef.current = true;
    setFeedIndex((current) => current + delta);
  };

  const settleLoop = () => {
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
      <BottomNav roomId={activeRoom.roomId} />
    </main>
  );
}
