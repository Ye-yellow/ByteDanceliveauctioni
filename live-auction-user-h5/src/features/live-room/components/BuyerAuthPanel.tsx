import type { FormEvent } from 'react';
import {
  BUYER_NICKNAME_MAX_LENGTH,
  BUYER_PASSWORD_MAX_LENGTH,
  BUYER_PASSWORD_MIN_LENGTH,
  BUYER_USERNAME_MAX_LENGTH,
  BUYER_USERNAME_MIN_LENGTH,
  BUYER_USERNAME_PATTERN,
} from '../../../shared/auth/credentialRules';
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
        <button type="button" className={auth.mode === 'reset' ? 'active' : ''} onClick={() => auth.setMode('reset')}>
          重置
        </button>
      </div>
      <form onSubmit={submit}>
        <label>
          <span>账号</span>
          <input
            value={auth.username}
            onChange={(event) => auth.setUsername(event.target.value)}
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
            value={auth.password}
            onChange={(event) => auth.setPassword(event.target.value)}
            type="password"
            autoComplete={auth.mode === 'login' ? 'current-password' : 'new-password'}
            minLength={BUYER_PASSWORD_MIN_LENGTH}
            maxLength={BUYER_PASSWORD_MAX_LENGTH}
            title="密码需为 8-128 位"
          />
        </label>
        {auth.mode === 'register' ? (
          <label>
            <span>昵称</span>
            <input value={auth.nickname} onChange={(event) => auth.setNickname(event.target.value)} autoComplete="nickname" maxLength={BUYER_NICKNAME_MAX_LENGTH} />
          </label>
        ) : null}
        <button className="bidButton" disabled={auth.busy}>
          {auth.busy ? '处理中...' : auth.mode === 'login' ? '登录后出价' : auth.mode === 'reset' ? '重置密码' : '注册买家账号'}
        </button>
        {auth.error ? <p className="bidError" role="alert">{auth.error}</p> : null}
      </form>
    </section>
  );
}
