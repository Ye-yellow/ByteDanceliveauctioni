import { useCallback, useEffect, useRef, useState } from 'react';
import Hls from 'hls.js';
import {
  isHls,
  resolveInitialLiveSource,
  resolveLivePlaylist,
  resolveNextLiveSource,
} from '../hooks/useLivePlayer';
import { LiveOverlay } from './LiveOverlay';

type Props = {
  poster?: string;
  anchorName?: string;
  onlineCount?: number;
  wsState: string;
  roomName: string;
};

export function LivePlayer({ poster, anchorName, onlineCount, wsState, roomName }: Props) {
  const videoRef = useRef<HTMLVideoElement | null>(null);
  const hlsRef = useRef<Hls | null>(null);
  const [source, setSource] = useState(resolveInitialLiveSource);
  const [message, setMessage] = useState('');
  const playlist = resolveLivePlaylist();
  const hasPlaylistLoop = playlist.length > 1;

  const playVideo = () => {
    const video = videoRef.current;
    if (!video) return;
    video.muted = true;
    video.volume = 0;
    video.setAttribute('muted', '');
    video.setAttribute('playsinline', '');
    void video.play().catch(() => undefined);
  };

  const goNext = useCallback(() => {
    const nextSource = resolveNextLiveSource(source);
    if (nextSource && nextSource !== source) {
      setMessage('直播画面加载中');
      setSource(nextSource);
    }
  }, [source]);

  useEffect(() => {
    const video = videoRef.current;
    if (!video) return undefined;
    let disposed = false;
    const retryTimers: number[] = [];

    setMessage('直播画面加载中');
    hlsRef.current?.destroy();
    hlsRef.current = null;

    if (isHls(source) && Hls.isSupported()) {
      const hls = new Hls({ liveSyncDurationCount: 3 });
      hlsRef.current = hls;
      hls.attachMedia(video);
      hls.on(Hls.Events.MEDIA_ATTACHED, () => hls.loadSource(source));
      hls.on(Hls.Events.ERROR, (_, data) => {
        if (data.fatal && !disposed) goNext();
      });
    } else {
      video.src = source;
      video.load();
    }

    const tryPlay = () => {
      if (!disposed) playVideo();
    };
    retryTimers.push(
      window.setTimeout(tryPlay, 0),
      window.setTimeout(tryPlay, 250),
      window.setTimeout(tryPlay, 800),
      window.setTimeout(tryPlay, 1500),
    );

    return () => {
      disposed = true;
      retryTimers.forEach((timer) => window.clearTimeout(timer));
      hlsRef.current?.destroy();
      hlsRef.current = null;
    };
  }, [goNext, source]);

  return (
    <section className="livePlayerShell">
      <video
        ref={videoRef}
        className="nativeLiveVideo"
        poster={poster}
        autoPlay
        muted
        playsInline
        preload="auto"
        loop={!hasPlaylistLoop && !isHls(source)}
        onCanPlay={() => {
          setMessage('');
          playVideo();
        }}
        onPlaying={() => setMessage('')}
        onWaiting={() => setMessage('直播画面加载中')}
        onEnded={() => {
          if (hasPlaylistLoop) goNext();
        }}
        onError={() => {
          if (hasPlaylistLoop) goNext();
          else setMessage('直播画面加载失败');
        }}
      />
      {message ? <div className="playerMessage">{message}</div> : null}
      <LiveOverlay anchorName={anchorName} onlineCount={onlineCount} wsState={wsState} roomName={roomName} />
    </section>
  );
}
