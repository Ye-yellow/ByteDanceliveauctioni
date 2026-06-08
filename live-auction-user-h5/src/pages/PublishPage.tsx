import { useEffect, useRef, useState } from 'react';
import './publish-replica.css';

const CAMERA_TOOLS = [
  { label: '翻转', icon: '↻' },
  { label: '闪光灯', icon: 'ϟ' },
  { label: '设置', icon: '⚙' },
  { label: '倒计时', icon: '◴' },
  { label: '美化', icon: '✦' },
  { label: '滤镜', icon: '◐' },
];

const PUBLISH_MODES = ['分段拍', '快拍', '影集', '开直播'];

function goBack() {
  if (window.history.length > 1) {
    window.history.back();
    return;
  }
  window.location.assign('/home');
}

export function PublishPage() {
  const videoRef = useRef<HTMLVideoElement | null>(null);
  const streamRef = useRef<MediaStream | null>(null);
  const [modeIndex, setModeIndex] = useState(1);
  const [cameraReady, setCameraReady] = useState(false);

  useEffect(() => {
    let mounted = true;

    async function openCamera() {
      if (!navigator.mediaDevices?.getUserMedia) return;
      try {
        const stream = await navigator.mediaDevices.getUserMedia({ video: true, audio: false });
        if (!mounted) {
          stream.getTracks().forEach((track) => track.stop());
          return;
        }
        streamRef.current = stream;
        if (videoRef.current) {
          videoRef.current.srcObject = stream;
          await videoRef.current.play().catch(() => undefined);
        }
        setCameraReady(true);
      } catch {
        setCameraReady(false);
      }
    }

    void openCamera();
    return () => {
      mounted = false;
      streamRef.current?.getTracks().forEach((track) => track.stop());
      streamRef.current = null;
    };
  }, []);

  return (
    <main className="dyPublishReplica" aria-label="发布">
      <video ref={videoRef} className="dyPublishReplicaVideo" autoPlay muted playsInline />
      <div className="dyPublishReplicaFallback" aria-hidden={cameraReady}>
        <span />
        <b>{cameraReady ? '' : '无法访问相机'}</b>
        <p>{cameraReady ? '' : '你仍可以查看抖音发布页的拍摄结构和操作入口'}</p>
      </div>

      <section className="dyPublishReplicaFloat">
        <button type="button" className="dyPublishReplicaClose" onClick={goBack} aria-label="关闭">
          ×
        </button>
        <a href="/home/music" className="dyPublishReplicaMusic">
          <span>♪</span>
          选择音乐
        </a>
        <nav className="dyPublishReplicaToolbar" aria-label="拍摄工具">
          {CAMERA_TOOLS.map((tool) => (
            <button type="button" key={tool.label}>
              <b>{tool.icon}</b>
              <span>{tool.label}</span>
            </button>
          ))}
        </nav>
      </section>

      <footer className="dyPublishReplicaFooter">
        <nav className="dyPublishReplicaModes" aria-label="发布模式">
          <i aria-hidden="true" />
          <i aria-hidden="true" />
          {PUBLISH_MODES.map((mode, index) => (
            <button
              type="button"
              className={modeIndex === index ? 'active' : ''}
              onClick={() => setModeIndex(index)}
              key={mode}
            >
              {mode}
            </button>
          ))}
        </nav>
        <div className="dyPublishReplicaCaptureRow">
          <a href="/me" aria-label="相册" className="dyPublishReplicaAlbum">
            ▣
          </a>
          <button type="button" className="dyPublishReplicaCapture" aria-label="拍摄">
            <span />
          </button>
          <a href={modeIndex === 3 ? '/home/live' : '/video-detail'} className="dyPublishReplicaNext">
            下一步
          </a>
        </div>
      </footer>
    </main>
  );
}
