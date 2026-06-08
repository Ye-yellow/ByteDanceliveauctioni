import { useEffect, useMemo, useRef, useState } from 'react';

type ChatMessage =
  | { id: string; type: 'time'; text: string }
  | { id: string; type: 'text'; from: 'me' | 'friend'; text: string; read?: string }
  | { id: string; type: 'image'; from: 'me' | 'friend'; tone: string }
  | { id: string; type: 'voice'; from: 'me' | 'friend'; duration: number }
  | { id: string; type: 'call'; from: 'me' | 'friend'; callType: '语音' | '视频'; state: '已取消' | '未接通' | '通话 21:44' }
  | { id: string; type: 'video'; from: 'me' | 'friend'; title: string; author: string; tone: string }
  | { id: string; type: 'redpacket'; from: 'me' | 'friend'; title: string; state: string };

const CHAT_MESSAGES: ChatMessage[] = [
  { id: 'time-1', type: 'time', text: '2021-01-02 21:21' },
  { id: 'meme-1', type: 'image', from: 'friend', tone: 'rose' },
  { id: 'img-1', type: 'image', from: 'me', tone: 'cyan' },
  { id: 'call-1', type: 'call', from: 'friend', callType: '视频', state: '未接通' },
  { id: 'voice-1', type: 'voice', from: 'me', duration: 10 },
  { id: 'text-1', type: 'text', from: 'friend', text: '又在刷抖音' },
  { id: 'text-2', type: 'text', from: 'friend', text: '我昨天 @ 你那个视频发给我下' },
  { id: 'text-3', type: 'text', from: 'me', text: '我找不到了', read: '已读' },
  {
    id: 'video-1',
    type: 'video',
    from: 'me',
    title: '服了，这个现场反应也太真实了',
    author: 'safasdfassafasdfas',
    tone: 'dark',
  },
  { id: 'packet-1', type: 'redpacket', from: 'friend', title: '大吉大利', state: '未领取' },
  { id: 'packet-2', type: 'redpacket', from: 'me', title: '大吉大利', state: '已过期' },
];

const CHAT_OPTIONS = ['照片', '拍摄', '红包', '视频通话', '语音通话', '一起看视频', '一起唱'];
const TOOLTIP_ACTIONS = ['点赞', '复制', '转发', '回复', '多选', '删除'];

function renderChatMessage(message: ChatMessage, onRedPacket: (message: ChatMessage) => void) {
  if (message.type === 'time') {
    return <time className="dyMsgChatTime">{message.text}</time>;
  }

  const isMe = message.from === 'me';
  const body = (() => {
    if (message.type === 'text') {
      return <p className="dyMsgChatText">{message.text}</p>;
    }
    if (message.type === 'image') {
      return <div className={`dyMsgChatImage dyMsgPoster-${message.tone}`} />;
    }
    if (message.type === 'voice') {
      return (
        <div className="dyMsgChatVoice">
          <span>▮▮▮</span>
          <b>{message.duration}"</b>
        </div>
      );
    }
    if (message.type === 'call') {
      return (
        <div className="dyMsgChatCall">
          <span>{message.callType}</span>
          <b>{message.state}</b>
        </div>
      );
    }
    if (message.type === 'video') {
      return (
        <article className="dyMsgChatVideoCard">
          <div className={`dyMsgChatVideoCover dyMsgPoster-${message.tone}`}>▶</div>
          <div>
            <b>{message.title}</b>
            <small>@{message.author}</small>
          </div>
        </article>
      );
    }
    return (
      <button className="dyMsgChatRedPacket" type="button" onClick={() => onRedPacket(message)}>
        <span>¥</span>
        <b>{message.title}</b>
        <small>{message.state}</small>
      </button>
    );
  })();

  return (
    <>
      <span className={`dyMsgAvatar ${isMe ? 'dyMsgAvatar-rose' : 'dyMsgAvatar-cyan'}`}>{isMe ? '我' : 'Z'}</span>
      <div className="dyMsgChatBubbleWrap">
        {body}
        {message.type === 'text' && message.read ? <small className="dyMsgChatRead">{message.read}</small> : null}
      </div>
    </>
  );
}

export function MessageChatPage() {
  const [showOptions, setShowOptions] = useState(false);
  const [recording, setRecording] = useState(false);
  const [tooltipId, setTooltipId] = useState('');
  const [redPacket, setRedPacket] = useState<ChatMessage | null>(null);
  const threadRef = useRef<HTMLDivElement>(null);
  const pressTimer = useRef<number | null>(null);

  useEffect(() => {
    const thread = threadRef.current;
    if (thread) {
      thread.scrollTop = thread.scrollHeight;
    }
  }, [showOptions, recording]);

  const tooltipTop = useMemo(() => {
    const index = CHAT_MESSAGES.findIndex((message) => message.id === tooltipId);
    return index > -1 ? `${Math.max(74, 114 + index * 44)}px` : '92px';
  }, [tooltipId]);

  function clearPressTimer() {
    if (pressTimer.current !== null) {
      window.clearTimeout(pressTimer.current);
      pressTimer.current = null;
    }
  }

  return (
    <main className="dyMsgPage dyMsgChatPage" aria-label="聊天">
      <section className="dyMsgChatSurface" onPointerDown={() => setTooltipId('')}>
        <header className="dyMsgChatHeader">
          <button className="dyMsgHeaderIcon" type="button" aria-label="返回" onClick={() => history.back()}>
            ‹
          </button>
          <div className="dyMsgChatPeer">
            <i>12</i>
            <b>zzzz</b>
          </div>
          <nav aria-label="聊天操作">
            <button type="button">☎</button>
            <button type="button">▣</button>
            <a href="/message/chat/detail">☰</a>
          </nav>
        </header>

        <div className={`dyMsgChatThread ${showOptions ? 'isExpanded' : ''}`} ref={threadRef}>
          {CHAT_MESSAGES.map((message) => (
            <div
              className={`dyMsgChatRow ${message.type === 'time' ? 'isTime' : ''} ${'from' in message && message.from === 'me' ? 'isMe' : 'isFriend'}`}
              key={message.id}
              onPointerDown={(event) => {
                event.stopPropagation();
                clearPressTimer();
                pressTimer.current = window.setTimeout(() => setTooltipId(message.id), 420);
              }}
              onPointerUp={clearPressTimer}
              onPointerCancel={clearPressTimer}
            >
              {renderChatMessage(message, setRedPacket)}
            </div>
          ))}
        </div>

        <footer className="dyMsgChatFooter">
          {!recording ? (
            <div className="dyMsgChatToolbar">
              <button type="button" aria-label="相机">▣</button>
              <input placeholder="发送信息..." onFocus={() => setShowOptions(false)} />
              <button
                type="button"
                aria-label="语音"
                onClick={() => {
                  setRecording(true);
                  setShowOptions(false);
                }}
              >
                ≋
              </button>
              <button type="button" aria-label="表情">☺</button>
              <button type="button" aria-label="更多" onClick={() => setShowOptions((value) => !value)}>
                ＋
              </button>
            </div>
          ) : (
            <div className="dyMsgChatRecord">
              <span>按住 说话</span>
              <button type="button" onClick={() => setRecording(false)}>⌨</button>
            </div>
          )}

          {showOptions ? (
            <div className="dyMsgChatOptions">
              {CHAT_OPTIONS.map((option, index) => (
                <button type="button" key={option}>
                  <span>{option.slice(0, 1)}</span>
                  <b>{option}</b>
                  {index === 2 ? <i>¥</i> : null}
                </button>
              ))}
            </div>
          ) : null}
        </footer>
      </section>

      {tooltipId ? (
        <div className="dyMsgChatTooltip" style={{ top: tooltipTop }}>
          {TOOLTIP_ACTIONS.map((action) => (
            <button type="button" key={action}>{action}</button>
          ))}
        </div>
      ) : null}

      {redPacket ? (
        <section className="dyMsgPacketLayer" aria-label="红包详情">
          <button className="dyMsgPacketMask" type="button" aria-label="关闭红包" onClick={() => setRedPacket(null)} />
          <article className="dyMsgPacketCard">
            <button type="button" aria-label="关闭" onClick={() => setRedPacket(null)}>×</button>
            <span className="dyMsgAvatar dyMsgAvatar-rose">我</span>
            <b>{redPacket.type === 'redpacket' ? redPacket.title : '大吉大利'}</b>
            <p>{redPacket.type === 'redpacket' && redPacket.state === '已过期' ? '红包已过期' : '来自好友的心意'}</p>
            <a href="/message/chat/red-packet-detail">查看红包详情 &gt;</a>
          </article>
        </section>
      ) : null}
    </main>
  );
}
