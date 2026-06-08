import { useState, type FormEvent } from 'react';
import {
  BUYER_PASSWORD_MAX_LENGTH,
  BUYER_PASSWORD_MIN_LENGTH,
  BUYER_USERNAME_MAX_LENGTH,
  BUYER_USERNAME_MIN_LENGTH,
  BUYER_USERNAME_PATTERN,
  normalizeBuyerUsername,
  validateBuyerCredentials,
} from '../shared/auth/credentialRules';
import { useAuthSession } from '../shared/auth/useAuthSession';
import './login-replica.css';

type AuthMode = 'login' | 'register' | 'reset';

function nextHref(): string {
  const next = new URLSearchParams(location.search).get('next') || '/home';
  if (!next.startsWith('/') || next.startsWith('//') || next.startsWith('/\\')) return '/home';
  try {
    const url = new URL(next, window.location.origin);
    if (url.origin !== window.location.origin) return '/home';
    return `${url.pathname}${url.search}${url.hash}` || '/home';
  } catch {
    return '/home';
  }
}

function nicknameFromUsername(username: string) {
  const value = username.trim();
  return value ? `买家${value.slice(0, 8)}` : 'H5 买家';
}

function goHome() {
  window.location.assign('/home');
}

export function LoginPage() {
  const { user, loginBuyer, registerBuyer, resetBuyerPassword, logout, reason } = useAuthSession();
  const [mode, setMode] = useState<AuthMode>('login');
  const [agreed, setAgreed] = useState(false);
  const [showTooltip, setShowTooltip] = useState(false);
  const [shakeProtocol, setShakeProtocol] = useState(false);
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const [message, setMessage] = useState('');
  const targetHref = nextHref();

  function requireAgreement() {
    if (agreed) return true;
    if (!shakeProtocol && !showTooltip) {
      setShakeProtocol(true);
      window.setTimeout(() => {
        setShakeProtocol(false);
        setShowTooltip(true);
      }, 450);
      window.setTimeout(() => setShowTooltip(false), 3200);
    }
    return false;
  }

  function switchMode(nextMode: AuthMode) {
    setMode(nextMode);
    setError('');
    setMessage('');
    setConfirmPassword('');
  }

  async function submit(event: FormEvent) {
    event.preventDefault();
    if (busy) return;
    if (!requireAgreement()) return;

    if (mode !== 'login' && password !== confirmPassword) {
      setError('两次输入的密码不一致');
      return;
    }

    const validationError = validateBuyerCredentials({
      username,
      password,
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
        await resetBuyerPassword(normalizedUsername, password);
        setMode('login');
        setPassword('');
        setConfirmPassword('');
        setMessage('密码已重置，请使用新密码登录');
        return;
      }
      if (mode === 'register') {
        await registerBuyer(normalizedUsername, password, nicknameFromUsername(normalizedUsername));
      } else {
        await loginBuyer(normalizedUsername, password);
      }
      window.location.assign(targetHref);
    } catch (e) {
      setError(e instanceof Error ? e.message : '账号处理失败，请重试');
    } finally {
      setBusy(false);
    }
  }

  if (user) {
    return (
      <main className="dyLoginReplica">
        <header className="dyLoginReplicaHeader">
          <button type="button" aria-label="关闭" onClick={goHome}>×</button>
          <a href="/login/help">帮助与设置</a>
        </header>
        <section className="dyLoginReplicaSigned">
          <span>{(user.nickname || user.username || '抖').slice(0, 1)}</span>
          <h1>{user.nickname || user.username}</h1>
          <p>买家ID {user.id || '待同步'}</p>
          <a href={targetHref}>继续</a>
          <button type="button" onClick={() => void logout()}>退出登录</button>
        </section>
      </main>
    );
  }

  return (
    <main className="dyLoginReplica">
      <header className="dyLoginReplicaHeader">
        <button type="button" aria-label="关闭" onClick={goHome}>×</button>
        <a href="/login/help">帮助与设置</a>
      </header>

      <section className="dyLoginReplicaContent">
        <section className="dyLoginReplicaDesc">
          <h1>{mode === 'register' ? '注册主账号' : mode === 'reset' ? '找回密码' : 'LiveAuction 账号登录'}</h1>
          <p>使用 LiveAuction 账号登录后，可以继续查看订单、竞拍结果和支付流程。</p>
        </section>

        <form className="dyLoginReplicaForm" onSubmit={submit}>
          <nav aria-label="登录方式">
            <button type="button" className={mode === 'login' ? 'active' : ''} onClick={() => switchMode('login')}>登录</button>
            <button type="button" className={mode === 'register' ? 'active' : ''} onClick={() => switchMode('register')}>注册</button>
            <button type="button" className={mode === 'reset' ? 'active' : ''} onClick={() => switchMode('reset')}>找回</button>
          </nav>
          <label>
            <span>账号</span>
            <input
              value={username}
              onChange={(event) => setUsername(event.target.value)}
              autoComplete="username"
              minLength={BUYER_USERNAME_MIN_LENGTH}
              maxLength={BUYER_USERNAME_MAX_LENGTH}
              pattern={BUYER_USERNAME_PATTERN}
              placeholder="请输入 LiveAuction 账号"
              title="账号需为 6-64 位，只能包含字母、数字、下划线或中横线"
            />
          </label>
          <label>
            <span>{mode === 'reset' ? '新密码' : '密码'}</span>
            <input
              value={password}
              onChange={(event) => setPassword(event.target.value)}
              type="password"
              autoComplete={mode === 'login' ? 'current-password' : 'new-password'}
              minLength={BUYER_PASSWORD_MIN_LENGTH}
              maxLength={BUYER_PASSWORD_MAX_LENGTH}
              placeholder={mode === 'reset' ? '请输入新密码' : '请输入密码'}
              title="密码需为 8-128 位"
            />
          </label>
          {mode !== 'login' ? (
            <label>
              <span>确认密码</span>
              <input
                value={confirmPassword}
                onChange={(event) => setConfirmPassword(event.target.value)}
                type="password"
                autoComplete="new-password"
                minLength={BUYER_PASSWORD_MIN_LENGTH}
                maxLength={BUYER_PASSWORD_MAX_LENGTH}
                placeholder="请再次输入密码"
              />
            </label>
          ) : null}
          <div className="dyLoginReplicaFieldActions">
            <button type="button" onClick={() => switchMode('reset')}>忘记密码</button>
            <button type="button" onClick={() => switchMode('register')}>注册主账号</button>
          </div>
          <button type="submit" className="dyLoginReplicaPrimary" disabled={busy}>
            {busy ? '处理中' : mode === 'register' ? '注册并登录' : mode === 'reset' ? '重置密码' : '登录'}
          </button>
        </form>

        <section className={`dyLoginReplicaProtocol${shakeProtocol ? ' isShake' : ''}`}>
          {showTooltip ? <em>请先勾选协议</em> : null}
          <label>
            <input type="checkbox" checked={agreed} onChange={(event) => setAgreed(event.target.checked)} />
            <span>我已阅读并同意 <a href="/service-protocol">用户协议</a> 和 <a href="/service-protocol">隐私政策</a></span>
          </label>
        </section>

        {message ? <p className="dyLoginReplicaNotice">{message}</p> : null}
        {error || reason ? <p className="dyLoginReplicaError" role="alert">{error || reason}</p> : null}

        {mode !== 'login' ? <button type="button" className="dyLoginReplicaBackQuick" onClick={() => switchMode('login')}>返回账号登录</button> : null}
      </section>
    </main>
  );
}
