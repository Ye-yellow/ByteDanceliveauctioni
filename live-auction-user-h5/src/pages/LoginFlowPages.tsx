import { useMemo, useState, type FormEvent } from 'react';
import {
  normalizeBuyerUsername,
  validateBuyerCredentials,
} from '../shared/auth/credentialRules';
import { useAuthSession } from '../shared/auth/useAuthSession';
import './login-flow-replica.css';

type SocialLogin = '今日头条登录' | 'QQ登录' | '微信登录' | '微博登录';

const SOCIAL_LOGINS: SocialLogin[] = ['今日头条登录', 'QQ登录', '微信登录', '微博登录'];
const HOT_CITIES = ['深圳', '北京', '上海', '成都', '广州', '重庆', '西安', '苏州', '武汉', '杭州', '郑州', '南京', '合肥', '长沙', '福州'];
const HISTORY_CITIES = ['大连', '深圳', '北京', '上海'];
const CITY_GROUPS = {
  A: ['安庆', '澳门', '鞍山', '安阳'],
  B: ['北京', '保定', '包头', '蚌埠', '北海'],
  C: ['成都', '重庆', '长沙', '长春', '常州'],
  D: ['大连', '东莞', '德阳', '大庆'],
  F: ['福州', '佛山', '抚州'],
  G: ['广州', '贵阳', '桂林'],
  H: ['杭州', '合肥', '哈尔滨', '海口', '惠州'],
  J: ['济南', '嘉兴', '金华', '江门'],
  N: ['南京', '南昌', '南宁', '宁波'],
  Q: ['青岛', '泉州'],
  S: ['上海', '深圳', '苏州', '沈阳', '石家庄'],
  W: ['武汉', '无锡', '乌鲁木齐'],
  X: ['西安', '厦门', '徐州'],
  Z: ['郑州', '珠海', '中山'],
};

function goBack(fallback = '/login') {
  if (window.history.length > 1) {
    window.history.back();
    return;
  }
  window.location.assign(fallback);
}

function Agreement({
  agreed,
  shake,
  tooltip,
  onChange,
}: {
  agreed: boolean;
  shake: boolean;
  tooltip: boolean;
  onChange: (value: boolean) => void;
}) {
  return (
    <section className={`dyLoginFlowProtocol${shake ? ' isShake' : ''}`}>
      {tooltip ? <em>请先勾选协议</em> : null}
      <label>
        <input type="checkbox" checked={agreed} onChange={(event) => onChange(event.target.checked)} />
        <span>
          已阅读并同意 <a href="/service-protocol">用户协议</a> 和 <a href="/service-protocol">隐私政策</a>
          ，同时登录并使用抖音火山版（原“火山小视频”）和抖音
        </span>
      </label>
    </section>
  );
}

function useAgreementGuard() {
  const [agreed, setAgreed] = useState(false);
  const [shake, setShake] = useState(false);
  const [tooltip, setTooltip] = useState(false);

  function requireAgreement() {
    if (agreed) return true;
    if (!shake && !tooltip) {
      setShake(true);
      window.setTimeout(() => {
        setShake(false);
        setTooltip(true);
      }, 450);
      window.setTimeout(() => setTooltip(false), 3000);
    }
    return false;
  }

  return { agreed, setAgreed, shake, tooltip, requireAgreement };
}

function Header({ title, right = '帮助与设置', fallback = '/login' }: { title?: string; right?: string; fallback?: string }) {
  return (
    <header className="dyLoginFlowHeader">
      <button type="button" aria-label="返回" onClick={() => goBack(fallback)}>‹</button>
      <b>{title || ''}</b>
      {right ? <a href="/login/help">{right}</a> : <span />}
    </header>
  );
}

function LoginInput({
  type = 'text',
  value,
  onChange,
  placeholder,
  autoFocus,
  action,
}: {
  type?: string;
  value: string;
  onChange: (value: string) => void;
  placeholder: string;
  autoFocus?: boolean;
  action?: React.ReactNode;
}) {
  return (
    <label className="dyLoginFlowInput">
      <input type={type} value={value} onChange={(event) => onChange(event.target.value)} placeholder={placeholder} autoFocus={autoFocus} />
      {action}
    </label>
  );
}

function SocialSheet({ onClose }: { onClose: () => void }) {
  return (
    <section className="dyLoginFlowSheetMask" onClick={onClose}>
      <div className="dyLoginFlowSocialSheet" role="dialog" aria-modal="true" aria-label="其他登录方式" onClick={(event) => event.stopPropagation()}>
        {SOCIAL_LOGINS.map((item) => (
          <button type="button" key={item}>
            <span>{item.slice(0, 1)}</span>
            {item}
          </button>
        ))}
        <i />
        <button type="button" onClick={onClose}>取消</button>
      </div>
    </section>
  );
}

export function OtherLoginPage() {
  const [phone, setPhone] = useState('');
  const [loading, setLoading] = useState(false);
  const [notice, setNotice] = useState('');
  const [socialOpen, setSocialOpen] = useState(false);
  const guard = useAgreementGuard();

  function getCode() {
    if (!guard.requireAgreement()) return;
    if (phone.length < 10) {
      setNotice('请输入正确的手机号');
      return;
    }
    setLoading(true);
    window.setTimeout(() => {
      window.location.assign(`/login/verification-code?phone=${encodeURIComponent(phone)}`);
    }, 450);
  }

  function openSocial() {
    if (guard.requireAgreement()) setSocialOpen(true);
  }

  return (
    <main className="dyLoginFlowPage">
      <Header />
      <section className="dyLoginFlowContent">
        <section className="dyLoginFlowDesc">
          <h1>登录看朋友内容</h1>
          <p>未注册的手机号验证通过后将自动注册</p>
        </section>
        <LoginInput value={phone} onChange={setPhone} placeholder="请输入手机号" autoFocus />
        {notice ? <p className="dyLoginFlowNotice">{notice}</p> : null}
        <Agreement agreed={guard.agreed} shake={guard.shake} tooltip={guard.tooltip} onChange={guard.setAgreed} />
        <button type="button" className="dyLoginFlowPrimary" disabled={phone.length < 10 || loading} onClick={getCode}>
          {loading ? '发送中' : '获取短信验证码'}
        </button>
        <footer className="dyLoginFlowOptions">
          <a href="/login/password">密码登录</a>
          <button type="button" onClick={openSocial}>其他方式登录</button>
        </footer>
      </section>
      {socialOpen ? <SocialSheet onClose={() => setSocialOpen(false)} /> : null}
    </main>
  );
}

export function PasswordLoginPage() {
  const { loginBuyer, reason } = useAuthSession();
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [loading, setLoading] = useState(false);
  const [notice, setNotice] = useState('');
  const guard = useAgreementGuard();

  async function submit(event: FormEvent) {
    event.preventDefault();
    if (loading || !guard.requireAgreement()) return;
    const error = validateBuyerCredentials({ username, password });
    if (error) {
      setNotice(error);
      return;
    }
    setLoading(true);
    setNotice('');
    try {
      await loginBuyer(normalizeBuyerUsername(username), password);
      window.location.assign('/home');
    } catch (e) {
      setNotice(e instanceof Error ? e.message : '登录失败，请重试');
    } finally {
      setLoading(false);
    }
  }

  return (
    <main className="dyLoginFlowPage">
      <Header />
      <form className="dyLoginFlowContent" onSubmit={submit}>
        <section className="dyLoginFlowDesc compact">
          <h1>手机号密码登录</h1>
        </section>
        <LoginInput value={username} onChange={setUsername} placeholder="请输入手机号" autoFocus />
        <LoginInput type="password" value={password} onChange={setPassword} placeholder="请输入密码" />
        <Agreement agreed={guard.agreed} shake={guard.shake} tooltip={guard.tooltip} onChange={guard.setAgreed} />
        {notice || reason ? <p className="dyLoginFlowNotice">{notice || reason}</p> : null}
        <button type="submit" className="dyLoginFlowPrimary" disabled={!username || !password || loading}>
          {loading ? '登录中' : '登录'}
        </button>
        <footer className="dyLoginFlowOptions single">
          <span>忘记了？<a href="/login/retrieve-password">找回密码</a></span>
        </footer>
      </form>
    </main>
  );
}

export function VerificationCodePage() {
  const params = new URLSearchParams(location.search);
  const phone = params.get('phone') || '13800138000';
  const [code, setCode] = useState('');
  const [sent, setSent] = useState(true);
  const [voiceVisible, setVoiceVisible] = useState(false);
  const [notice, setNotice] = useState('');
  const [loading, setLoading] = useState(false);

  function sendCode() {
    setSent(false);
    window.setTimeout(() => setSent(true), 500);
  }

  function login() {
    setLoading(true);
    window.setTimeout(() => {
      setLoading(false);
      setNotice('当前项目未接入短信验证码后端，请使用密码登录。');
    }, 650);
  }

  return (
    <main className="dyLoginFlowPage">
      <Header />
      <section className="dyLoginFlowContent">
        <section className="dyLoginFlowDesc compact">
          <h1>请输入验证码</h1>
          <p>验证码已通过短信发送到+86 {phone}</p>
        </section>
        <LoginInput
          value={code}
          onChange={setCode}
          placeholder="请输入验证码"
          autoFocus
          action={<button type="button" onClick={sendCode}>{sent ? '重新发送' : '发送中'}</button>}
        />
        <footer className="dyLoginFlowOptions single">
          <span>收不到短信？<button type="button" onClick={() => setVoiceVisible(true)}>获取语音验证码</button></span>
        </footer>
        <button type="button" className="dyLoginFlowPrimary topGap" disabled={code.length < 4 || loading} onClick={login}>
          {loading ? '登录中' : '登录'}
        </button>
        {notice ? <p className="dyLoginFlowNotice">{notice} <a href="/login/password">去密码登录</a></p> : null}
      </section>
      {voiceVisible ? (
        <section className="dyLoginFlowDialogMask">
          <div className="dyLoginFlowDialog">
            <h2>语音验证码</h2>
            <p>我们将以电话的方式告知你验证码，请注意接听。</p>
            <button type="button" onClick={() => setVoiceVisible(false)}>知道了</button>
          </div>
        </section>
      ) : null}
    </main>
  );
}

export function RetrievePasswordPage() {
  const { resetBuyerPassword, reason } = useAuthSession();
  const [username, setUsername] = useState('');
  const [code, setCode] = useState('');
  const [password, setPassword] = useState('');
  const [loading, setLoading] = useState(false);
  const [notice, setNotice] = useState('');
  const [voiceVisible, setVoiceVisible] = useState(false);
  const guard = useAgreementGuard();

  async function submit(event: FormEvent) {
    event.preventDefault();
    if (loading || !guard.requireAgreement()) return;
    const error = validateBuyerCredentials({ username, password });
    if (error) {
      setNotice(error);
      return;
    }
    if (code.length < 4) {
      setNotice('请输入验证码');
      return;
    }
    setLoading(true);
    setNotice('');
    try {
      await resetBuyerPassword(normalizeBuyerUsername(username), password);
      setNotice('密码已重置，请返回密码登录。');
    } catch (e) {
      setNotice(e instanceof Error ? e.message : '重置失败，请重试');
    } finally {
      setLoading(false);
    }
  }

  return (
    <main className="dyLoginFlowPage">
      <Header right="" />
      <form className="dyLoginFlowContent" onSubmit={submit}>
        <section className="dyLoginFlowDesc compact">
          <h1>找回密码</h1>
          <p>验证码已通过短信发送。密码为8 - 20位，至少包含字母、数字、符号2种组合</p>
        </section>
        <LoginInput value={username} onChange={setUsername} placeholder="请输入手机号" autoFocus />
        <LoginInput value={code} onChange={setCode} placeholder="请输入验证码" action={<button type="button">重新发送</button>} />
        <LoginInput type="password" value={password} onChange={setPassword} placeholder="请输入密码" />
        <Agreement agreed={guard.agreed} shake={guard.shake} tooltip={guard.tooltip} onChange={guard.setAgreed} />
        {notice || reason ? <p className="dyLoginFlowNotice">{notice || reason}</p> : null}
        <button type="submit" className="dyLoginFlowPrimary" disabled={!username || !code || !password || loading}>
          {loading ? '处理中' : '完成'}
        </button>
        <footer className="dyLoginFlowOptions single">
          <span>收不到短信？<button type="button" onClick={() => setVoiceVisible(true)}>获取语音验证码</button></span>
        </footer>
      </form>
      {voiceVisible ? (
        <section className="dyLoginFlowDialogMask">
          <div className="dyLoginFlowDialog">
            <h2>语音验证码</h2>
            <p>我们将以电话的方式告知你验证码，请注意接听。</p>
            <button type="button" onClick={() => setVoiceVisible(false)}>知道了</button>
          </div>
        </section>
      ) : null}
    </main>
  );
}

export function LoginHelpPage() {
  const helpItems = [
    ['账号登录', '支持 live auction 账号密码登录，手机号登录页会统一跳转到本地账号体系。'],
    ['协议与隐私', '用户协议和隐私政策保留本地页面，不再依赖外部客服 iframe。'],
    ['反馈入口', '遇到登录异常时，先检查账号、密码和当前网络，再联系后台管理员处理。'],
  ];

  return (
    <main className="dyLoginFlowHelp">
      <header className="dyLoginFlowHelpHeader">
        <button type="button" onClick={() => goBack('/login')}>‹</button>
        <b>帮助与设置</b>
        <span>我的反馈</span>
      </header>
      <section className="dyLoginFlowHelpContent">
        <h1>登录帮助</h1>
        <p>当前 H5 使用 live auction 的账号密码登录，不再跳转第三方客服页面。</p>
        <div className="dyLoginFlowHelpList">
          {helpItems.map(([title, desc]) => (
            <article key={title}>
              <b>{title}</b>
              <span>{desc}</span>
            </article>
          ))}
        </div>
        <a href="/login" className="dyLoginFlowHelpAction">返回登录</a>
      </section>
    </main>
  );
}

export function CountryChoosePage() {
  const [query, setQuery] = useState('');
  const filteredGroups = useMemo(() => {
    const keyword = query.trim();
    if (!keyword) return CITY_GROUPS;
    return Object.fromEntries(
      Object.entries(CITY_GROUPS)
        .map(([letter, cities]) => [letter, cities.filter((city) => city.includes(keyword))])
        .filter(([, cities]) => cities.length > 0),
    ) as typeof CITY_GROUPS;
  }, [query]);

  return (
    <main className="dyCountryChoosePage">
      <header className="dyCountryHeader">
        <button type="button" onClick={() => goBack('/me/edit-userinfo')}>‹</button>
        <b>切换城市</b>
        <span />
        <label>
          <span>⌕</span>
          <input value={query} onChange={(event) => setQuery(event.target.value)} placeholder="输入城市或区县名搜索" />
          {query ? <button type="button" onClick={() => setQuery('')}>取消</button> : null}
        </label>
      </header>
      <section className="dyCountryContent">
        <div className="dyCountryRow muted">默认位置</div>
        <button type="button" className="dyCountryRow">同城</button>
        <section className="dyCountryTopBlock">
          <p>历史访问</p>
          <div>{HISTORY_CITIES.map((city) => <button type="button" key={city}>{city}</button>)}</div>
        </section>
        <section className="dyCountryTopBlock">
          <p>抖单热门城市</p>
          <div>{HOT_CITIES.map((city) => <button type="button" key={city}>{city}</button>)}</div>
        </section>
        {Object.entries(filteredGroups).map(([letter, cities]) => (
          <section className="dyCountryGroup" id={`city-${letter}`} key={letter}>
            <p>{letter}</p>
            {cities.map((city) => <button type="button" key={city}>{city}</button>)}
          </section>
        ))}
      </section>
      <nav className="dyCountryIndicator" aria-label="城市索引">
        <a href="#top">⌃</a>
        {'ABCDEFGHIJKLMNOPQRSTUVWXYZ'.split('').map((letter) => <a href={`#city-${letter}`} key={letter}>{letter}</a>)}
      </nav>
    </main>
  );
}
