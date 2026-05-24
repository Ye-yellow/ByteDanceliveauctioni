import { useMemo, useState, type FormEvent } from 'react';
import {
  BUYER_NICKNAME_MAX_LENGTH,
  BUYER_PASSWORD_MAX_LENGTH,
  BUYER_PASSWORD_MIN_LENGTH,
  BUYER_USERNAME_MAX_LENGTH,
  BUYER_USERNAME_MIN_LENGTH,
  BUYER_USERNAME_PATTERN,
  normalizeBuyerUsername,
  validateBuyerCredentials,
} from '../shared/auth/credentialRules';
import { useAuthSession } from '../shared/auth/useAuthSession';

const PROFILE_STATS = [
  { value: '23', label: '获赞' },
  { value: '8', label: '互关' },
  { value: '100', label: '关注' },
  { value: '34', label: '粉丝' },
];

const PROFILE_TABS = ['作品', '日常', '推荐', '收藏', '喜欢'];

function getProfileRoomId() {
  const params = new URLSearchParams(location.search);
  return params.get('roomId') || 'room-jewel-01';
}

function nicknameFromUsername(username: string) {
  const value = username.trim();
  return value ? `买家${value.slice(0, 8)}` : 'H5 买家';
}

function buyerDisplayName(user: ReturnType<typeof useAuthSession>['user']): string {
  if (!user) return '未登录买家';
  return user.nickname?.trim() || '已登录买家';
}

function AccountMenu({
  onClose,
}: {
  onClose: () => void;
}) {
  const { user, status, reason, loginBuyer, registerBuyer, resetBuyerPassword, logout } = useAuthSession();
  const [mode, setMode] = useState<'login' | 'register' | 'reset'>('register');
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [nickname, setNickname] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const [message, setMessage] = useState('');

  const switchMode = (nextMode: 'login' | 'register' | 'reset') => {
    setMode(nextMode);
    setError('');
    setMessage('');
  };

  const submit = async (event: FormEvent) => {
    event.preventDefault();
    if (busy) return;
    const validationError = validateBuyerCredentials({
      username,
      password,
      nickname,
      requireNickname: mode === 'register',
    });
    if (validationError) {
      setError(validationError);
      return;
    }

    setBusy(true);
    setError('');
    setMessage('');
    try {
      const normalizedUsername = normalizeBuyerUsername(username);
      if (mode === 'reset') {
        const resetUser = await resetBuyerPassword(normalizedUsername, password);
        setMode('login');
        setPassword('');
        setNickname('');
        setMessage(`${resetUser.username || normalizedUsername} 密码已重置，请使用新密码登录`);
        return;
      }
      if (mode === 'login') await loginBuyer(normalizedUsername, password);
      else await registerBuyer(normalizedUsername, password, nickname.trim() || nicknameFromUsername(normalizedUsername));
      onClose();
    } catch (e) {
      setError(e instanceof Error ? e.message : '账号处理失败，请重试');
    } finally {
      setBusy(false);
    }
  };

  const handleLogout = () => {
    void logout();
    onClose();
  };

  return (
    <div className="accountMenuMask" onClick={onClose}>
      <section className="accountMenuSheet" role="dialog" aria-modal="true" aria-label="账号菜单" onClick={(event) => event.stopPropagation()}>
        <header className="accountMenuHeader">
          <div>
            <b>{user ? '买家账号' : mode === 'reset' ? '重置密码' : '注册或登录'}</b>
            <span>{user ? '当前账号可参与直播竞拍' : mode === 'reset' ? '输入账号和新密码，重置后再登录' : '登录后才能参与竞拍和查看订单'}</span>
          </div>
          <button type="button" onClick={onClose} aria-label="关闭账号菜单">×</button>
        </header>

        {user ? (
          <section className="accountSignedIn">
            <p><span>昵称</span><b key={`nickname-${user.id}-${user.nickname || ''}`} translate="no">{buyerDisplayName(user)}</b></p>
            <p><span>买家ID</span><b key={`id-${user.id}`} translate="no">{user.id || '待同步'}</b></p>
            <p><span>账号</span><b key={`username-${user.username}`} translate="no">{user.username}</b></p>
            <p><span>状态</span><b>{status === 'authenticated' ? '已登录' : status}</b></p>
            <button type="button" onClick={handleLogout}>退出登录</button>
          </section>
        ) : (
          <form className="accountAuthForm" onSubmit={submit}>
            <div className="accountAuthTabs">
              <button type="button" className={mode === 'register' ? 'active' : ''} onClick={() => switchMode('register')}>注册</button>
              <button type="button" className={mode === 'login' ? 'active' : ''} onClick={() => switchMode('login')}>登录</button>
              <button type="button" className={mode === 'reset' ? 'active' : ''} onClick={() => switchMode('reset')}>重置</button>
            </div>
            <label>
              <span>账号</span>
              <input
                value={username}
                onChange={(event) => setUsername(event.target.value)}
                autoComplete="username"
                minLength={BUYER_USERNAME_MIN_LENGTH}
                maxLength={BUYER_USERNAME_MAX_LENGTH}
                pattern={BUYER_USERNAME_PATTERN}
                title="账号需为 6-64 位，只能包含字母、数字、下划线或中横线"
              />
            </label>
            <label>
              <span>密码</span>
              <input
                value={password}
                onChange={(event) => setPassword(event.target.value)}
                type="password"
                autoComplete={mode === 'login' ? 'current-password' : 'new-password'}
                minLength={BUYER_PASSWORD_MIN_LENGTH}
                maxLength={BUYER_PASSWORD_MAX_LENGTH}
                title="密码需为 8-128 位"
              />
            </label>
            {mode === 'register' ? (
              <label>
                <span>昵称</span>
                <input value={nickname} onChange={(event) => setNickname(event.target.value)} autoComplete="nickname" maxLength={BUYER_NICKNAME_MAX_LENGTH} />
              </label>
            ) : null}
            <button className="accountPrimaryButton" disabled={busy}>{busy ? '处理中...' : mode === 'register' ? '注册买家账号' : mode === 'reset' ? '重置密码' : '登录买家账号'}</button>
            {message ? <p className="accountFormNotice">{message}</p> : null}
            {error || reason ? <p className="accountFormError">{error || reason}</p> : null}
          </form>
        )}
      </section>
    </div>
  );
}

export function ProfilePage() {
  const { user } = useAuthSession();
  const [accountMenuOpen, setAccountMenuOpen] = useState(false);
  const roomId = getProfileRoomId();
  const isSignedIn = Boolean(user);
  const displayName = useMemo(() => buyerDisplayName(user), [user]);
  const identityKey = user?.id || 'anonymous';
  const avatarText = isSignedIn ? displayName.slice(0, 1) : '未';
  const buyerId = user?.id?.trim() || '';
  const accountLine = buyerId ? `买家ID：${buyerId}` : '登录后显示买家ID';
  const historyHref = `/m/history?roomId=${encodeURIComponent(roomId)}&from=profile`;

  return (
    <main className="mobileShell profileShell">
      <div className="profileScroller">
        <header className="profileHero">
          <button type="button" className="profileMenuButton" onClick={() => setAccountMenuOpen(true)} aria-label="打开账号菜单">
            <span />
            <span />
            <span />
          </button>
          <section className="profileIdentity" aria-label="用户信息">
            <div className={`profileAvatar${isSignedIn ? '' : ' empty'}`}>
              <span key={`avatar-${identityKey}-${avatarText}`} translate="no">{avatarText}</span>
              {isSignedIn ? <i>+</i> : null}
            </div>
            <div>
              <h1>
                <span className="profileDisplayName" key={`display-${identityKey}-${displayName}`} translate="no">{displayName}</span>
                <button type="button" className="profileExpandButton" aria-label="展开主页资料" />
              </h1>
              <p key={`account-${identityKey}-${accountLine}`} translate="no">{accountLine}</p>
            </div>
          </section>
        </header>

        <section className="profileBody">
          <div className="profileStats">
            {PROFILE_STATS.map((item) => (
              <span key={item.label}>
                <b>{item.value}</b>
                <small>{item.label}</small>
              </span>
            ))}
            <button type="button">编辑主页</button>
          </div>

          <p className="profileBio">点击添加介绍，让大家认识你... ✎</p>
          <button type="button" className="profileTagButton">+ 添加性别等标签</button>

          <nav className="profileQuickActions" aria-label="我的功能">
            <a href={historyHref}>
              <b>我的订单</b>
            </a>
            <button type="button">
              <b>我的预约</b>
            </button>
            <button type="button">
              <b>观看历史</b>
            </button>
            <button type="button">
              <b>我的钱包</b>
            </button>
            <button type="button">
              <b>全部功能</b>
            </button>
          </nav>

          <nav className="profileTabs" aria-label="内容分类">
            {PROFILE_TABS.map((tab, index) => (
              <button className={index === 0 ? 'active' : ''} type="button" key={tab}>
                {tab}{index === 0 ? ' ▾' : ''}
              </button>
            ))}
          </nav>

          <section className="profileWorksEmpty" aria-label="作品列表">
            <b>暂无公开作品</b>
            <span>发布拍品开箱、竞拍心得后会显示在这里</span>
          </section>
        </section>
      </div>

      <nav className="homeBottomNav" aria-label="底部导航">
        <a href="/">首页</a>
        <span aria-hidden="true" />
        <button type="button" aria-label="发布">+</button>
        <span aria-hidden="true" />
        <b>我</b>
      </nav>
      {accountMenuOpen ? <AccountMenu onClose={() => setAccountMenuOpen(false)} /> : null}
    </main>
  );
}
