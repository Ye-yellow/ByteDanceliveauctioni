import { useCallback, useEffect, useRef, useState } from 'react';
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
  source?: string;
  onSourceChange?: (source: string) => void;
};

export function LivePlayer({ poster, anchorName, onlineCount, wsState, roomName, source: controlledSource, onSourceChange }: Props) {
  const videoRef = useRef<HTMLVideoElement | null>(null);
  const [internalSource, setInternalSource] = useState(resolveInitialLiveSource);
  const source = controlledSource || internalSource;
  const [message, setMessage] = useState('');
  const playlist = resolveLivePlaylist();
  const hasPlaylistLoop = playlist.length > 1;
  const updateSource = useCallback((nextSource: string) => {
    if (onSourceChange) onSourceChange(nextSource);
    else setInternalSource(nextSource);
  }, [onSourceChange]);

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
      updateSource(nextSource);
    }
  }, [source, updateSource]);

  useEffect(() => {
    const video = videoRef.current;
    if (!video) return undefined;
    let disposed = false;
    const retryTimers: number[] = [];

    setMessage('直播画面加载中');
    video.pause();
    video.removeAttribute('src');

    if (isHls(source) && !video.canPlayType('application/vnd.apple.mpegurl')) {
      setMessage('当前浏览器不支持 HLS 直播源');
      if (hasPlaylistLoop) {
        retryTimers.push(window.setTimeout(() => {
          if (!disposed) goNext();
        }, 1200));
      }
      return () => {
        disposed = true;
        retryTimers.forEach((timer) => window.clearTimeout(timer));
      };
    }

    video.src = source;
    video.load();

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
      video.removeAttribute('src');
      video.load();
    };
  }, [goNext, hasPlaylistLoop, source]);

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
