import { type ReactNode } from 'react';
import { useAuthSession } from '../shared/auth/useAuthSession';

type Row = {
  title: string;
  desc?: string;
  href?: string;
};

const FRIENDS = ['青柠汽水', '山海收藏家', '奶油小熊', '林间晚风', 'LiveAuction 小助手', 'Yexieer'];
const MUSIC_POSTERS = ['龙卷风', '爱在西元前', '蜗牛', '半岛铁盒', '七里香', '发如雪'];
const NOTICE_COPY = {
  helper: ['#今天谁请客呢', '创作灵感', '抖音小助手推荐你参与热门话题。'],
  system: ['协议修订通知', '账号安全提醒', '服务协议与隐私政策已更新。'],
  task: ['发作品得流量', '完善主页', '连续发布优质作品，有机会获得更多曝光。'],
  live: ['直播预约提醒', '举报结果通知', '你关注的严选专场即将开播。'],
  money: ['卡券发放提醒', '保证金提醒', '交易相关资金与订单请以 LiveAuction 订单页为准。'],
};

function back(fallback = '/home') {
  if (window.history.length > 1) {
    window.history.back();
    return;
  }
  window.location.assign(fallback);
}

function Header({ title, fallback = '/home', right }: { title: string; fallback?: string; right?: ReactNode }) {
  return (
    <header className="dyUtilityHeader">
      <button type="button" aria-label="返回" onClick={() => back(fallback)}>‹</button>
      <h1>{title}</h1>
      <span>{right}</span>
    </header>
  );
}

function Avatar({ name }: { name: string }) {
  return <i className="dyUtilityAvatar">{name.slice(0, 1)}</i>;
}

function Rows({ rows }: { rows: Row[] }) {
  return (
    <section className="dyUtilityRows">
      {rows.map((row) => {
        const inner = (
          <>
            <span><b>{row.title}</b>{row.desc ? <small>{row.desc}</small> : null}</span>
            <i>›</i>
          </>
        );
        return row.href ? <a href={row.href} key={row.title}>{inner}</a> : <button type="button" key={row.title}>{inner}</button>;
      })}
    </section>
  );
}

export function AlbumDetailPage() {
  return (
    <main className="mobileShell dyUtilityDarkPage">
      <Header title="合集" fallback="/video-detail" />
      <section className="dyUtilityAlbumHero"><b>LiveAuction 合集</b><p>拍品开箱、直播片段、成交晒单</p></section>
      <section className="dyUtilityPosterGrid">
        {MUSIC_POSTERS.map((item) => <a href="/video-detail" key={item}><span>{item.slice(0, 1)}</span><b>{item}</b><small>视频合集</small></a>)}
      </section>
    </main>
  );
}

export function ServiceProtocolPage() {
  return (
    <main className="mobileShell dyUtilityLightPage">
      <Header title="服务协议" fallback="/me" />
      <article className="dyUtilityArticle">
        <h1>服务协议与隐私政策</h1>
        <p>本页面承接抖音参考项目的信息架构，用于展示服务说明、账号规则、内容规范和隐私保护条款。</p>
        <p>LiveAuction 竞拍、订单和支付以本项目真实后端返回的数据为准。</p>
      </article>
    </main>
  );
}

export function NoticePage({ title, group }: { title: string; group: keyof typeof NOTICE_COPY }) {
  const [first, second, desc] = NOTICE_COPY[group];
  return (
    <main className="mobileShell dyUtilityDarkPage">
      <Header title={title} fallback="/message" right={<a href="/message/notice-setting">设置</a>} />
      <section className="dyUtilityNoticeList">
        {[first, second].map((item, index) => (
          <a href={group === 'money' ? '/shop/orders?from=money-notice' : '/message'} key={item}>
            <header><b>{item}</b>{index === 0 ? <em>官方</em> : null}</header>
            <p>{desc}</p>
            <time>今天 10:28</time>
          </a>
        ))}
      </section>
    </main>
  );
}

export function NoticeSettingPage() {
  return (
    <main className="mobileShell dyUtilityDarkPage">
      <Header title="通知设置" fallback="/message" />
      <Rows rows={[
        { title: '互动消息', desc: '赞、评论、@我的' },
        { title: '直播通知', desc: '关注主播开播提醒' },
        { title: '商城消息', desc: '订单、物流、售后' },
        { title: '系统通知', desc: '协议、安全和账号提醒' },
      ]} />
    </main>
  );
}

export function ChatDetailPage() {
  return (
    <main className="mobileShell dyUtilityDarkPage">
      <Header title="聊天详情" fallback="/message/chat" />
      <section className="dyUtilityProfileCard"><Avatar name="zzzz" /><b>zzzz</b><span>抖音号：dy2026</span></section>
      <Rows rows={[{ title: '查找聊天内容' }, { title: '消息免打扰', desc: '未开启' }, { title: '置顶聊天', desc: '未开启' }, { title: '设置备注名', href: '/set-remark' }]} />
    </main>
  );
}

export function RedPacketPage() {
  return (
    <main className="mobileShell dyUtilityDarkPage">
      <Header title="红包详情" fallback="/message/chat" />
      <section className="dyUtilityRedPacket"><Avatar name="z" /><h1>0.01元</h1><p>zzzz 的红包 · 大吉大利</p><a href="/message/chat">返回聊天</a></section>
    </main>
  );
}

export function MoreSearchPage() {
  return (
    <main className="mobileShell dyUtilityDarkPage">
      <header className="dyUtilitySearchHeader"><button type="button" onClick={() => back('/message')}>‹</button><label><span>⌕</span><input autoFocus placeholder="搜索联系人或群聊" /></label><button type="button">取消</button></header>
      <section className="dyUtilityPeopleList">{FRIENDS.map((name) => <a href="/message/chat" key={name}><Avatar name={name} /><span><b>{name}</b><small>抖音号：dy_{name.length}2026</small></span></a>)}</section>
    </main>
  );
}

export function JoinedGroupChatPage() {
  return <PeopleListPage title="已加入的群聊" desc="群聊成员在线" />;
}

export function PeopleListPage({ title = '发现朋友', desc = '可能认识的人' }: { title?: string; desc?: string }) {
  return (
    <main className="mobileShell dyUtilityDarkPage">
      <Header title={title} fallback="/message" />
      <section className="dyUtilityPeopleList">{FRIENDS.map((name) => <a href="/message/chat" key={name}><Avatar name={name} /><span><b>{name}</b><small>{desc}</small></span><button type="button">关注</button></a>)}</section>
    </main>
  );
}

export function ScanPage() {
  return (
    <main className="mobileShell dyUtilityScanPage">
      <Header title="扫一扫" />
      <section><span /><b>将二维码放入框内</b><p>自动扫描，支持抖音码和 LiveAuction 链接</p></section>
    </main>
  );
}

export function RemarkPage() {
  return (
    <main className="mobileShell dyUtilityDarkPage">
      <Header title="设置备注名" fallback="/message/chat/detail" />
      <label className="dyUtilityInput"><input autoFocus placeholder="填写备注名" /></label>
    </main>
  );
}

export function ProfileEditPage() {
  const { user } = useAuthSession();
  const name = user?.nickname || user?.username || '点击设置';
  return (
    <main className="mobileShell dyUtilityLightPage">
      <Header title="编辑资料" fallback="/me" right="已完成85%" />
      <section className="dyUtilityEditAvatar"><Avatar name={name} /><span>点击更换头像</span></section>
      <Rows rows={[
        { title: '名字', desc: name, href: '/me/edit-userinfo-item?type=name' },
        { title: '抖音号', desc: '点击设置', href: '/me/edit-userinfo-item?type=id' },
        { title: '简介', desc: '介绍你的拍品偏好', href: '/me/edit-userinfo-item?type=bio' },
        { title: '性别', desc: '不展示' },
        { title: '生日', desc: '点击设置' },
        { title: '所在地', desc: '深圳', href: '/me/choose-location' },
        { title: '学校', desc: '点击设置', href: '/me/add-school' },
      ]} />
    </main>
  );
}

export function ProfileSettingPage() {
  const { user, logout } = useAuthSession();
  return (
    <main className="mobileShell dyUtilityLightPage">
      <Header title="设置" fallback="/me" />
      <Rows rows={[
        { title: '账号与安全', desc: '登录设备、密码与身份信息' },
        { title: '隐私设置', desc: '作品、关注、通讯录可见范围' },
        { title: '通知设置', desc: '互动、直播和系统提醒' },
        { title: '通用设置', desc: '清理缓存、播放与语言' },
        { title: '关于抖音', desc: '版本、协议和帮助中心', href: '/service-protocol' },
      ]} />
      <section className="dyUtilityAccount"><b>{user ? user.nickname || user.username : '未登录'}</b><button type="button" onClick={() => user ? void logout() : window.location.assign('/login')}>{user ? '退出登录' : '登录账号'}</button></section>
    </main>
  );
}

export function ProfileHistoryPage({ title = '观看历史' }: { title?: string }) {
  return (
    <main className="mobileShell dyUtilityLightPage">
      <Header title={title} fallback="/me" />
      <nav className="dyUtilityHistoryTabs">{['视频', '直播', '商城', '竞拍订单'].map((tab, index) => <a href={index === 3 ? '/shop/orders?from=look-history' : '/video-detail'} className={index === 0 ? 'active' : ''} key={tab}>{tab}</a>)}</nav>
      <section className="dyUtilityPosterGrid light">{MUSIC_POSTERS.map((item) => <a href="/video-detail" key={item}><span>{item.slice(0, 1)}</span><b>{item}</b><small>浏览记录</small></a>)}</section>
    </main>
  );
}

export function ProfileSimplePage({ title, rows }: { title: string; rows?: Row[] }) {
  return (
    <main className="mobileShell dyUtilityLightPage">
      <Header title={title} fallback="/me" />
      <Rows rows={rows || [{ title: '选项一', desc: '点击设置' }, { title: '完成', desc: '保存后返回个人主页', href: '/me' }]} />
    </main>
  );
}

export function LoginSubPage({ title, help = false }: { title: string; help?: boolean }) {
  return (
    <main className="mobileShell dyUtilityDarkPage">
      <Header title={title} fallback="/login" />
      {help ? (
        <Rows rows={[{ title: '账号登录问题' }, { title: '手机号不可用' }, { title: '密码找回', href: '/login/retrieve-password' }, { title: '服务协议', href: '/service-protocol' }]} />
      ) : (
        <section className="dyUtilityLoginPanel">
          <h1>{title}</h1>
          <p>抖音参考入口已接入；真实买家登录、注册、重置密码仍走 LiveAuction 后端。</p>
          <input placeholder="账号 / 邮箱 / 手机号" />
          <input placeholder={title.includes('验证码') ? '验证码' : '密码'} type={title.includes('验证码') ? 'text' : 'password'} />
          <a href="/login">使用真实买家账号继续</a>
        </section>
      )}
    </main>
  );
}
