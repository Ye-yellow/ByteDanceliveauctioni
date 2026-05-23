import { useEffect, useRef, useState } from 'react';
import Player from 'xgplayer';
import type { IPlayerOptions } from 'xgplayer';
import HlsPlugin from 'xgplayer-hls-live';
import Hls from 'hls.js';
import 'xgplayer/dist/index.min.css';
import { isHls, resolveLiveSource } from '../hooks/useLivePlayer';
import { LiveOverlay } from './LiveOverlay';

type Props = { poster?: string; anchorName?: string; onlineCount?: number; wsState: string; roomName: string };
export function LivePlayer({ poster, anchorName, onlineCount, wsState, roomName }: Props) {
  const containerRef = useRef<HTMLDivElement | null>(null); const hlsVideoRef = useRef<HTMLVideoElement | null>(null); const playerRef = useRef<Player | null>(null); const hlsRef = useRef<Hls | null>(null);
  const [source, setSource] = useState(resolveLiveSource()); const [mode, setMode] = useState<'xgplayer' | 'hlsjs' | 'poster'>('xgplayer'); const [message, setMessage] = useState('');
  useEffect(() => { let disposed = false; const resetTimer = window.setTimeout(() => { if (!disposed) { setMode('xgplayer'); setMessage(''); } }, 0); const fallbackToMp4 = () => { if (disposed) return; if (source !== '/demo-live.mp4') { setMessage('直播流加载失败，正在切换备用源'); setSource('/demo-live.mp4'); } else { setMode('poster'); setMessage('直播流加载失败，正在切换备用源'); } };
    const initXg = () => { if (!containerRef.current) return; playerRef.current?.destroy(); containerRef.current.innerHTML = ''; try { const config = { el: containerRef.current, url: source, poster, autoplay: true, muted: true, playsinline: true, loop: !isHls(source), fluid: true, controls: true, lang: 'zh-cn', ...(isHls(source) ? { plugins: [HlsPlugin] } : {}) } as IPlayerOptions; const player = new Player(config); playerRef.current = player; (player as unknown as { on: (event: string, handler: () => void) => void }).on('error', () => { if (isHls(source) && Hls.isSupported()) setMode('hlsjs'); else fallbackToMp4(); }); } catch { if (isHls(source) && Hls.isSupported()) setMode('hlsjs'); else fallbackToMp4(); } };
    initXg(); return () => { disposed = true; window.clearTimeout(resetTimer); playerRef.current?.destroy(); playerRef.current = null; };
  }, [source, poster]);
  useEffect(() => { if (mode !== 'hlsjs' || !hlsVideoRef.current) return; const video = hlsVideoRef.current; hlsRef.current?.destroy(); const hls = new Hls({ liveSyncDurationCount: 3 }); hlsRef.current = hls; hls.attachMedia(video); hls.on(Hls.Events.MEDIA_ATTACHED, () => hls.loadSource(source)); hls.on(Hls.Events.ERROR, (_, data) => { if (data.fatal) { setSource('/demo-live.mp4'); setMode('xgplayer'); } }); return () => { hls.destroy(); hlsRef.current = null; }; }, [mode, source]);
  return <section className="livePlayerShell"><div className="xgMount" ref={containerRef} style={{ display: mode === 'xgplayer' ? 'block' : 'none' }} />{mode === 'hlsjs' ? <video ref={hlsVideoRef} className="nativeLiveVideo" poster={poster} autoPlay muted playsInline controls /> : null}{mode === 'poster' ? <div className="posterFallback">{poster ? <img src={poster} alt="直播备用海报" /> : null}<p>{message}</p></div> : null}{message ? <div className="playerMessage">{message}</div> : null}<LiveOverlay anchorName={anchorName} onlineCount={onlineCount} wsState={wsState} roomName={roomName} /></section>;
}
