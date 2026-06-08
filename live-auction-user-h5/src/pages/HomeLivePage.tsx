import { useEffect, useState } from 'react';
import { listPublicRooms } from '../features/auction/api/auctionApi';
import type { Room } from '../shared/api/types';
import './home-live-replica.css';

const CHANNELS = ['关注', '推荐', '同城', '严选'];
const FALLBACK_ROOM_NAMES = ['严选好物直播', '城市生活馆', '今晚开箱', '潮流集合店'];

function backHome() {
  if (window.history.length > 1) {
    window.history.back();
    return;
  }
  window.location.assign('/home');
}

function roomName(room: Room, index: number) {
  return room.name || FALLBACK_ROOM_NAMES[index % FALLBACK_ROOM_NAMES.length] || `直播间${index + 1}`;
}

function roomHref(room: Room) {
  return `/m/room/${encodeURIComponent(room.id)}`;
}

function popularity(index: number) {
  if (index === 0) return '999w';
  return `${8 + index}.${(index * 7) % 10}w`;
}

function RoomVisual({ name, index, featured = false, live = true }: { name: string; index: number; featured?: boolean; live?: boolean }) {
  return (
    <span className={`dyLiveReplicaVisual tone-${index % 5}${featured ? ' featured' : ''}`} aria-hidden="true">
      <i>{live ? 'LIVE' : 'WAIT'}</i>
      <b>{name.slice(0, 2)}</b>
      <em>{live ? `${popularity(index)}人气` : '等待开播'}</em>
    </span>
  );
}

function RoomCard({ room, index }: { room: Room; index: number }) {
  const name = roomName(room, index);
  return (
    <a className="dyLiveReplicaCard" href={roomHref(room)}>
      <RoomVisual name={name} index={index} />
      <span className="dyLiveReplicaCardMeta">
        <b>{name}</b>
        <small>{room.platform || 'LiveAuction'} · 正在直播</small>
      </span>
    </a>
  );
}

export function HomeLivePage() {
  const [activeChannel, setActiveChannel] = useState(1);
  const [rooms, setRooms] = useState<Room[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  const visibleRooms = rooms;
  const featuredRoom = visibleRooms[0];

  function loadRooms() {
    setLoading(true);
    setError('');
    void listPublicRooms()
      .then((nextRooms) => setRooms(nextRooms))
      .catch((reason) => {
        setRooms([]);
        setError(reason instanceof Error ? reason.message : '直播列表加载失败');
      })
      .finally(() => setLoading(false));
  }

  useEffect(() => {
    let disposed = false;
    void listPublicRooms()
      .then((nextRooms) => {
        if (!disposed) setRooms(nextRooms);
      })
      .catch((reason) => {
        if (disposed) return;
        setRooms([]);
        setError(reason instanceof Error ? reason.message : '直播列表加载失败');
      })
      .finally(() => {
        if (!disposed) setLoading(false);
      });
    return () => {
      disposed = true;
    };
  }, []);

  return (
    <main className="mobileShell dyLiveReplicaPage">
      <header className="dyLiveReplicaTopbar">
        <button type="button" aria-label="返回首页" onClick={backHome}>‹</button>
        <nav aria-label="直播频道">
          {CHANNELS.map((channel, index) => (
            <button
              type="button"
              className={activeChannel === index ? 'active' : ''}
              onClick={() => setActiveChannel(index)}
              key={channel}
            >
              {channel}
            </button>
          ))}
        </nav>
        <a href="/home/search" aria-label="搜索">⌕</a>
      </header>

      <section className="dyLiveReplicaStage">
        {featuredRoom ? (
          <a className="dyLiveReplicaHero" href={roomHref(featuredRoom)}>
            <RoomVisual name={roomName(featuredRoom, 0)} index={0} featured />
            <span>
              <i>直播中</i>
              <b>{roomName(featuredRoom, 0)}</b>
              <small>{featuredRoom.platform || 'LiveAuction'} · 今日精选讲解</small>
            </span>
          </a>
        ) : (
          <section className="dyLiveReplicaHero empty">
            <RoomVisual name="暂无直播" index={0} featured live={false} />
            <span>
              <i>{loading ? '加载中' : '直播中'}</i>
              <b>{loading ? '正在寻找直播间' : '暂无正在直播'}</b>
              <small>有真实房间开播后会显示在这里</small>
            </span>
          </section>
        )}

        <section className="dyLiveReplicaStrip" aria-label="正在直播">
          {visibleRooms.length ? (
            visibleRooms.slice(1, 5).map((room, index) => (
              <a href={roomHref(room)} key={room.id}>
                <RoomVisual name={roomName(room, index + 1)} index={index + 1} />
                <b>{roomName(room, index + 1)}</b>
              </a>
            ))
          ) : FALLBACK_ROOM_NAMES.map((name, index) => (
              <span className="dyLiveReplicaPlaceholder" key={name}>
                <RoomVisual name={name} index={index + 1} live={false} />
                <b>{name}</b>
              </span>
            ))}
        </section>
      </section>

      <section className="dyLiveReplicaGrid" aria-label="直播列表">
        <header>
          <b>{CHANNELS[activeChannel]}直播</b>
          <button type="button" onClick={loadRooms}>{loading ? '刷新中' : '换一批'}</button>
        </header>
        {error ? <p className="dyLiveReplicaError">{error}</p> : null}
        <div>
          {visibleRooms.length ? visibleRooms.map((room, index) => (
            <RoomCard room={room} index={index} key={room.id} />
          )) : <p className="dyLiveReplicaEmpty">当前没有可进入的真实直播间，先回推荐流继续看视频。</p>}
        </div>
      </section>
    </main>
  );
}
