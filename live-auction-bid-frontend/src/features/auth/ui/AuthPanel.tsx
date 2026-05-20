import { useEffect, useState } from 'react';
import { login, logout, register } from '../api/authApi';
import { currentAuth } from '../api/authApi';
import { resultMessage } from '../../../shared/api/result';
import type { User, UserRole } from '../../../shared/api/types';

type Props = {
  mode: 'buyer' | 'host';
  onUserChange?: (user: User | null) => void;
};

function roleAllowed(mode: Props['mode'], role?: UserRole) {
  if (mode === 'buyer') return role === 'USER_ROLE_BUYER';
  return role === 'USER_ROLE_ANCHOR' || role === 'USER_ROLE_OPERATOR' || role === 'USER_ROLE_ADMIN';
}

function roleLabel(role?: UserRole) {
  switch (role) {
    case 'USER_ROLE_BUYER': return '买家';
    case 'USER_ROLE_ANCHOR': return '主播';
    case 'USER_ROLE_OPERATOR': return '运营';
    case 'USER_ROLE_ADMIN': return '管理员';
    default: return '未登录';
  }
}

export function AuthPanel({ mode, onUserChange }: Props) {
  const [user, setUser] = useState<User | null>(() => currentAuth().user);
  const [username, setUsername] = useState(mode === 'host' ? 'admin' : `buyer${Math.floor(Math.random() * 9000 + 1000)}`);
  const [password, setPassword] = useState(mode === 'host' ? 'admin_dev_password' : 'password123');
  const [nickname, setNickname] = useState(mode === 'host' ? '管理员' : `观众${Math.floor(Math.random() * 90 + 10)}`);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    const sync = () => {
      const next = currentAuth().user;
      setUser(next);
      onUserChange?.(next);
    };
    sync();
    window.addEventListener('auth-state-change', sync);
    return () => window.removeEventListener('auth-state-change', sync);
  }, [onUserChange]);

  const submit = async (kind: 'login' | 'register') => {
    if (busy) return;
    setBusy(true);
    setError('');
    try {
      const next = kind === 'login' ? await login(username.trim(), password) : await register(username.trim(), password, nickname.trim());
      setUser(next);
      onUserChange?.(next);
    } catch (e) {
      setError(resultMessage(e));
    } finally {
      setBusy(false);
    }
  };

  const doLogout = async () => {
    await logout();
    setUser(null);
    onUserChange?.(null);
  };

  return (
    <article className="card authCard">
      <h2>{mode === 'host' ? '主播/运营登录' : '观众登录'}</h2>
      {user ? (
        <>
          <p><strong>{user.nickname}</strong> · {roleLabel(user.role)}</p>
          <p className="meta">当前操作会使用 JWT access token，后端按角色做权限判断。</p>
          {!roleAllowed(mode, user.role) && <p className="formError">当前角色不能用于{mode === 'buyer' ? '观众出价' : '主播控制台'}，请退出后切换账号。</p>}
          <button className="ghostButton" disabled={busy} onClick={doLogout}>退出登录</button>
        </>
      ) : (
        <>
          <input value={username} onChange={(e) => setUsername(e.target.value)} placeholder="username" />
          <input value={password} onChange={(e) => setPassword(e.target.value)} placeholder="password" type="password" />
          {mode === 'buyer' && <input value={nickname} onChange={(e) => setNickname(e.target.value)} placeholder="nickname" />}
          <div className="bidRow">
            <button disabled={busy || !username.trim() || !password} onClick={() => submit('login')}>{busy ? '处理中...' : '登录'}</button>
            {mode === 'buyer' && <button className="ghost" disabled={busy || !username.trim() || !password || !nickname.trim()} onClick={() => submit('register')}>注册买家</button>}
          </div>
          {mode === 'host' && <p className="meta">本地默认：admin / admin_dev_password。创建、开拍、揭示、Duel、落锤需要 anchor/operator/admin 权限。</p>}
        </>
      )}
      {error && <p className="formError">{error}</p>}
    </article>
  );
}
