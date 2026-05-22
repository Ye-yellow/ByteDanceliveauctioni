import { useEffect, useMemo, useState } from 'react';
import {
  ArrowLeft,
  ArrowRight,
  ClipboardList,
  Coins,
  Eye,
  EyeOff,
  Gavel,
  LockKeyhole,
  LogIn,
  Package,
  Radio,
  ShieldCheck,
  ShoppingBag,
  Sparkles,
  UserRound,
} from 'lucide-react';
import { currentAuth, login, logout } from '../../features/auth/api/authApi';
import { resultMessage } from '../../shared/api/result';

function nextPath(fallback = '/host') {
  const params = new URLSearchParams(location.search);
  const next = params.get('next');
  if (!next || !next.startsWith('/') || next.startsWith('//')) return fallback;
  if (!next.startsWith('/host') && !next.startsWith('/admin')) return fallback;
  return next;
}

const capabilityItems = [
  { icon: <Package size={16} />, label: '拍品准备' },
  { icon: <Gavel size={16} />, label: '竞拍玩法' },
  { icon: <ClipboardList size={16} />, label: '成交处理' },
];

function LoginParticleField() {
  return (
    <div className="authParticleField" aria-hidden="true">
      {Array.from({ length: 64 }).map((_, index) => (
        <i key={index} style={{ '--i': index } as React.CSSProperties} />
      ))}
    </div>
  );
}

function AuctionDecorations() {
  return (
    <div className="authFloatLayer" aria-hidden="true">
      <div className="authFloatCard authLiveCard">
        <small>LIVE AUCTION</small>
        <strong>竞拍中</strong>
        <span>00:20</span>
      </div>
      <div className="authBidStack">
        <span>Bid +¥50</span>
        <span>Bid +¥100</span>
        <span>Bid +¥200</span>
      </div>
      <div className="authProductCard authArtCard">
        <i><Sparkles size={22} /></i>
        <strong>Art Print</strong>
        <span>当前价 ¥199</span>
      </div>
      <div className="authProductCard authBagCard">
        <i><ShoppingBag size={24} /></i>
        <strong>Designer Bag</strong>
        <span>出价 ¥2,880</span>
      </div>
      <div className="authGiftBubble"><ShoppingBag size={34} /></div>
      <div className="authGavelSketch"><Gavel size={112} /></div>
      <div className="authCoin coinA"><Coins size={16} /></div>
      <div className="authCoin coinB"><Coins size={14} /></div>
    </div>
  );
}

export function LoginPage({ embedded = false }: { embedded?: boolean; title?: string }) {
  const redirectTo = useMemo(() => nextPath('/host'), []);
  const [username, setUsername] = useState('admin');
  const [password, setPassword] = useState('admin_dev_password');
  const [rememberMe, setRememberMe] = useState(true);
  const [showPassword, setShowPassword] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');

  const current = currentAuth().user;

  useEffect(() => {
    const params = new URLSearchParams(location.search);
    if (!params.get('expired')) return;
    const fallback = '登录已过期，请重新登录';
    const message = sessionStorage.getItem('liveauction.auth.expiredMessage') || fallback;
    sessionStorage.removeItem('liveauction.auth.expiredMessage');
    setError(message);
  }, []);

  const submit = async () => {
    if (busy) return;
    setBusy(true);
    setError('');
    try {
      await login(username.trim(), password);
      location.href = redirectTo;
    } catch (e) {
      setError(resultMessage(e));
    } finally {
      setBusy(false);
    }
  };

  const switchAccount = async () => {
    setBusy(true);
    setError('');
    try {
      await logout();
      location.reload();
    } catch (e) {
      setError(resultMessage(e));
      setBusy(false);
    }
  };

  return (
    <main className={embedded ? 'loginPage adminOnlyLogin embedded' : 'loginPage adminOnlyLogin'}>
      <LoginParticleField />
      <AuctionDecorations />
      {!embedded && <a className="loginBack" href="/home"><ArrowLeft size={16} /> 返回首页</a>}

      <section className="loginCenter authShell" aria-label="ByteDance LiveAuction 登录入口">
        <aside className="authBrandPanel">
          <div>
            <p className="authKicker"><Radio size={14} /> LIVEAUCTION STUDIO</p>
            <h1>直播间竞拍工作台</h1>
            <p className="authBrandLead">拍品准备、竞拍玩法、直播间中控台、成交处理，从这里开始。</p>
          </div>

          <div className="authBrandScene" aria-hidden="true">
            <div className="authVisualGavel"><Gavel size={88} /></div>
            <div className="authVisualProduct">
              <ShoppingBag size={34} />
              <strong>Premium Lot</strong>
              <span>¥ 1,280</span>
            </div>
            <div className="authVisualBid bidOne">Bid +¥50</div>
            <div className="authVisualBid bidTwo">Bid +¥100</div>
          </div>

          <div className="authCapabilityGrid" aria-label="核心能力">
            {capabilityItems.map((item) => (
              <span key={item.label}>{item.icon}{item.label}</span>
            ))}
          </div>
        </aside>

        <article className="loginCard adminLoginCard authFormPanel">
          {current ? (
            <div className="authSignedInState">
              <p className="loginEyebrow"><ShieldCheck size={14} /> ByteDance LiveAuction</p>
              <h2>欢迎回来，{current.nickname}</h2>
              <p className="loginSubcopy">当前账号已登录，系统会根据该账号绑定的团队权限开放拍品准备、直播中控、成交处理、数据复盘等功能。</p>
              {error && <p className="loginError">{error}</p>}
              <div className="authSignedActions">
                <a className="loginSubmit authPrimaryLink" href={redirectTo}>继续进入工作台 <ArrowRight size={17} /></a>
                <button className="loginGhost" type="button" disabled={busy} onClick={switchAccount}>切换账号</button>
              </div>
            </div>
          ) : (
            <>
              <p className="loginEyebrow"><ShieldCheck size={14} /> ByteDance LiveAuction</p>
              <h2>进入工作台</h2>
              <p className="loginSubcopy">使用主播主账号或团队子账号登录；账号所属主播空间和岗位权限由系统自动识别。</p>

              <label className="loginField">
                <span>账号 / 邮箱 / 手机号</span>
                <div><UserRound size={16} /><input value={username} onChange={(e) => setUsername(e.target.value)} placeholder="请输入主播团队账号" autoComplete="username" /></div>
              </label>
              <label className="loginField">
                <span>密码</span>
                <div>
                  <LockKeyhole size={16} />
                  <input value={password} onChange={(e) => setPassword(e.target.value)} placeholder="请输入密码" type={showPassword ? 'text' : 'password'} autoComplete="current-password" />
                  <button type="button" className="passwordToggle" onClick={() => setShowPassword((x) => !x)} aria-label={showPassword ? '隐藏密码' : '显示密码'}>{showPassword ? <EyeOff size={15} /> : <Eye size={15} />}</button>
                </div>
              </label>

              <div className="authFormOptions">
                <label><input type="checkbox" checked={rememberMe} onChange={(e) => setRememberMe(e.target.checked)} /> 记住我</label>
                <a href="/login">忘记密码</a>
              </div>

              {error && <p className="loginError">{error}</p>}

              <button className="loginSubmit" disabled={busy || !username.trim() || !password} onClick={submit}>
                <LogIn size={17} /> {busy ? '登录中...' : `进入 LiveAuction 工作台`}
              </button>

              <p className="loginHint">测试账号：admin / admin_dev_password。登录后按账号绑定权限进入后台。</p>
            </>
          )}
        </article>
      </section>
    </main>
  );
}
