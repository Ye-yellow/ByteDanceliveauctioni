import { useState } from 'react';
import { useAuthSession } from '../../../shared/auth/useAuthSession';

const ME_TABS = ['作品', '私密', '喜欢', '收藏'] as const;
type MeTab = (typeof ME_TABS)[number];

export function UserPanel() {
  const { user } = useAuthSession();
  const [activeTab, setActiveTab] = useState<MeTab>('作品');
  const isLoggedIn = Boolean(user);
  const displayName = user?.nickname?.trim() || user?.username?.trim() || '未登录';
  const douyinId = user?.username?.trim() || (isLoggedIn ? user?.id || '未设置' : '未设置');
  const avatarText = isLoggedIn ? displayName.slice(0, 1) : '';

  return (
    <aside className="dyHomeReplicaUserPanel">
      <header className="dyHomeReplicaMeFloat float">
        <a className="dyHomeReplicaMeEdit left" href={isLoggedIn ? '/me/edit' : '/login?next=/home'}>
          <svg className="iconify iconify--ri" width="1em" height="1em" viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M7.243 17.997H3v-4.243L14.435 2.319a1 1 0 0 1 1.414 0l2.829 2.828a1 1 0 0 1 0 1.415zm-4.243 2h18v2H3z" /></svg>
          <span>编辑资料</span>
        </a>
        <div className="dyHomeReplicaMeFloatActions right">
          <button type="button" className="dyHomeReplicaMeFloatItem item" aria-label="常用互动">
            <svg className="finger iconify iconify--fluent-emoji-high-contrast" width="1em" height="1em" viewBox="0 0 32 32" aria-hidden="true"><path fill="currentColor" d="M15.86 31c-6.5 0-10.876-5.269-10.876-13.109a3.42 3.42 0 0 1 1.176-2.843a3.3 3.3 0 0 1 1.854-.679c.151-1.585.76-3.231 2.944-3.39c.355-.026.711 0 1.058.08V4.531A3.53 3.53 0 0 1 15.531 1a3.457 3.457 0 0 1 3.453 3.531v5.61q.479-.055.956.008a3.53 3.53 0 0 1 2.344 1.435a2.9 2.9 0 0 1 1.5-.269a3.216 3.216 0 0 1 3.187 3.31C26.969 24.879 22.815 31 15.86 31M8.016 16.373a1.5 1.5 0 0 0-.614.243a1.59 1.59 0 0 0-.418 1.275C6.984 24.535 10.551 29 15.86 29c8.221 0 9.109-10.053 9.109-14.375c0-.8-.452-1.245-1.345-1.315a.86.86 0 0 0-.77.306a.98.98 0 0 1-.963.488a1 1 0 0 1-.845-.672c0-.005-.433-1.18-1.361-1.3a1.55 1.55 0 0 0-.7.036v1.113a1 1 0 1 1-2 0v-1.717l-.001-.015V4.531A1.453 1.453 0 0 0 15.531 3a1.54 1.54 0 0 0-1.515 1.531v8.2a1 1 0 0 1-1.762.648a1.53 1.53 0 0 0-1.15-.408c-.444.033-.937.069-1.088 1.453v2.092a1 1 0 1 1-2 0z" /></svg>
          </button>
          <a className="dyHomeReplicaMeFloatItem item" href="/people/find-acquaintance" aria-label="朋友">
            <svg className="iconify iconify--eva" width="1em" height="1em" viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M9 11a4 4 0 1 0-4-4a4 4 0 0 0 4 4m8 2a3 3 0 1 0-3-3a3 3 0 0 0 3 3m0 1a5 5 0 0 0-3.06 1.05A7 7 0 0 0 2 20a1 1 0 0 0 2 0a5 5 0 0 1 10 0a1 1 0 0 0 2 0a6.9 6.9 0 0 0-.86-3.35A3 3 0 0 1 20 19a1 1 0 0 0 2 0a5 5 0 0 0-5-5" /></svg>
          </a>
          <a className="dyHomeReplicaMeFloatItem item" href="/home/search" aria-label="搜索">
            <svg className="iconify iconify--ic" width="1em" height="1em" viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M15.5 14h-.79l-.28-.27a6.5 6.5 0 0 0 1.48-5.34c-.47-2.78-2.79-5-5.59-5.34a6.505 6.505 0 0 0-7.27 7.27c.34 2.8 2.56 5.12 5.34 5.59a6.5 6.5 0 0 0 5.34-1.48l.27.28v.79l4.25 4.25c.41.41 1.08.41 1.49 0s.41-1.08 0-1.49zM9.5 14C7.01 14 5 11.99 5 9.5S7.01 5 9.5 5S14 7.01 14 9.5S11.99 14 9.5 14" /></svg>
          </a>
          <a className="dyHomeReplicaMeFloatItem item" href="/me/right-menu/setting" aria-label="菜单">
            <svg className="iconify iconify--ic" width="1em" height="1em" viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M4 18h16c.55 0 1-.45 1-1s-.45-1-1-1H4c-.55 0-1 .45-1 1s.45 1 1 1m0-5h16c.55 0 1-.45 1-1s-.45-1-1-1H4c-.55 0-1 .45-1 1s.45 1 1 1M3 7c0 .55.45 1 1 1h16c.55 0 1-.45 1-1s-.45-1-1-1H4c-.55 0-1 .45-1 1" /></svg>
          </a>
        </div>
      </header>

      <section className="dyHomeReplicaMeProfile">
        <header className={`dyHomeReplicaMeCover${isLoggedIn ? '' : ' isAnonymous'}`}>
          <div className="dyHomeReplicaMeIdentity">
            <span className={`dyHomeReplicaMeAvatar${isLoggedIn ? '' : ' isEmpty'}`}>{avatarText}</span>
            <div>
              <h2>{displayName}</h2>
              <p>
                抖音号：{douyinId}
                {isLoggedIn ? <img src="/douyin-assets/me/qrcode-gray.png" alt="" /> : null}
              </p>
            </div>
          </div>
        </header>

        <section className="dyHomeReplicaMeDetail">
          <div className="dyHomeReplicaMeHead">
            <nav className="dyHomeReplicaMeHeat" aria-label="账号数据">
            {['0 获赞', '0 朋友', '0 关注', '0 粉丝'].map((item) => {
              const [value, label] = item.split(' ');
                return <span className="dyHomeReplicaMeHeatText" key={item}><b>{value}</b><small>{label}</small></span>;
            })}
            </nav>
            <a className="dyHomeReplicaMePrimaryButton" href={isLoggedIn ? '/people/find-acquaintance' : '/login?next=/home'}>
            {isLoggedIn ? '添加朋友' : '登录'}
            </a>
          </div>

          <div className="dyHomeReplicaMeSignature">
            <div>{isLoggedIn ? '点击添加介绍，让大家认识你...' : '登录后完善资料，让大家认识你...'}</div>
          </div>
          <div className="dyHomeReplicaMeMore">
            <span className="dyHomeReplicaMeMetaItem">{isLoggedIn ? '暂无资料' : '未登录'}</span>
          </div>

          <nav className="dyHomeReplicaMeOther" aria-label="个人功能">
            <a href="/shop">
              <svg viewBox="0 0 24 24" aria-hidden="true">
                <g fill="none">
                  <path fill="currentColor" d="M3 2.25a.75.75 0 0 0 0 1.5zM5 3l.748-.058A.75.75 0 0 0 5 2.25zm16 3l.745.083A.75.75 0 0 0 21 5.25zM5.23 6l-.747.058zm13.109 9.119l.053.748zm-10.355.74l-.053-.749zM3 3.75h2v-1.5H3zm5.037 12.856l10.355-.74l-.107-1.495l-10.354.74zm12.892-3.179l.816-7.344l-1.49-.166l-.816 7.345zM4.252 3.057l.231 3l1.496-.115l-.231-3zm.231 3l.617 8.017l1.495-.115l-.616-8.017zM21 5.25H5.23v1.5H21zm-2.608 10.617a2.75 2.75 0 0 0 2.537-2.44l-1.49-.165a1.25 1.25 0 0 1-1.154 1.109zM7.931 15.11a1.25 1.25 0 0 1-1.336-1.15l-1.495.114a2.75 2.75 0 0 0 2.937 2.532z" />
                  <path stroke="currentColor" strokeLinejoin="round" strokeWidth="2.25" d="M8.5 20.5h.01v.01H8.5zm9 0h.01v.01h-.01z" />
                </g>
              </svg>
              <span>抖音商城</span>
            </a>
            <a href="/home/music">
              <svg viewBox="0 0 24 24" aria-hidden="true">
                <g fill="none" stroke="currentColor" strokeLinecap="round" strokeWidth="1.5">
                  <circle cx="6" cy="18" r="3" strokeLinejoin="round" />
                  <path strokeLinejoin="round" d="M9 18V5" />
                  <path d="M21 3L9 5m12 2L9 9" />
                  <circle cx="18" cy="16" r="3" strokeLinejoin="round" />
                  <path strokeLinejoin="round" d="M21 16V3" />
                </g>
              </svg>
              <span>我的音乐</span>
            </a>
            <a href="/message">
              <svg viewBox="0 0 14 14" aria-hidden="true">
                <g fill="none" stroke="currentColor" strokeLinecap="round" strokeLinejoin="round">
                  <path d="M9.25 5a4.25 4.25 0 0 1 3.54 6.6l.71 1.9l-2.39-.43A4.25 4.25 0 1 1 9.25 5" />
                  <path d="M9.86 2.51A5.24 5.24 0 0 0 .5 5.75a5.2 5.2 0 0 0 .88 2.91L.5 11l2.12-.38" />
                </g>
              </svg>
              <span>我的群聊</span>
            </a>
            <a href="/me/right-menu/setting">
              <svg viewBox="0 0 24 24" aria-hidden="true">
                <g fill="none">
                  <path fill="currentColor" d="M3 2.25a.75.75 0 0 0 0 1.5zM5 3l.748-.058A.75.75 0 0 0 5 2.25zm16 3l.745.083A.75.75 0 0 0 21 5.25zM5.23 6l-.747.058zm13.109 9.119l.053.748zm-10.355.74l-.053-.749zM3 3.75h2v-1.5H3zm5.037 12.856l10.355-.74l-.107-1.495l-10.354.74zm12.892-3.179l.816-7.344l-1.49-.166l-.816 7.345zM4.252 3.057l.231 3l1.496-.115l-.231-3zm.231 3l.617 8.017l1.495-.115l-.616-8.017zM21 5.25H5.23v1.5H21zm-2.608 10.617a2.75 2.75 0 0 0 2.537-2.44l-1.49-.165a1.25 1.25 0 0 1-1.154 1.109zM7.931 15.11a1.25 1.25 0 0 1-1.336-1.15l-1.495.114a2.75 2.75 0 0 0 2.937 2.532z" />
                  <path stroke="currentColor" strokeLinejoin="round" strokeWidth="2.25" d="M8.5 20.5h.01v.01H8.5zm9 0h.01v.01h-.01z" />
                </g>
              </svg>
              <span>查看更多</span>
            </a>
          </nav>
        </section>
      </section>

      <nav className="dyHomeReplicaMeTabs" aria-label="作品分类">
        {ME_TABS.map((tab) => (
          <button type="button" className={activeTab === tab ? 'active' : ''} onClick={() => setActiveTab(tab)} key={tab}>{tab}</button>
        ))}
      </nav>

      <section className="dyHomeReplicaMeEmpty">
        {activeTab === '作品' ? null : <img src="/douyin-assets/me/lock-gray.png" alt="" />}
        <p>{activeTab === '作品' ? '还没有发布作品' : activeTab === '私密' ? '只有你能看到设为私密的作品和日常' : activeTab === '喜欢' ? '只有你能看到自己的喜欢列表' : '只有你能看到自己的收藏列表'}</p>
        <small>暂时没有更多了</small>
      </section>

    </aside>
  );
}
