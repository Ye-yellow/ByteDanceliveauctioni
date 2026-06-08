import { useMemo, useState } from 'react';
import { DouyinTabBar } from '../shared/ui/DouyinTabBar';
import './message-replica.css';

type Tone = 'cyan' | 'rose' | 'amber' | 'violet' | 'green' | 'blue' | 'dark';

type Friend = {
  id: string;
  name: string;
  avatar: string;
  account: string;
  tone: Tone;
};

type MessageRow = {
  id: string;
  title: string;
  avatar: string;
  tone: Tone;
  detail: string;
  time?: string;
  href: string;
  tag?: string;
  unread?: string;
  dot?: boolean;
  arrow?: boolean;
  online?: boolean;
};

type GroupChat = {
  id: string;
  name: string;
  count: number;
  avatar: string;
  tone: Tone;
};

const FRIENDS: Friend[] = [
  { id: 'zoe', name: 'Zoe', avatar: 'Z', account: 'zooe_1007', tone: 'cyan' },
  { id: 'mori', name: '林间晚风', avatar: '林', account: 'mori_011', tone: 'green' },
  { id: 'nana', name: '奶油小熊', avatar: '奶', account: 'nana_77', tone: 'amber' },
  { id: 'qingning', name: '青柠汽水', avatar: '青', account: 'lemon_2024', tone: 'rose' },
  { id: 'shanhai', name: '山海收藏家', avatar: '山', account: 'shanhai_9', tone: 'blue' },
  { id: 'xiaomai', name: '小麦同学', avatar: '麦', account: 'maimai_5', tone: 'violet' },
  { id: 'akira', name: 'Akira', avatar: 'A', account: 'akira_08', tone: 'dark' },
];

const MESSAGE_ROWS: MessageRow[] = [
  {
    id: 'new-friends',
    title: '新朋友',
    avatar: '新',
    tone: 'rose',
    detail: '青柠汽水 关注了你',
    href: '/message/fans',
    arrow: true,
  },
  {
    id: 'activity',
    title: '互动消息',
    avatar: '互',
    tone: 'cyan',
    detail: '奶油小熊 近期访问过你的主页',
    href: '/message/all',
    arrow: true,
  },
  {
    id: 'chat',
    title: 'Zoe',
    avatar: 'Z',
    tone: 'violet',
    detail: '哈哈哈哈哈',
    time: '10-10',
    href: '/message/chat',
    unread: '2',
    online: true,
  },
  {
    id: 'douyin-helper',
    title: '抖音小助手',
    avatar: '抖',
    tone: 'dark',
    detail: '#今天谁请客呢',
    time: '星期四',
    href: '/message/douyin-helper',
    tag: '官方',
    dot: true,
  },
  {
    id: 'system',
    title: '系统通知',
    avatar: '系',
    tone: 'blue',
    detail: '协议修订通知',
    time: '08-31',
    href: '/message/system-notice',
    tag: '官方',
    dot: true,
  },
  {
    id: 'request-update',
    title: '求更新',
    avatar: '更',
    tone: 'green',
    detail: '你收到过1次求更新',
    time: '10-09',
    href: '/me/request-update',
    tag: '官方',
    dot: true,
  },
  {
    id: 'task',
    title: '任务通知',
    avatar: '任',
    tone: 'amber',
    detail: '发作品得流量',
    time: '05-26',
    href: '/message/task-notice',
    tag: '官方',
    dot: true,
  },
  {
    id: 'live',
    title: '直播通知',
    avatar: '直',
    tone: 'rose',
    detail: '举报结果通知',
    time: '05-26',
    href: '/message/live-notice',
    tag: '官方',
    dot: true,
  },
  {
    id: 'money',
    title: '钱包通知',
    avatar: '钱',
    tone: 'cyan',
    detail: '卡券发放提醒',
    time: '05-26',
    href: '/message/money-notice',
    tag: '官方',
    dot: true,
  },
];

const GROUP_CHATS: GroupChat[] = [
  { id: 'music', name: 'AAAAAAAAA、BBBBBBBBBBBBB、CCCCCCCC', count: 3, avatar: 'A', tone: 'cyan' },
  { id: 'city', name: '同城朋友群', count: 12, avatar: '同', tone: 'rose' },
  { id: 'photo', name: '拍照搭子和日常分享', count: 8, avatar: '拍', tone: 'amber' },
  { id: 'school', name: '今天谁请客呢', count: 5, avatar: '请', tone: 'green' },
  { id: 'weekend', name: '周末去哪里', count: 16, avatar: '周', tone: 'violet' },
];

function filterFriends(keyword: string): Friend[] {
  const value = keyword.trim().toLowerCase();
  if (!value) return [];
  return FRIENDS.filter((friend) => {
    return friend.name.toLowerCase().includes(value) || friend.account.toLowerCase().includes(value);
  });
}

function toggleId(ids: string[], id: string): string[] {
  return ids.includes(id) ? ids.filter((item) => item !== id) : [...ids, id];
}

function Avatar({ item, className = '' }: { item: Pick<Friend | GroupChat | MessageRow, 'avatar' | 'tone'>; className?: string }) {
  return <span className={`dyMessageAvatar dyMessageAvatar-${item.tone} ${className}`}>{item.avatar}</span>;
}

export function MessagePage() {
  const [searching, setSearching] = useState(false);
  const [searchKey, setSearchKey] = useState('');
  const [createChatOpen, setCreateChatOpen] = useState(false);
  const [createChatKey, setCreateChatKey] = useState('');
  const [joinedGroupsOpen, setJoinedGroupsOpen] = useState(false);
  const [selectedFriendIds, setSelectedFriendIds] = useState<string[]>([]);

  const searchFriends = useMemo(() => filterFriends(searchKey), [searchKey]);
  const createChatFriends = useMemo(() => filterFriends(createChatKey), [createChatKey]);
  const selectedCount = selectedFriendIds.length;

  const closeSearch = () => {
    setSearching(false);
    setSearchKey('');
  };

  const closeCreateChat = () => {
    setCreateChatOpen(false);
    setCreateChatKey('');
    setJoinedGroupsOpen(false);
  };

  if (searching) {
    return (
      <main className="mobileShell dyMessageReplica dyMessageSearchReplica" aria-label="消息搜索">
        <section className="dyMessageSearchHeader">
          <label className="dyMessageSearchBox">
            <span aria-hidden="true">⌕</span>
            <input
              autoFocus
              value={searchKey}
              placeholder="搜索"
              onChange={(event) => setSearchKey(event.target.value)}
            />
          </label>
          <button type="button" onClick={closeSearch}>取消</button>
        </section>

        <section className="dyMessageSearchContent">
          {searchKey ? (
            <>
              {searchFriends.length ? (
                <header className="dyMessageSubTitle">
                  <span>联系人</span>
                  {searchFriends.length > 3 ? <a href={`/message/more-search?key=${encodeURIComponent(searchKey)}`}>更多联系人 ›</a> : null}
                </header>
              ) : null}

              {searchFriends.slice(0, 3).map((friend) => (
                <a className="dyMessagePeopleResult" href="/message/chat" key={friend.id}>
                  <Avatar item={friend} />
                  <span>
                    <b>{friend.name}</b>
                    <small>抖音号:{friend.account}</small>
                  </span>
                </a>
              ))}

              <a className="dyMessageGotoSearch" href={`/home/search?key=${encodeURIComponent(searchKey)}`}>
                <i aria-hidden="true">⌕</i>
                <span>
                  <b>搜索 <mark>{searchKey}</mark></b>
                  <small>视频、用户、音乐、话题、地点等</small>
                </span>
                <em aria-hidden="true">›</em>
              </a>
            </>
          ) : (
            <>
              <header className="dyMessageSubTitle">更多聊天</header>
              {FRIENDS.slice(0, 3).map((friend) => (
                <a className="dyMessagePeopleResult" href="/message/chat" key={friend.id}>
                  <Avatar item={friend} />
                  <span>
                    <b>{friend.name}</b>
                    <small>抖音号:{friend.account}</small>
                  </span>
                </a>
              ))}
            </>
          )}
        </section>
      </main>
    );
  }

  return (
    <main className={`mobileShell dyMessageReplica ${createChatOpen ? 'isCreateChatOpen' : ''}`}>
      <section className="dyMessageMain">
        <header className="dyMessageHeader" aria-label="消息操作">
          <button className="dyMessageIconButton" type="button" aria-label="创建聊天" onClick={() => setCreateChatOpen(true)}>＋</button>
          <a className="dyMessageIconButton" href="/scan" aria-label="拍摄">◉</a>
          <button className="dyMessageIconButton" type="button" aria-label="搜索" onClick={() => setSearching(true)}>⌕</button>
        </header>

        <section className="dyMessageScroll" aria-label="消息首页">
          <div className="dyMessageFriends" aria-label="朋友">
            {FRIENDS.map((friend, index) => (
              <a className="dyMessageFriend" href="/message/chat" key={friend.id}>
                <span className={`dyMessageFriendAvatar ${index % 2 === 0 ? 'isOnline' : ''}`}>
                  <Avatar item={friend} />
                </span>
                <span>{friend.name}</span>
              </a>
            ))}
            <button className="dyMessageFriend" type="button" onClick={() => setCreateChatOpen(true)}>
              <span className="dyMessageFriendAvatar dyMessageStatusAvatar">设</span>
              <span>状态设置</span>
            </button>
          </div>

          <div className="dyMessageRows" aria-label="消息列表">
            {MESSAGE_ROWS.map((row) => (
              <a className="dyMessageRow" href={row.href} key={row.id}>
                <span className={`dyMessageRowAvatar ${row.online ? 'isOnline' : ''}`}>
                  <Avatar item={row} />
                </span>
                <span className="dyMessageRowContent">
                  <span className="dyMessageRowLeft">
                    <span className="dyMessageRowName">
                      <b>{row.title}</b>
                      {row.tag ? <em>{row.tag}</em> : null}
                    </span>
                    <span className="dyMessageRowDetail">
                      <span>{row.detail}</span>
                      {row.time ? (
                        <>
                          <i aria-hidden="true" />
                          <time>{row.time}</time>
                        </>
                      ) : null}
                    </span>
                  </span>
                  <span className="dyMessageRowRight">
                    {row.arrow ? <span className="dyMessageArrow" aria-hidden="true">›</span> : null}
                    {row.unread ? <span className="dyMessageBadge">{row.unread}</span> : null}
                    {row.dot ? <span className="dyMessageUnreadDot" aria-label="未读" /> : null}
                  </span>
                </span>
              </a>
            ))}
            <p className="dyMessageNoMore">暂时没有更多了</p>
          </div>
        </section>

        <DouyinTabBar active="message" />
      </section>

      {createChatOpen ? (
        <section className="dyMessageCreateLayer" aria-label="创建聊天">
          <button className="dyMessageCreateMask" type="button" aria-label="关闭创建聊天" onClick={closeCreateChat} />
          <article className="dyMessageCreateSheet">
            {joinedGroupsOpen ? (
              <>
                <header className="dyMessageJoinedNav">
                  <button type="button" aria-label="返回创建聊天" onClick={() => setJoinedGroupsOpen(false)}>‹</button>
                  <b>已加入的群聊</b>
                  <span />
                </header>
                <div className="dyMessageSheetScroll">
                  {GROUP_CHATS.map((group) => (
                    <a className="dyMessageGroupRow" href="/message/chat" key={group.id}>
                      <Avatar item={group} />
                      <span>
                        <b>{group.name.length > 20 ? `${group.name.slice(0, 20)}...` : group.name}</b>
                        <small>({group.count})</small>
                      </span>
                      <em aria-hidden="true">›</em>
                    </a>
                  ))}
                  <p className="dyMessageNoMore">暂时没有更多了</p>
                </div>
              </>
            ) : (
              <>
                <div className="dyMessageCreateSearch">
                  <label className="dyMessageSearchBox">
                    <span aria-hidden="true">⌕</span>
                    <input
                      value={createChatKey}
                      placeholder="搜索用户"
                      onChange={(event) => setCreateChatKey(event.target.value)}
                    />
                  </label>
                  {createChatKey ? <button type="button" onClick={() => setCreateChatKey('')}>取消</button> : null}
                </div>

                <div className="dyMessageSheetScroll">
                  {createChatKey ? (
                    createChatFriends.length ? (
                      <div className="dyMessageCreateResults">
                        {createChatFriends.map((friend) => {
                          const selected = selectedFriendIds.includes(friend.id);
                          return (
                            <button
                              className="dyMessageCreateResult"
                              type="button"
                              key={friend.id}
                              onClick={() => setSelectedFriendIds((ids) => toggleId(ids, friend.id))}
                            >
                              <Avatar item={friend} />
                              <span>
                                <b>{friend.name}</b>
                                <small>抖音号:{friend.account}</small>
                              </span>
                              <i className={`dyMessageCheck ${selected ? 'isChecked' : ''}`} aria-hidden="true" />
                            </button>
                          );
                        })}
                      </div>
                    ) : (
                      <div className="dyMessageNoResult">
                        <b>搜索结果为空</b>
                        <small>没有搜索到相关的联系人</small>
                      </div>
                    )
                  ) : (
                    <>
                      <button className="dyMessageJoinedEntry" type="button" onClick={() => setJoinedGroupsOpen(true)}>
                        <span aria-hidden="true">群</span>
                        <b>已加入的群聊</b>
                        <em aria-hidden="true">›</em>
                      </button>

                      <section className="dyMessageFriendList" aria-label="联系人">
                        <h2>Z</h2>
                        {FRIENDS.map((friend) => {
                          const selected = selectedFriendIds.includes(friend.id);
                          return (
                            <button
                              className="dyMessageCreateFriend"
                              type="button"
                              key={friend.id}
                              aria-pressed={selected}
                              onClick={() => setSelectedFriendIds((ids) => toggleId(ids, friend.id))}
                            >
                              <Avatar item={friend} />
                              <span>{friend.name}</span>
                              <i className={`dyMessageCheck ${selected ? 'isChecked' : ''}`} aria-hidden="true" />
                            </button>
                          );
                        })}
                      </section>
                    </>
                  )}
                </div>

                <footer className="dyMessageCreateFooter">
                  <button type="button" disabled={!selectedCount}>
                    发起群聊{selectedCount ? `(${selectedCount})` : ''}
                  </button>
                </footer>
              </>
            )}
          </article>
        </section>
      ) : null}
    </main>
  );
}
