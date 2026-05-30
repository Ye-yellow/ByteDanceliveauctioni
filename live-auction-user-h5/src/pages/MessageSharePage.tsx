import { useMemo, useState } from 'react';

type ShareFriend = {
  id: string;
  name: string;
  account: string;
  avatar: string;
  group: string;
};

const SHARE_FRIENDS: ShareFriend[] = [
  { id: 'recent-1', name: '青柠汽水', account: 'qingning77', avatar: '青', group: '最近聊天' },
  { id: 'recent-2', name: '山海收藏家', account: 'auctionsea', avatar: '山', group: '最近聊天' },
  { id: 'both-1', name: '奶油小熊', account: 'bearcream', avatar: '奶', group: '互关好友' },
  { id: 'both-2', name: '林间晚风', account: 'linwind', avatar: '林', group: '互关好友' },
  { id: 'a-1', name: 'Aki', account: 'aki_05', avatar: 'A', group: 'A' },
  { id: 'c-1', name: '陈小满', account: 'chenxiaoman', avatar: '陈', group: 'C' },
  { id: 'l-1', name: 'LiveAuction 小助手', account: 'liveauction', avatar: '拍', group: 'L' },
  { id: 'y-1', name: 'Yexieer', account: 'yexieer', avatar: 'Y', group: 'Y' },
  { id: 'z-1', name: 'zzzz', account: 'zzzz2026', avatar: 'Z', group: 'Z' },
];

const INDEXES = ['#', 'A', 'C', 'L', 'Y', 'Z'];

function highlight(value: string, keyword: string) {
  if (!keyword) return value;
  const index = value.toLowerCase().indexOf(keyword.toLowerCase());
  if (index < 0) return value;
  return (
    <>
      {value.slice(0, index)}
      <mark>{value.slice(index, index + keyword.length)}</mark>
      {value.slice(index + keyword.length)}
    </>
  );
}

export function MessageSharePage() {
  const [query, setQuery] = useState('');
  const [selectedIds, setSelectedIds] = useState<string[]>([]);
  const [createGroup, setCreateGroup] = useState(false);

  const selectedFriends = useMemo(
    () => SHARE_FRIENDS.filter((friend) => selectedIds.includes(friend.id)),
    [selectedIds],
  );

  const searchResults = useMemo(() => {
    const keyword = query.trim().toLowerCase();
    if (!keyword) return [];
    return SHARE_FRIENDS.filter((friend) => friend.name.toLowerCase().includes(keyword) || friend.account.toLowerCase().includes(keyword));
  }, [query]);

  const groupedFriends = useMemo(() => {
    return SHARE_FRIENDS.reduce<Record<string, ShareFriend[]>>((groups, friend) => {
      const key = friend.group;
      groups[key] = groups[key] ? [...groups[key], friend] : [friend];
      return groups;
    }, {});
  }, []);

  function toggleFriend(friend: ShareFriend) {
    setSelectedIds((value) => (value.includes(friend.id) ? value.filter((id) => id !== friend.id) : [...value, friend.id]));
  }

  function jumpToIndex(index: string) {
    const target = document.getElementById(`dy-share-index-${index === '#' ? 'top' : index}`);
    target?.scrollIntoView({ block: 'start' });
  }

  return (
    <main className="dyMsgPage dyMsgSharePage" aria-label="私信分享">
      <header className="dyMsgShareHeader">
        <button type="button" aria-label="关闭" onClick={() => history.back()}>×</button>
        <b>私信给</b>
        <span />
        <label className="dyMsgShareSearch">
          {selectedFriends.length ? (
            <span className="dyMsgShareSelected">
              {selectedFriends.map((friend) => (
                <i className="dyMsgAvatar" key={friend.id} onClick={() => toggleFriend(friend)}>{friend.avatar}</i>
              ))}
            </span>
          ) : (
            <i aria-hidden="true">⌕</i>
          )}
          <input value={query} placeholder="搜索" onChange={(event) => setQuery(event.target.value)} />
          {query ? <button type="button" aria-label="清空" onClick={() => setQuery('')}>×</button> : null}
        </label>
      </header>

      {query ? (
        <section className="dyMsgShareSearchPane" aria-label="搜索结果">
          {searchResults.length ? (
            searchResults.map((friend) => (
              <button className="dyMsgShareFriendRow" type="button" key={friend.id} onClick={() => toggleFriend(friend)}>
                <span className={selectedIds.includes(friend.id) ? 'dyMsgCheck isChecked' : 'dyMsgCheck'} />
                <i className="dyMsgAvatar dyMsgAvatar-cyan">{friend.avatar}</i>
                <span>
                  <b>{highlight(friend.name, query)}</b>
                  <small>抖音号：{highlight(friend.account, query)}</small>
                </span>
              </button>
            ))
          ) : (
            <div className="dyMsgNoResult">
              <span>⌕</span>
              <b>搜索结果为空</b>
              <small>没有搜索到相关的联系人</small>
            </div>
          )}
        </section>
      ) : null}

      <section className="dyMsgShareContent" aria-label="联系人列表" id="dy-share-index-top">
        <a className="dyMsgShareGroupLink" href="/message/joined-group-chat">
          <span>已加入的群聊</span>
          <b>›</b>
        </a>
        {Object.entries(groupedFriends).map(([group, friends]) => (
          <section className="dyMsgShareGroup" key={group} id={`dy-share-index-${group === '最近聊天' ? 'top' : group}`}>
            <h2>{group}</h2>
            {friends.map((friend) => (
              <button className="dyMsgShareFriendRow" type="button" key={friend.id} onClick={() => toggleFriend(friend)}>
                <span className={selectedIds.includes(friend.id) ? 'dyMsgCheck isChecked' : 'dyMsgCheck'} />
                <i className="dyMsgAvatar dyMsgAvatar-rose">{friend.avatar}</i>
                <span>
                  <b>{friend.name}</b>
                  <small>抖音号：{friend.account}</small>
                </span>
              </button>
            ))}
          </section>
        ))}
      </section>

      <nav className="dyMsgShareIndex" aria-label="联系人索引">
        {INDEXES.map((index) => (
          <button type="button" key={index} onClick={() => jumpToIndex(index)}>{index}</button>
        ))}
      </nav>

      {selectedFriends.length ? (
        <aside className="dyMsgShareSendBar" aria-label="发送面板">
          <textarea placeholder="有什么想和好友说的..." />
          <span className="dyMsgPoster dyMsgPoster-rose" />
          <label>
            <input type="checkbox" checked={createGroup} onChange={(event) => setCreateGroup(event.target.checked)} />
            创建群聊
          </label>
          <button type="button">{selectedFriends.length > 1 && !createGroup ? '分别发送' : '发送'}</button>
        </aside>
      ) : null}
    </main>
  );
}
