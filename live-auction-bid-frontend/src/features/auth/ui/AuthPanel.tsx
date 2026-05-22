import { useEffect, useState } from 'react';
import { login, logout } from '../api/authApi';
import { currentAuth } from '../api/authApi';
import { resultMessage } from '../../../shared/api/result';
import type { User, UserRole } from '../../../shared/api/types';

type Props = {
  onUserChange?: (user: User | null) => void;
};

function roleAllowed(role?: UserRole) {
  return role === 'USER_ROLE_ANCHOR' || role === 'USER_ROLE_OPERATOR' || role === 'USER_ROLE_ADMIN';
}

function roleLabel(role?: UserRole) {
  switch (role) {
    case 'USER_ROLE_ANCHOR': return '主播';
    case 'USER_ROLE_OPERATOR': return '运营';
    case 'USER_ROLE_ADMIN': return '主播主账号';
    default: return role ? '非后台角色（不可操作后台）' : '未登录';
  }
}

export function AuthPanel({ onUserChange }: Props) {
  const [user, setUser] = useState<User | null>(() => currentAuth().user);
  const [username, setUsername] = useState('admin');
  const [password, setPassword] = useState('admin_dev_password');
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

  const submit = async () => {
    if (busy) return;
    setBusy(true);
    setError('');
    try {
      const next = await login(username.trim(), password);
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
    <article className="card authCard adminAuthCard">
      <h2>后台账号</h2>
      {user ? (
        <>
          <p><strong>{user.nickname}</strong> · {roleLabel(user.role)}</p>
          <p className="meta">当前操作会使用 JWT access token，后端按角色做权限判断。</p>
          {!roleAllowed(user.role) && <p className="formError">当前角色不能操作主播团队工作台，请退出后切换主播主账号、场控、商品助理或订单客服账号。</p>}
          <button className="ghostButton" disabled={busy} onClick={doLogout}>退出登录</button>
        </>
      ) : (
        <>
          <input value={username} onChange={(e) => setUsername(e.target.value)} placeholder="username" autoComplete="username" />
          <input value={password} onChange={(e) => setPassword(e.target.value)} placeholder="password" type="password" autoComplete="current-password" />
          <div className="bidRow">
            <button disabled={busy || !username.trim() || !password} onClick={submit}>{busy ? '处理中...' : '登录后台'}</button>
          </div>
          <p className="meta">本地默认：admin / admin_dev_password。创建、开拍、揭示、Duel、落锤需要 anchor/operator/admin 权限。</p>
        </>
      )}
      {error && <p className="formError">{error}</p>}
    </article>
  );
}
