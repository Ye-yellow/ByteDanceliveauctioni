import { useMemo, useState } from 'react';

type PeopleMode = 'fans' | 'visitors';

type PeopleItem = {
  id: string;
  name: string;
  avatar: string;
  desc: string;
  relation: string;
};

const PEOPLE: PeopleItem[] = [
  { id: 'p1', name: '青柠汽水', avatar: '青', desc: '常看直播拍卖和好物分享', relation: '已关注' },
  { id: 'p2', name: '山海收藏家', avatar: '山', desc: '最近访问过你的主页', relation: '回关' },
  { id: 'p3', name: '奶油小熊', avatar: '奶', desc: '和你有 3 个共同关注', relation: '关注' },
  { id: 'p4', name: '林间晚风', avatar: '林', desc: '喜欢同城直播和生活记录', relation: '关注' },
];

function PeopleRows({ items, mode }: { items: PeopleItem[]; mode: PeopleMode }) {
  return (
    <div className="dyMsgPeopleRows">
      {items.map((person) => (
        <article className="dyMsgPeopleRow" key={person.id}>
          <span className={`dyMsgAvatar ${mode === 'fans' ? 'dyMsgAvatar-rose' : 'dyMsgAvatar-cyan'}`}>{person.avatar}</span>
          <div>
            <b>{person.name}</b>
            <small>{mode === 'fans' ? person.desc : '近期访问过你的主页'}</small>
          </div>
          <button type="button">{mode === 'fans' ? person.relation : '查看'}</button>
        </article>
      ))}
    </div>
  );
}

export function MessagePeoplePage({ initialMode = 'fans' }: { initialMode?: PeopleMode }) {
  const [mode, setMode] = useState<PeopleMode>(initialMode);
  const [visitorEnabled, setVisitorEnabled] = useState(false);
  const [settingOpen, setSettingOpen] = useState(false);
  const fans = useMemo(() => PEOPLE.slice(0, 2), []);
  const recommend = useMemo(() => PEOPLE.slice(2), []);

  return (
    <main className="dyMsgPage dyMsgPeoplePage" aria-label={mode === 'fans' ? '粉丝' : '主页访客'}>
      <header className="dyMsgHeader">
        <button className="dyMsgHeaderIcon" type="button" aria-label="返回" onClick={() => history.back()}>
          ‹
        </button>
        <div className="dyMsgPeopleTabs" role="tablist" aria-label="消息人群">
          <button className={mode === 'fans' ? 'active' : ''} type="button" onClick={() => setMode('fans')}>粉丝</button>
          <button className={mode === 'visitors' ? 'active' : ''} type="button" onClick={() => setMode('visitors')}>主页访客</button>
        </div>
        {mode === 'visitors' ? (
          <button className="dyMsgHeaderText" type="button" onClick={() => setSettingOpen(true)}>设置</button>
        ) : (
          <span />
        )}
      </header>

      {mode === 'fans' ? (
        <section className="dyMsgPeopleContent">
          <PeopleRows items={fans} mode="fans" />
          <h2>朋友推荐 <small>ⓘ</small></h2>
          <PeopleRows items={recommend} mode="fans" />
          <p className="dyMsgNoMore">暂时没有更多了</p>
        </section>
      ) : (
        <section className="dyMsgPeopleContent">
          {visitorEnabled ? (
            <>
              <PeopleRows items={PEOPLE} mode="visitors" />
              <p className="dyMsgNoMore">暂时没有更多了</p>
            </>
          ) : (
            <article className="dyMsgVisitorAuth">
              <div className="dyMsgVisitorAvatars">
                <span className="dyMsgAvatar dyMsgAvatar-cyan">访</span>
                <span className="dyMsgAvatar dyMsgAvatar-rose">我</span>
                <span className="dyMsgAvatar dyMsgAvatar-amber">客</span>
              </div>
              <h1>查看新访客需要你的授权</h1>
              <ul>
                <li>访客记录中仅展示同样已授权的用户</li>
                <li>开启后，你访问他人主页也会留下记录</li>
                <li>你可以随时在访客设置中关闭授权</li>
              </ul>
              <div>
                <button type="button" onClick={() => history.back()}>保持关闭</button>
                <button type="button" onClick={() => setVisitorEnabled(true)}>开启访客</button>
              </div>
            </article>
          )}
        </section>
      )}

      {settingOpen ? (
        <section className="dyMsgBottomSheet" aria-label="主页访客设置">
          <button className="dyMsgPacketMask" type="button" aria-label="关闭设置" onClick={() => setSettingOpen(false)} />
          <article className="dyMsgVisitorSetting">
            <button type="button" aria-label="关闭" onClick={() => setSettingOpen(false)}>×</button>
            <span className="dyMsgAvatar dyMsgAvatar-cyan">访</span>
            <h2>主页访客</h2>
            <p>关闭后，你查看他人主页时不会留下记录；同时，你也无法查看谁访问了你的主页。</p>
            <label>
              <span>展示主页访客</span>
              <input type="checkbox" checked={visitorEnabled} onChange={(event) => setVisitorEnabled(event.target.checked)} />
            </label>
          </article>
        </section>
      ) : null}
    </main>
  );
}

export function MessageFansPage() {
  return <MessagePeoplePage initialMode="fans" />;
}

export function MessageVisitorsPage() {
  return <MessagePeoplePage initialMode="visitors" />;
}
