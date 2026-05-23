import type { FormEvent } from 'react';
import type { LiveRoomController } from '../hooks/useLiveRoomController';

type BuyerAuth = LiveRoomController['buyerAuth'];

export function BuyerAuthPanel({ auth }: { auth: BuyerAuth }) {
  const submit = (event: FormEvent) => {
    event.preventDefault();
    void auth.submit();
  };

  return (
    <section className="buyerAuthPanel">
      <div className="buyerAuthTabs">
        <button type="button" className={auth.mode === 'login' ? 'active' : ''} onClick={() => auth.setMode('login')}>
          登录
        </button>
        <button type="button" className={auth.mode === 'register' ? 'active' : ''} onClick={() => auth.setMode('register')}>
          注册
        </button>
      </div>
      <form onSubmit={submit}>
        <label>
          <span>账号</span>
          <input value={auth.username} onChange={(event) => auth.setUsername(event.target.value)} autoComplete="username" />
        </label>
        {auth.mode === 'register' ? (
          <label>
            <span>昵称</span>
            <input value={auth.nickname} onChange={(event) => auth.setNickname(event.target.value)} autoComplete="nickname" />
          </label>
        ) : null}
        <label>
          <span>密码</span>
          <input
            value={auth.password}
            onChange={(event) => auth.setPassword(event.target.value)}
            type="password"
            autoComplete={auth.mode === 'login' ? 'current-password' : 'new-password'}
          />
        </label>
        <button className="bidButton" disabled={auth.busy}>
          {auth.busy ? '处理中...' : auth.mode === 'login' ? '登录后出价' : '注册买家账号'}
        </button>
        {auth.error ? <p className="bidError" role="alert">{auth.error}</p> : null}
      </form>
    </section>
  );
}
