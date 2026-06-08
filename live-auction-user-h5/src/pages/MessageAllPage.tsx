import { useMemo, useState } from 'react';

type MessageFilter = {
  id: string;
  label: string;
  icon: string;
};

type ActivityMessage = {
  id: string;
  name: string;
  avatar: string;
  summary: string;
  time: string;
  posterTone: string;
  badge?: string;
};

const MESSAGE_FILTERS: MessageFilter[] = [
  { id: 'all', label: '全部消息', icon: '✓' },
  { id: 'like', label: '赞', icon: '♥' },
  { id: 'mention', label: '@我的', icon: '@' },
  { id: 'comment', label: '评论', icon: '✎' },
];

const ACTIVITY_MESSAGES: ActivityMessage[] = [
  {
    id: 'visitor-1',
    name: '青柠汽水',
    avatar: '青',
    summary: '近期访问过你的主页',
    time: '01-11',
    posterTone: 'cyan',
    badge: '新访客',
  },
  {
    id: 'visitor-2',
    name: '山海收藏家',
    avatar: '山',
    summary: '赞了你的直播回放',
    time: '昨天',
    posterTone: 'rose',
  },
  {
    id: 'visitor-3',
    name: '奶油小熊',
    avatar: '奶',
    summary: '@你看一场同城直播',
    time: '周二',
    posterTone: 'amber',
  },
  {
    id: 'visitor-4',
    name: 'LiveAuction 小助手',
    avatar: '拍',
    summary: '成交提醒和竞拍动态会同步到这里',
    time: '05-26',
    posterTone: 'dark',
    badge: '官方',
  },
];

export function MessageAllPage() {
  const [isFilterOpen, setFilterOpen] = useState(false);
  const [activeFilter, setActiveFilter] = useState(MESSAGE_FILTERS[0]);

  const messages = useMemo(() => {
    if (activeFilter.id === 'all') return ACTIVITY_MESSAGES;
    const keywordMap: Record<string, string> = {
      like: '赞',
      mention: '@',
      comment: '评论',
    };
    return ACTIVITY_MESSAGES.filter((item) => item.summary.includes(keywordMap[activeFilter.id] ?? ''));
  }, [activeFilter]);

  return (
    <main className="dyMsgPage dyMsgAllPage" aria-label="全部消息">
      <header className="dyMsgHeader">
        <button className="dyMsgHeaderIcon" type="button" aria-label="返回" onClick={() => history.back()}>
          ‹
        </button>
        <button className="dyMsgHeaderTitleButton" type="button" onClick={() => setFilterOpen((value) => !value)}>
          <span>{activeFilter.label}</span>
          <i className={isFilterOpen ? 'isOpen' : ''} aria-hidden="true">⌃</i>
        </button>
        <a className="dyMsgHeaderText" href="/message">消息</a>
      </header>

      {isFilterOpen ? (
        <section className="dyMsgFilterLayer" aria-label="消息分类">
          <div className="dyMsgFilterPanel">
            {MESSAGE_FILTERS.map((filter) => (
              <button
                className={filter.id === activeFilter.id ? 'active' : ''}
                type="button"
                key={filter.id}
                onClick={() => {
                  setActiveFilter(filter);
                  setFilterOpen(false);
                }}
              >
                <span>{filter.icon}</span>
                <b>{filter.label}</b>
              </button>
            ))}
          </div>
          <button className="dyMsgFilterMask" type="button" aria-label="关闭分类" onClick={() => setFilterOpen(false)} />
        </section>
      ) : null}

      <section className="dyMsgList" aria-label={`${activeFilter.label}列表`}>
        {messages.map((message) => (
          <a className="dyMsgActivityRow" href="/message/visitors" key={message.id}>
            <span className={`dyMsgAvatar dyMsgAvatar-${message.posterTone}`}>{message.avatar}</span>
            <span className="dyMsgActivityMain">
              <span className="dyMsgActivityName">
                <b>{message.name}</b>
                {message.badge ? <em>{message.badge}</em> : null}
              </span>
              <span className="dyMsgActivitySummary">
                <small>{message.summary}</small>
                <time>{message.time}</time>
              </span>
            </span>
            <span className={`dyMsgPoster dyMsgPoster-${message.posterTone}`} aria-hidden="true" />
          </a>
        ))}
      </section>
    </main>
  );
}
