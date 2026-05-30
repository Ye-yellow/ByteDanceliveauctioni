import { useMemo, useState, type ReactNode } from 'react';
import { useAuthSession } from '../shared/auth/useAuthSession';
import './profile-utility-replica.css';

type SettingGroup = {
  title?: string;
  rows: Array<{ label: string; icon: string; href?: string; desc?: string }>;
};

const FRIENDS = [
  { name: '青柠汽水', time: '今天 15:20' },
  { name: '山海收藏家', time: '昨天' },
  { name: '奶油小熊', time: '3天前' },
  { name: '林间晚风', time: '05-24' },
  { name: 'LiveAuction 小助手', time: '05-20' },
];

const POSTERS = [
  { title: '服了，这个现场反应也太真实了', tag: '浏览记录', tone: 'rose' },
  { title: '今晚 8 点严选拍品开箱', tag: '直播回放', tone: 'cyan' },
  { title: '把雨拍成电影，慢一点也很好看', tag: '视频', tone: 'amber' },
  { title: '拍品准备、讲解卡和成交节奏', tag: '竞拍', tone: 'violet' },
  { title: '同城直播开场', tag: '直播', tone: 'dark' },
  { title: '收藏的短视频合集', tag: '收藏', tone: 'rose' },
];

const SETTINGS: SettingGroup[] = [
  {
    title: '帐号',
    rows: [
      { label: '帐号与安全', icon: '人' },
      { label: '隐私设置', icon: '锁' },
    ],
  },
  {
    title: '通用',
    rows: [
      { label: '通知设置', icon: '铃', href: '/message/notice-setting' },
      { label: '动态壁纸', icon: '图' },
      { label: '通用设置', icon: '设' },
    ],
  },
  {
    title: '帐号互通',
    rows: [{ label: '头条主页', icon: '头' }],
  },
  {
    title: '关于',
    rows: [
      { label: '广告反馈与设置', icon: '广' },
      { label: '用户协议', icon: '协', href: '/service-protocol' },
      { label: '社区自律公约', icon: '约' },
      { label: '隐私政策', icon: '隐', href: '/service-protocol' },
      { label: '应用权限', icon: '权' },
      { label: '第三方SDK列表', icon: 'S' },
      { label: '关于抖音', icon: '抖' },
      { label: '反馈与帮助', icon: '问' },
      { label: '清理占用空间', icon: '清', desc: '268.4MB' },
    ],
  },
  {
    rows: [
      { label: '切换空间', icon: '切' },
      { label: '退出登录', icon: '退' },
    ],
  },
];

function goBack(fallback = '/me') {
  if (window.history.length > 1) {
    window.history.back();
    return;
  }
  window.location.assign(fallback);
}

function Header({ title, right, fallback = '/me' }: { title?: string; right?: ReactNode; fallback?: string }) {
  return (
    <header className="dyProfileUtilityHeader">
      <button type="button" aria-label="返回" onClick={() => goBack(fallback)}>‹</button>
      <h1>{title || ''}</h1>
      <span>{right}</span>
    </header>
  );
}

function Avatar({ name, large = false }: { name: string; large?: boolean }) {
  return <i className={large ? 'dyProfileUtilityAvatar large' : 'dyProfileUtilityAvatar'}>{name.slice(0, 1)}</i>;
}

function FriendRows() {
  return (
    <section className="dyProfileUtilityFriendRows">
      {FRIENDS.map((friend) => (
        <a href="/message/chat" key={friend.name}>
          <span><Avatar name={friend.name} /><b>{friend.name}</b></span>
          <time>{friend.time}</time>
        </a>
      ))}
    </section>
  );
}

function PosterGrid({ empty = false }: { empty?: boolean }) {
  if (empty) {
    return (
      <section className="dyProfileUtilityEmpty">
        <div />
        <b>暂无观影历史记录</b>
      </section>
    );
  }
  return (
    <section className="dyProfileUtilityPosterGrid">
      {POSTERS.map((poster, index) => (
        <a href="/video-detail" className={`tone-${poster.tone}`} key={`${poster.title}-${index}`}>
          <span>▶</span>
          <b>{poster.title}</b>
          <small>{poster.tag}</small>
        </a>
      ))}
    </section>
  );
}

function BottomSheet({ open, onClose, children }: { open: boolean; onClose: () => void; children: ReactNode }) {
  if (!open) return null;
  return (
    <>
      <button type="button" className="dyProfileUtilityMask" aria-label="关闭弹层" onClick={onClose} />
      <section className="dyProfileUtilitySheet" role="dialog" aria-modal="true">{children}</section>
    </>
  );
}

function SettingRow({ row }: { row: SettingGroup['rows'][number] }) {
  const inner = (
    <>
      <span><i>{row.icon}</i><b>{row.label}</b></span>
      <em>{row.desc || '›'}</em>
    </>
  );
  if (row.href) return <a href={row.href}>{inner}</a>;
  return <button type="button">{inner}</button>;
}

export function MyCardPage() {
  const { user } = useAuthSession();
  const [sharing, setSharing] = useState(false);
  const name = user?.nickname || user?.username || 'ZZZZZZZZZZ';
  const qrCells = useMemo(() => Array.from({ length: 49 }, (_, index) => index), []);

  return (
    <main className="mobileShell dyMyCardPage">
      <Header fallback="/me" right={<button type="button" onClick={() => setSharing(true)}>↗</button>} />
      <section className="dyMyCardContent">
        <div className="dyMyCardQr" aria-label="抖音码">
          {qrCells.map((cell) => <i className={(cell + cell % 5) % 3 === 0 ? 'filled' : ''} key={cell} />)}
          <Avatar name={name} large />
        </div>
        <b>{name}</b>
        <p>抖音扫一扫，立即关注我</p>
      </section>
      <footer className="dyMyCardFooter">
        <a href="/scan"><span>▣</span><b>扫一扫</b></a>
        <button type="button"><span>↓</span><b>保存</b></button>
      </footer>
      <BottomSheet open={sharing} onClose={() => setSharing(false)}>
        <header><b>分享到</b></header>
        <div className="dyProfileUtilityShareGrid">
          {['微信', '朋友圈', 'QQ', '微博', '复制链接', '私信朋友'].map((item) => (
            <a href={item === '私信朋友' ? '/message/share-to-friend' : '/me/my-card'} key={item}><span>{item.slice(0, 1)}</span><b>{item}</b></a>
          ))}
        </div>
        <button type="button" onClick={() => setSharing(false)}>取消</button>
      </BottomSheet>
    </main>
  );
}

export function RequestUpdatePage() {
  const [sheet, setSheet] = useState(false);
  const [enabled, setEnabled] = useState(true);
  return (
    <main className="mobileShell dyProfileUtilityPage dark dyRequestUpdatePage">
      <Header title="求更新" fallback="/me" right={<button type="button" onClick={() => setSheet(true)}>⋯</button>} />
      <section className="dyRequestUpdateContent">
        <p>历史求更新（粉丝送礼后7天未开播则退还礼物）</p>
        <FriendRows />
        <small>最多展示100位粉丝的历史求更新记录</small>
      </section>
      <footer className="dyRequestUpdateFooter">
        <a href="/publish">发布作品</a>
        <button type="button">去直播</button>
      </footer>
      <BottomSheet open={sheet} onClose={() => setSheet(false)}>
        <button type="button" onClick={() => { setEnabled((value) => !value); setSheet(false); }}>{enabled ? '关闭' : '开启'}求更新提醒</button>
        <a href="/me/my-request-update">我的求更新提醒</a>
        <i />
        <button type="button" onClick={() => setSheet(false)}>取消</button>
      </BottomSheet>
    </main>
  );
}

export function MyRequestUpdatePage() {
  return (
    <main className="mobileShell dyProfileUtilityPage dark">
      <Header title="我的求更新记录" fallback="/me/request-update" />
      <section className="dyRequestUpdateContent"><FriendRows /></section>
    </main>
  );
}

export function ProfileSettingReplicaPage() {
  const { user, logout } = useAuthSession();
  const display = user?.nickname || user?.username || '未登录';
  return (
    <main className="mobileShell dyProfileUtilityPage dark">
      <Header title="设置" fallback="/me" />
      <section className="dySettingReplicaContent">
        {SETTINGS.map((group, index) => (
          <section className="dySettingReplicaGroup" key={group.title || `group-${index}`}>
            {group.title ? <h2>{group.title}</h2> : null}
            {group.rows.map((row) => row.label === '退出登录' ? (
              <button type="button" onClick={() => user ? void logout() : window.location.assign('/login')} key={row.label}>
                <span><i>{row.icon}</i><b>{user ? row.label : '登录账号'}</b></span>
                <em>{display}</em>
              </button>
            ) : <SettingRow row={row} key={row.label} />)}
          </section>
        ))}
        <p>抖音 React H5</p>
      </section>
    </main>
  );
}

export function LookHistoryPage({ title = '观看历史', collect = false }: { title?: string; collect?: boolean }) {
  const [tab, setTab] = useState(0);
  const [cleared, setCleared] = useState(false);
  return (
    <main className="mobileShell dyProfileUtilityPage dark">
      <Header title={title} fallback="/me" right={!cleared && !collect ? <button type="button" onClick={() => setCleared(true)}>清空</button> : null} />
      <nav className="dyHistoryReplicaTabs" aria-label={title}>
        {['视频', collect ? '直播' : '影视综'].map((item, index) => (
          <button type="button" className={tab === index ? 'active' : ''} onClick={() => setTab(index)} key={item}>{item}</button>
        ))}
      </nav>
      {cleared || tab === 1 ? <PosterGrid empty /> : <PosterGrid />}
    </main>
  );
}

export function DeclareSchoolPage() {
  return (
    <main className="mobileShell dyProfileUtilityPage light">
      <Header title="声明学校" fallback="/me/add-school" />
      <section className="dyDeclareSchoolPanel">
        <h2>声明学校信息</h2>
        <p>选择学校和院系后，可在主页展示校友身份。</p>
        <a href="/me/choose-school"><b>学校名称</b><span>请选择 ›</span></a>
        <a href="/me/choose-department"><b>院系信息</b><span>请选择 ›</span></a>
        <button type="button">提交认证</button>
      </section>
    </main>
  );
}

export function MinorProtectionPage() {
  return (
    <main className="mobileShell dyProfileUtilityPage dark">
      <Header title="未成年保护工具" fallback="/me/right-menu/setting" />
      <section className="dyMinorRows">
        {[
          ['时间锁', '0'],
          ['青少年模式', '1'],
          ['亲子平台', '2'],
        ].map(([label, type]) => (
          <a href={`/me/right-menu/minor-protection/detail-setting?type=${type}`} key={label}>
            <b>{label}</b><span>未开启 ›</span>
          </a>
        ))}
      </section>
    </main>
  );
}

export function MinorProtectionDetailPage() {
  const type = new URLSearchParams(location.search).get('type') || '0';
  const triggerTime = localStorage.getItem('changeTriggerTime') || '60';

  if (type === '1' || type === '2') {
    const youth = type === '1';
    return (
      <main className="mobileShell dyProfileUtilityPage dark">
        <Header fallback="/me/right-menu/minor-protection/index" />
        <section className={youth ? 'dyMinorHero youth' : 'dyMinorHero parent'}>
          <span>{youth ? '青' : '亲'}</span>
          <h1>{youth ? '青少年模式' : '亲子平台'}</h1>
          <p>{youth ? '精选适合青少年的内容，限制使用时长和消费能力。' : '绑定亲子关系，帮助家长了解和守护未成年使用体验。'}</p>
        </section>
        <footer className="dyMinorFooter">
          <p>{youth ? '更多信息可阅读《儿童/青少年使用须知》' : '我已阅读并接受《抖音亲子平台服务协议》'}</p>
          <button type="button">{youth ? '开启青少年模式' : '立即绑定'}</button>
        </footer>
      </main>
    );
  }

  return (
    <main className="mobileShell dyProfileUtilityPage dark">
      <Header fallback="/me/right-menu/minor-protection/index" />
      <section className="dyTimeLockNotice"><span>锁</span><b>时间锁已关闭</b></section>
      <section className="dyMinorRows muted">
        <p>可为时间锁设置一个触发时间</p>
        <p>开启时间锁后，单日使用时长超过触发时间，需输入密码才能继续使用</p>
        <p>开启时间锁，需先设置独立密码；忘记密码后可通过申诉重置密码</p>
        <a href={`/me/right-menu/minor-protection/trigger-time?triggerTime=${triggerTime}`}><b>触发时间</b><span>{triggerTime}分钟 ›</span></a>
      </section>
      <footer className="dyMinorFooter"><button type="button">开启时间锁</button></footer>
    </main>
  );
}

export function TriggerTimePage() {
  const initial = new URLSearchParams(location.search).get('triggerTime') || localStorage.getItem('changeTriggerTime') || '60';
  const [selected, setSelected] = useState(initial);
  function select(value: string) {
    setSelected(value);
    localStorage.setItem('changeTriggerTime', value);
  }
  return (
    <main className="mobileShell dyProfileUtilityPage dark">
      <Header title="触发时间" fallback="/me/right-menu/minor-protection/detail-setting?type=0" />
      <section className="dyMinorRows">
        {['40', '60', '90', '120'].map((value) => (
          <button type="button" onClick={() => select(value)} key={value}>
            <b>{value}分钟</b><span>{selected === value ? '✓' : ''}</span>
          </button>
        ))}
      </section>
    </main>
  );
}
