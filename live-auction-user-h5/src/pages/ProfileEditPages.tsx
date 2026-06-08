import { useState, type ReactNode } from 'react';
import { useAuthSession } from '../shared/auth/useAuthSession';
import './profile-edit-replica.css';

type Row = {
  title: string;
  value?: string;
  href?: string;
  onClick?: () => void;
};

const SCHOOLS = Array.from({ length: 20 }, (_, index) => `所有大学${index}`);
const NEARBY_SCHOOLS = Array.from({ length: 20 }, (_, index) => `附近大学${index}`);
const DEPARTMENTS = Array.from({ length: 5 }, (_, index) => `院系${index}`);
const PROVINCES = ['河北', '山西', '辽宁', '吉林', '黑龙江', '江苏', '浙江', '安徽', '福建', '江西', '山东', '河南', '湖北', '湖南', '广东', '海南', '四川', '贵州', '云南', '陕西', '甘肃', '青海', '台湾', '内蒙古', '广西', '西藏', '宁夏', '新疆', '北京', '天津', '上海', '重庆', '香港', '澳门'];
const CITIES = ['成都', '深圳', '广州', '北京', '上海', '杭州', '南京', '苏州', '武汉', '西安', '重庆', '厦门'];
const DISPLAY_TYPES = [
  { label: '公开可见', value: '1' },
  { label: '校友可见', value: '2' },
  { label: '仅自己可见', value: '3' },
];

function goBack(fallback = '/me') {
  if (window.history.length > 1) {
    window.history.back();
    return;
  }
  window.location.assign(fallback);
}

function Header({
  title,
  subtitle,
  right,
  onBack,
}: {
  title: string;
  subtitle?: string;
  right?: ReactNode;
  onBack?: () => void;
}) {
  return (
    <header className="dyProfileEditHeader">
      <button type="button" aria-label="返回" onClick={onBack || (() => goBack())}>‹</button>
      <span>
        <b>{title}</b>
        {subtitle ? <small>{subtitle}</small> : null}
      </span>
      <div>{right}</div>
    </header>
  );
}

function RowList({ rows }: { rows: Row[] }) {
  return (
    <section className="dyProfileEditRows">
      {rows.map((row) => {
        const content = (
          <>
            <span>{row.title}</span>
            <p>{row.value || '点击设置'}<i>›</i></p>
          </>
        );
        if (row.href) return <a href={row.href} key={row.title}>{content}</a>;
        return <button type="button" onClick={row.onClick} key={row.title}>{content}</button>;
      })}
    </section>
  );
}

function SelectSheet({
  title,
  options,
  onSelect,
  onClose,
}: {
  title: string;
  options: string[];
  onSelect: (value: string) => void;
  onClose: () => void;
}) {
  return (
    <section className="dyProfileEditSheetMask" onClick={onClose}>
      <div className="dyProfileEditSheet" role="dialog" aria-modal="true" aria-label={title} onClick={(event) => event.stopPropagation()}>
        <b>{title}</b>
        {options.map((option) => (
          <button type="button" onClick={() => onSelect(option)} key={option}>{option}</button>
        ))}
        <i />
        <button type="button" onClick={onClose}>取消</button>
      </div>
    </section>
  );
}

function displayNameFromUser(user: ReturnType<typeof useAuthSession>['user']) {
  return user?.nickname?.trim() || user?.username?.trim() || '透明人';
}

function readSchoolDraft() {
  return {
    school: localStorage.getItem('changeSchool') || '',
    department: localStorage.getItem('changeDepartment') || '',
    joinTime: localStorage.getItem('changeJoinTime') || '',
    education: localStorage.getItem('changeEducation') || '',
    displayType: localStorage.getItem('changeDisplayType') || '1',
  };
}

export function EditUserInfoPage() {
  const { user } = useAuthSession();
  const [sex, setSex] = useState('');
  const [birthday, setBirthday] = useState('');
  const [sheet, setSheet] = useState<'avatar' | 'sex' | 'birthday' | null>(null);
  const name = displayNameFromUser(user);
  const douyinId = user?.username || 'douyin_react_h5';

  const rows: Row[] = [
    { title: '名字', value: name, href: '/me/edit-userinfo-item?type=1' },
    { title: '抖音号', value: douyinId, href: '/me/edit-userinfo-item?type=2' },
    { title: '简介', value: '点击设置', href: '/me/edit-userinfo-item?type=3' },
    { title: '性别', value: sex, onClick: () => setSheet('sex') },
    { title: '生日', value: birthday, onClick: () => setSheet('birthday') },
    { title: '所在地', value: '广东 - 深圳', href: '/me/choose-location' },
    { title: '学校', value: localStorage.getItem('changeSchool') || '', href: '/me/add-school' },
  ];

  return (
    <main className="dyProfileEditPage">
      <Header title="编辑资料" subtitle="已完成85%" />
      <section className="dyProfileEditAvatarBlock">
        <button type="button" onClick={() => setSheet('avatar')}>
          <span>{name.slice(0, 1)}</span>
          <i>⌕</i>
        </button>
        <p>点击更换头像</p>
      </section>
      <RowList rows={rows} />
      {sheet === 'avatar' ? (
        <SelectSheet
          title="更换头像"
          options={['拍一张', '从相册选择', '查看大图']}
          onSelect={(value) => {
            if (value === '查看大图') setSheet(null);
            else setSheet(null);
          }}
          onClose={() => setSheet(null)}
        />
      ) : null}
      {sheet === 'sex' ? (
        <SelectSheet
          title="性别"
          options={['男', '女', '不展示']}
          onSelect={(value) => {
            setSex(value === '不展示' ? '' : value);
            setSheet(null);
          }}
          onClose={() => setSheet(null)}
        />
      ) : null}
      {sheet === 'birthday' ? (
        <SelectSheet
          title="生日"
          options={['2000-01-01', '1998-06-18', '1995-10-10', '不展示']}
          onSelect={(value) => {
            setBirthday(value === '不展示' ? '' : value);
            setSheet(null);
          }}
          onClose={() => setSheet(null)}
        />
      ) : null}
    </main>
  );
}

export function EditUserInfoItemPage() {
  const { user } = useAuthSession();
  const params = new URLSearchParams(location.search);
  const type = params.get('type') || '1';
  const title = type === '2' ? '修改抖音号' : type === '3' ? '修改简介' : '修改名字';
  const initialValue = type === '2' ? user?.username || 'douyin_react_h5' : type === '3' ? '' : displayNameFromUser(user);
  const [value, setValue] = useState(initialValue);
  const [confirmOpen, setConfirmOpen] = useState(false);
  const changed = value.trim() !== initialValue.trim() && (type !== '1' || value.trim().length > 0);

  function back() {
    if (changed) setConfirmOpen(true);
    else goBack('/me/edit-userinfo');
  }

  function save() {
    if (!changed) return;
    localStorage.setItem(`profileEditItem:${type}`, value.trim());
    goBack('/me/edit-userinfo');
  }

  return (
    <main className="dyProfileEditPage">
      <Header title={title} onBack={back} right={<button type="button" className={changed ? 'saveYes' : 'saveNo'} onClick={save}>保存</button>} />
      <section className="dyProfileEditItemContent">
        {type === '3' ? (
          <>
            <p>个人简介</p>
            <textarea value={value} onChange={(event) => setValue(event.target.value)} placeholder="你可以填写兴趣爱好、心情愿望，有趣的介绍能让被关注的概率变高噢！" autoFocus />
          </>
        ) : (
          <>
            <p>{type === '2' ? '我的抖音号' : '我的名字'}</p>
            <label>
              <input value={value} onChange={(event) => setValue(event.target.value)} placeholder={type === '2' ? '请输入抖音号' : '记得填写名字哦'} autoFocus maxLength={type === '1' ? 20 : 16} />
              {value ? <button type="button" onClick={() => setValue('')}>×</button> : null}
            </label>
            <small>{type === '2' ? '最多16个字，只允许包含字母、数字、下划线和点，30天内仅能修改一次' : `${value.length}/20`}</small>
          </>
        )}
      </section>
      {confirmOpen ? (
        <section className="dyProfileEditDialogMask">
          <div className="dyProfileEditDialog">
            <p>是否保存修改</p>
            <button type="button" onClick={save}>保存</button>
            <button type="button" onClick={() => goBack('/me/edit-userinfo')}>不保存</button>
          </div>
        </section>
      ) : null}
    </main>
  );
}

export function AddSchoolPage() {
  const [draft, setDraft] = useState(readSchoolDraft);
  const [sheet, setSheet] = useState<'join' | 'education' | null>(null);
  const displayType = DISPLAY_TYPES.find((item) => item.value === draft.displayType)?.label || '公开可见';
  const changed = Object.values(draft).some(Boolean);

  function refresh() {
    setDraft(readSchoolDraft());
  }

  function save() {
    localStorage.setItem('profileSchoolSaved', JSON.stringify(readSchoolDraft()));
    localStorage.removeItem('changeDepartment');
    localStorage.removeItem('changeDisplayType');
    localStorage.removeItem('changeEducation');
    localStorage.removeItem('changeJoinTime');
    goBack('/me/edit-userinfo');
  }

  return (
    <main className="dyProfileEditPage" onFocus={refresh}>
      <Header title="添加学校" right={<button type="button" className={changed ? 'saveYes' : 'saveNo'} onClick={save}>保存</button>} />
      <RowList rows={[
        { title: '学校', value: draft.school, href: '/me/choose-school' },
        { title: '院系', value: draft.department, href: '/me/choose-department' },
        { title: '入学时间', value: draft.joinTime, onClick: () => setSheet('join') },
        { title: '学历', value: draft.education, onClick: () => setSheet('education') },
        { title: '展示范围', value: displayType, href: `/me/display-type?displayType=${draft.displayType}` },
      ]} />
      {sheet === 'join' ? (
        <SelectSheet
          title="入学时间"
          options={Array.from({ length: 8 }, (_, index) => `${new Date().getFullYear() - index}`)}
          onSelect={(value) => {
            localStorage.setItem('changeJoinTime', value);
            setDraft(readSchoolDraft());
            setSheet(null);
          }}
          onClose={() => setSheet(null)}
        />
      ) : null}
      {sheet === 'education' ? (
        <SelectSheet
          title="学历"
          options={['专科', '本科', '硕士', '博士']}
          onSelect={(value) => {
            localStorage.setItem('changeEducation', value);
            setDraft(readSchoolDraft());
            setSheet(null);
          }}
          onClose={() => setSheet(null)}
        />
      ) : null}
    </main>
  );
}

export function ChooseSchoolPage() {
  const [query, setQuery] = useState('');
  const [searched, setSearched] = useState(false);
  const allSchools = NEARBY_SCHOOLS.concat(SCHOOLS);
  const results = allSchools.filter((school) => school.includes(query));

  function setSchool(school: string) {
    localStorage.setItem('changeSchool', school);
    goBack('/me/add-school');
  }

  return (
    <main className="dyProfileEditPage">
      <Header title="添加学校" right={<a className="dyProfileEditRightLink" href="/me/declare-school?type=1">没有找到?</a>} />
      <section className="dyProfileEditSearch">
        <label><span>⌕</span><input value={query} onChange={(event) => setQuery(event.target.value)} placeholder="搜索大学名称" /></label>
        <button type="button" onClick={() => setSearched(Boolean(query))}>搜索</button>
      </section>
      <section className="dyProfileChooseList hasSearch">
        {!searched ? (
          <>
            <p>离我最近</p>
            {NEARBY_SCHOOLS.length ? NEARBY_SCHOOLS.slice(0, 4).map((school) => <button type="button" onClick={() => setSchool(school)} key={school}>{school}</button>) : <button type="button">无法获取</button>}
            <i />
            {SCHOOLS.map((school) => <button type="button" onClick={() => setSchool(school)} key={school}>{school}</button>)}
          </>
        ) : results.length ? (
          results.map((school) => <button type="button" onClick={() => setSchool(school)} key={school}>{school}</button>)
        ) : (
          <section className="dyProfileEditEmpty"><span /> <b>搜索结果为空</b><p>没有搜索到相关的内容</p><a href="/me/declare-school">没有学校信息？去申报</a></section>
        )}
      </section>
    </main>
  );
}

export function ChooseDepartmentPage() {
  function setDepartment(department: string) {
    localStorage.setItem('changeDepartment', department);
    goBack('/me/add-school');
  }

  return (
    <main className="dyProfileEditPage">
      <Header title="选择院系" right={<a className="dyProfileEditRightLink" href="/me/declare-school?type=2">没有找到?</a>} />
      <section className="dyProfileChooseList">
        {DEPARTMENTS.map((department) => <button type="button" onClick={() => setDepartment(department)} key={department}>{department}</button>)}
      </section>
    </main>
  );
}

export function DisplayTypePage() {
  const params = new URLSearchParams(location.search);
  const [selected, setSelected] = useState(params.get('displayType') || '1');

  function select(value: string) {
    setSelected(value);
    localStorage.setItem('changeDisplayType', value);
    window.setTimeout(() => goBack('/me/add-school'), 120);
  }

  return (
    <main className="dyProfileEditPage">
      <Header title="展示范围" />
      <section className="dyProfileChooseList">
        {DISPLAY_TYPES.map((item) => (
          <button type="button" onClick={() => select(item.value)} key={item.value}>
            {item.label}
            {selected === item.value ? <span className="dyProfileEditCheck">✓</span> : null}
          </button>
        ))}
      </section>
    </main>
  );
}

export function ChooseLocationPage() {
  return (
    <main className="dyProfileEditPage">
      <Header title="选择地区" />
      <section className="dyProfileChooseList">
        <button type="button" onClick={() => goBack('/me/edit-userinfo')}>暂不设置</button>
        <i />
        <p>当前位置</p>
        <button type="button">无法获取</button>
        <i />
        <p>其他地区</p>
        <a href="/me/choose-province">中国 <span>›</span></a>
      </section>
    </main>
  );
}

export function ChooseProvincePage() {
  return (
    <main className="dyProfileEditPage">
      <Header title="中国" />
      <section className="dyProfileChooseList">
        {PROVINCES.map((province) => <a href={`/me/choose-city?province=${encodeURIComponent(province)}`} key={province}>{province}<span>›</span></a>)}
      </section>
    </main>
  );
}

export function ChooseCityPage() {
  const province = new URLSearchParams(location.search).get('province') || '四川';

  function save(city: string) {
    localStorage.setItem('profileLocation', `中国-${province}-${city}`);
    window.history.go(-3);
  }

  return (
    <main className="dyProfileEditPage">
      <Header title={province} />
      <section className="dyProfileChooseList">
        {CITIES.map((city) => <button type="button" onClick={() => save(city)} key={city}>{city}</button>)}
      </section>
    </main>
  );
}
