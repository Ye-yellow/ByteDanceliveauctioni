import { useMemo, useState } from 'react';
import { ArrowLeft, Eye, EyeOff, LockKeyhole, LogIn, UserRound } from 'lucide-react';
import { login, register } from '../../features/auth/api/authApi';
import { currentAuth } from '../../features/auth/api/authApi';
import { resultMessage } from '../../shared/api/result';

function nextPath(fallback = '/host') {
  const params = new URLSearchParams(location.search);
  const next = params.get('next');
  if (!next || !next.startsWith('/') || next.startsWith('//')) return fallback;
  return next;
}

export function LoginPage({ embedded = false, title = '欢迎回来' }: { embedded?: boolean; title?: string }) {
  const redirectTo = useMemo(() => nextPath('/host'), []);
  const [mode, setMode] = useState<'host' | 'buyer'>('host');
  const [username, setUsername] = useState('admin');
  const [password, setPassword] = useState('admin_dev_password');
  const [nickname, setNickname] = useState('观众一号');
  const [showPassword, setShowPassword] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');

  const current = currentAuth().user;

  const switchMode = (next: 'host' | 'buyer') => {
    setMode(next);
    setError('');
    if (next === 'host') {
      setUsername('admin');
      setPassword('admin_dev_password');
    } else {
      setUsername(`buyer${Math.floor(Math.random() * 9000 + 1000)}`);
      setPassword('password123');
      setNickname(`观众${Math.floor(Math.random() * 90 + 10)}`);
    }
  };

  const submit = async (kind: 'login' | 'register') => {
    if (busy) return;
    setBusy(true);
    setError('');
    try {
      if (kind === 'register') {
        await register(username.trim(), password, nickname.trim());
      } else {
        await login(username.trim(), password);
      }
      location.href = mode === 'buyer' ? '/room' : redirectTo;
    } catch (e) {
      setError(resultMessage(e));
    } finally {
      setBusy(false);
    }
  };

  return (
    <main className={embedded ? 'loginPage embedded' : 'loginPage'}>
      {!embedded && <a className="loginBack" href="/home"><ArrowLeft size={16} /> 返回首页</a>}
      <section className="loginCenter">
        <a className="loginLogo" href="/home" aria-label="实时竞拍大师首页">
          <span>竞</span>
          <strong>实时竞拍大师</strong>
          <small>Live Auction Master</small>
        </a>

        <article className="loginCard">
          <p className="loginEyebrow">Douyin E-commerce Auction</p>
          <h1>{current ? `已登录：${current.nickname}` : title}</h1>
          <p className="loginSubcopy">登录后进入主播运营后台；观众可注册买家账号进入直播间参与出价。</p>

          <div className="loginTabs" role="tablist" aria-label="登录模式">
            <button className={mode === 'host' ? 'active' : ''} onClick={() => switchMode('host')}>主播/运营</button>
            <button className={mode === 'buyer' ? 'active' : ''} onClick={() => switchMode('buyer')}>观众买家</button>
          </div>

          <label className="loginField">
            <span>账号</span>
            <div><UserRound size={16} /><input value={username} onChange={(e) => setUsername(e.target.value)} placeholder="username" /></div>
          </label>
          <label className="loginField">
            <span>密码</span>
            <div>
              <LockKeyhole size={16} />
              <input value={password} onChange={(e) => setPassword(e.target.value)} placeholder="password" type={showPassword ? 'text' : 'password'} />
              <button type="button" className="passwordToggle" onClick={() => setShowPassword((x) => !x)}>{showPassword ? <EyeOff size={15} /> : <Eye size={15} />}</button>
            </div>
          </label>
          {mode === 'buyer' && (
            <label className="loginField">
              <span>昵称</span>
              <div><UserRound size={16} /><input value={nickname} onChange={(e) => setNickname(e.target.value)} placeholder="nickname" /></div>
            </label>
          )}

          {error && <p className="loginError">{error}</p>}

          <button className="loginSubmit" disabled={busy || !username.trim() || !password || (mode === 'buyer' && !nickname.trim())} onClick={() => submit('login')}>
            <LogIn size={17} /> {busy ? '处理中...' : '登录'}
          </button>
          {mode === 'buyer' && <button className="loginGhost" disabled={busy || !username.trim() || !password || !nickname.trim()} onClick={() => submit('register')}>注册买家并进入直播间</button>}

          <p className="loginHint">本地主播默认账号：admin / admin_dev_password</p>
        </article>
      </section>
    </main>
  );
}
