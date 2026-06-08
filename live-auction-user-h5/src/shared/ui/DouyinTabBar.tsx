import { useState, type MouseEvent } from 'react';
import { navigateTo } from '../navigation';

export type DouyinTab = 'home' | 'shop' | 'publish' | 'message' | 'me';

type DouyinTabBarProps = {
  active: DouyinTab;
  isWhite?: boolean;
  className?: string;
  onPublish?: () => void;
  onTab?: (tab: Exclude<DouyinTab, 'publish'>, href: string) => void;
};

const TABS = [
  { key: 'home', label: '首页', href: '/home' },
  { key: 'shop', label: '商城', href: '/shop' },
  { key: 'message', label: '消息', href: '/message', badge: '2' },
] satisfies Array<{ key: Exclude<DouyinTab, 'publish' | 'me'>; label: string; href: string; badge?: string }>;

export function DouyinTabBar({ active, isWhite, className = '', onPublish, onTab }: DouyinTabBarProps) {
  const [refreshing, setRefreshing] = useState<Extract<DouyinTab, 'home' | 'shop'> | ''>('');

  const handleTab = (tab: Exclude<DouyinTab, 'publish'>, href: string) => (
    event: MouseEvent<HTMLAnchorElement | HTMLButtonElement>,
  ) => {
    event.preventDefault();
    if (tab === active && (tab === 'home' || tab === 'shop')) {
      setRefreshing(tab);
      window.setTimeout(() => setRefreshing(''), 700);
      return;
    }

    if (onTab) {
      onTab(tab, href);
      return;
    }

    navigateTo(href);
  };

  const renderTab = (tab: { key: Exclude<DouyinTab, 'publish' | 'me'>; label: string; href: string; badge?: string }) => {
    const content = (
      <>
        {refreshing === tab.key ? <span className="douyinTabRefresh" aria-hidden="true" /> : <span className={active === tab.key ? 'active' : ''}>{tab.label}</span>}
        {tab.badge ? <span className="badge">{tab.badge}</span> : null}
      </>
    );
    return active === tab.key ? (
      <button type="button" className="l-button refreshable" onClick={handleTab(tab.key, tab.href)} key={tab.key}>{content}</button>
    ) : (
      <a className="l-button" href={tab.href} onClick={handleTab(tab.key, tab.href)} key={tab.key}>{content}</a>
    );
  };

  const publishControl = onPublish ? (
    <button type="button" className="douyinTabPublish l-button" aria-label="发布" onClick={onPublish}>
      <span className="add-ctn">
        <img src="/douyin-assets/icons/add-light.png" alt="" className="add" />
      </span>
    </button>
  ) : (
    <a className="douyinTabPublish l-button" aria-label="发布" href="/publish" onClick={(event) => { event.preventDefault(); navigateTo('/publish'); }}>
      <span className="add-ctn">
        <img src="/douyin-assets/icons/add-light.png" alt="" className="add" />
      </span>
    </a>
  );

  const whiteMode = isWhite ?? active === 'shop';

  return (
    <nav className={`douyinTabBar footer ${whiteMode ? 'isWhite' : ''} ${className}`.trim()} aria-label="底部导航">
      {TABS.slice(0, 2).map(renderTab)}
      {publishControl}
      {renderTab(TABS[2])}
      {active === 'me'
        ? <button type="button" className="l-button" onClick={handleTab('me', '/me')}><span className="active">我</span></button>
        : <a className="l-button" href="/me" onClick={handleTab('me', '/me')}><span>我</span></a>}
    </nav>
  );
}
