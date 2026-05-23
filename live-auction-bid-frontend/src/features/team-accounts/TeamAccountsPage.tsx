import { useEffect, useMemo, useState } from 'react';
import { AlertTriangle, CheckCircle2, RefreshCw, Search, ShieldAlert, ShieldCheck, UserCog, Users } from 'lucide-react';
import { currentAuth } from '../auth/api/authApi';
import { adminCreateUser, adminUpdateUserRole, listAdminUsers, type AdminUsersQuery } from '../admin-user/api/adminUserApi';
import { USER_ROLE, type User, type UserRole } from '../../shared/api/types';
import { resultMessage } from '../../shared/api/result';
import { formatDateTimeText } from '../../shared/lib/format';
import { USER_ROLE_FILTERS, USER_ROLE_OPTIONS, userRoleMeta } from '../../entities/user/model/userRole';
import { StudioBadge, StudioButton, StudioCard, StudioEmptyState, StudioField, StudioMetricCard, StudioPageHeader, StudioTable, StudioTableSkeleton, StudioToastViewport, useStudioToast } from '../../pages/host-console/components/studio-ui';

const DEFAULT_PAGE_SIZE = 20;
const BACKOFFICE_ROLES: UserRole[] = [USER_ROLE.ADMIN, USER_ROLE.ANCHOR, USER_ROLE.OPERATOR];

export function TeamAccountsPage() {
  const currentUser = currentAuth().user;
  const isAdmin = currentUser?.role === USER_ROLE.ADMIN;
  const [query, setQuery] = useState<AdminUsersQuery>({ page: 1, pageSize: DEFAULT_PAGE_SIZE });
  const [users, setUsers] = useState<User[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [createForm, setCreateForm] = useState({ username: '', password: '', nickname: '', role: USER_ROLE.OPERATOR as UserRole });
  const [updateForm, setUpdateForm] = useState({ userId: '', role: USER_ROLE.OPERATOR as UserRole });
  const [creating, setCreating] = useState(false);
  const [updating, setUpdating] = useState(false);
  const { toasts, showToast } = useStudioToast();

  const totalPages = Math.max(1, Math.ceil(total / (query.pageSize || DEFAULT_PAGE_SIZE)));

  const syncUsers = async (nextQuery = query) => {
    if (!isAdmin) return;
    setLoading(true);
    setError('');
    try {
      const page = await listAdminUsers(nextQuery);
      setUsers(page.users);
      setTotal(page.total);
      setQuery((current) => ({ ...current, page: page.page, pageSize: page.pageSize }));
    } catch (e) {
      const message = resultMessage(e);
      setError(message);
      showToast({ id: 'admin-users-sync-failed', tone: 'danger', title: '账号列表同步失败', description: message });
    } finally {
      setLoading(false);
    }
  };

  const updateQuery = (patch: Partial<AdminUsersQuery>) => {
    setQuery((current) => ({ ...current, ...patch, page: patch.page ?? 1 }));
  };

  const submitCreate = async () => {
    if (!isAdmin || !createForm.username.trim() || !createForm.password || !createForm.nickname.trim()) return;
    setCreating(true);
    setError('');
    try {
      const user = await adminCreateUser({ ...createForm, username: createForm.username.trim(), nickname: createForm.nickname.trim() });
      setCreateForm({ username: '', password: '', nickname: '', role: USER_ROLE.OPERATOR });
      showToast({ tone: 'success', title: '账号已创建', description: `${user.username} · ${userRoleMeta(user.role).label}` });
      await syncUsers({ ...query, page: 1 });
    } catch (e) {
      const message = resultMessage(e);
      setError(message);
      showToast({ tone: 'danger', title: '创建账号失败', description: message });
    } finally {
      setCreating(false);
    }
  };

  const submitUpdate = async () => {
    if (!isAdmin || !updateForm.userId.trim()) return;
    setUpdating(true);
    setError('');
    try {
      const user = await adminUpdateUserRole(updateForm.userId.trim(), updateForm.role);
      setUpdateForm({ userId: '', role: USER_ROLE.OPERATOR });
      showToast({ tone: 'success', title: '账号角色已更新', description: `${user.username} · ${userRoleMeta(user.role).label}` });
      await syncUsers(query);
    } catch (e) {
      const message = resultMessage(e);
      setError(message);
      showToast({ tone: 'danger', title: '更新角色失败', description: message });
    } finally {
      setUpdating(false);
    }
  };

  useEffect(() => { void syncUsers(); }, [isAdmin]);
  useEffect(() => { void syncUsers(query); }, [query.page, query.role]);

  const metrics = useMemo(() => ({
    admins: users.filter((user) => user.role === USER_ROLE.ADMIN).length,
    backoffice: users.filter((user) => BACKOFFICE_ROLES.includes(user.role)).length,
    buyers: users.filter((user) => user.role === USER_ROLE.BUYER).length,
  }), [users]);

  return <section className="teamAccountPage">
    <StudioToastViewport toasts={toasts} />
    <section className="laSettingsHero laMerchantsHero"><StudioPageHeader eyebrow="Admin users" title="团队协作" description="账号列表、角色筛选和搜索来自 /api/admin/users；创建和改角色仅 USER_ROLE_ADMIN 可提交。" actions={<StudioButton type="button" variant="secondary" icon={<RefreshCw size={15} />} loading={loading} disabled={!isAdmin} onClick={() => void syncUsers()}>{loading ? '同步中' : '同步账号'}</StudioButton>} /></section>
    {error ? <div className="auctionMgmtNotice danger"><AlertTriangle size={16} />{error}</div> : null}
    {!isAdmin ? <NonAdminAccountNotice currentUser={currentUser} /> : <>
      <section className="laStatsGrid">
        <StudioMetricCard icon={<ShieldCheck />} label="账号总数" value={total} trend="ADMIN 查询接口" tone="info" />
        <StudioMetricCard icon={<UserCog />} label="后台角色" value={metrics.backoffice} trend="ADMIN / ANCHOR / OPERATOR" tone="success" />
        <StudioMetricCard icon={<ShieldAlert />} label="管理员" value={metrics.admins} trend="可管理账号" tone="purple" />
        <StudioMetricCard icon={<Users />} label="买家账号" value={metrics.buyers} trend="BUYER 禁止进入后台" tone="warning" />
      </section>
      <StudioCard padding="md">
        <div className="auctionFilterBar queueFilters" aria-label="账号筛选">
          <label><Search size={15} /><input value={query.keyword || ''} onChange={(e) => updateQuery({ keyword: e.target.value })} onKeyDown={(e) => { if (e.key === 'Enter') void syncUsers({ ...query, page: 1 }); }} placeholder="搜索用户 ID / 用户名 / 昵称" /></label>
          <StudioField label="角色"><select value={query.role || ''} onChange={(e) => updateQuery({ role: e.target.value as AdminUsersQuery['role'] })}>{USER_ROLE_FILTERS.map((item) => <option key={item.label} value={item.value}>{item.label}</option>)}</select></StudioField>
          <StudioButton type="button" variant="primary" icon={<Search size={15} />} onClick={() => void syncUsers({ ...query, page: 1 })}>查询</StudioButton>
        </div>
      </StudioCard>
      <section className="laGrid laGrid-1-1">
        <StudioCard title="创建团队账号" subtitle="/api/admin/users" padding="md">
          <div className="laFormGrid">
            <StudioField label="用户名"><input value={createForm.username} onChange={(e) => setCreateForm({ ...createForm, username: e.target.value })} placeholder="team_operator_01" autoComplete="off" /></StudioField>
            <StudioField label="昵称"><input value={createForm.nickname} onChange={(e) => setCreateForm({ ...createForm, nickname: e.target.value })} placeholder="运营账号昵称" /></StudioField>
            <StudioField label="初始密码"><input value={createForm.password} onChange={(e) => setCreateForm({ ...createForm, password: e.target.value })} placeholder="输入后端允许的密码" type="password" autoComplete="new-password" /></StudioField>
            <StudioField label="角色"><select value={createForm.role} onChange={(e) => setCreateForm({ ...createForm, role: e.target.value as UserRole })}>{USER_ROLE_OPTIONS.map((item) => <option key={item.role} value={item.role}>{item.label}</option>)}</select></StudioField>
          </div>
          <div className="laRulePreview">{USER_ROLE_OPTIONS.map((item) => <span key={item.role}>{item.label} · {item.hint}</span>)}</div>
          <div className="drawerActions"><StudioButton type="button" variant="primary" loading={creating} disabled={!createForm.username.trim() || !createForm.password || !createForm.nickname.trim()} onClick={() => void submitCreate()}>创建账号</StudioButton></div>
        </StudioCard>
        <StudioCard title="调整用户角色" subtitle="/api/admin/users/{userId}/role" padding="md">
          <div className="laFormGrid">
            <StudioField label="用户 ID"><input value={updateForm.userId} onChange={(e) => setUpdateForm({ ...updateForm, userId: e.target.value })} placeholder="输入后端 user.id" /></StudioField>
            <StudioField label="新角色"><select value={updateForm.role} onChange={(e) => setUpdateForm({ ...updateForm, role: e.target.value as UserRole })}>{USER_ROLE_OPTIONS.map((item) => <option key={item.role} value={item.role}>{item.label}</option>)}</select></StudioField>
          </div>
          <StudioEmptyState compact icon={<CheckCircle2 size={22} />} title="角色变更后立即刷新列表" description="后端仍负责最终权限校验，页面只在 ADMIN 下展示可提交表单。" />
          <div className="drawerActions"><StudioButton type="button" variant="primary" loading={updating} disabled={!updateForm.userId.trim()} onClick={() => void submitUpdate()}>更新角色</StudioButton></div>
        </StudioCard>
      </section>
      {loading ? <StudioTableSkeleton rows={6} columns={6} /> : <StudioTable
        rows={users}
        rowKey={(user) => user.id}
        header={`共 ${total} 个账号 · 第 ${query.page || 1} / ${totalPages} 页`}
        filters={<div className="postLiveFilters"><span>分页</span><button type="button" disabled={(query.page || 1) <= 1 || loading} onClick={() => setQuery((current) => ({ ...current, page: Math.max(1, (current.page || 1) - 1) }))}>上一页</button><button type="button" disabled={(query.page || 1) >= totalPages || loading} onClick={() => setQuery((current) => ({ ...current, page: (current.page || 1) + 1 }))}>下一页</button></div>}
        empty={<StudioEmptyState compact icon={<Users size={28} />} title="暂无账号" description="当前筛选条件下没有用户。" />}
        columns={[
          { label: '用户', render: (user) => <div><b>{user.username}</b><br /><span>{user.nickname || '未设置昵称'}</span></div> },
          { label: '用户 ID', render: (user) => <code>{user.id}</code> },
          { label: '角色', render: (user) => <StudioBadge tone={userRoleMeta(user.role).tone}>{userRoleMeta(user.role).label}</StudioBadge> },
          { label: '创建时间', render: (user) => formatDateTimeText(user.createdAtUnixMs) },
          { label: '更新时间', render: (user) => formatDateTimeText(user.updatedAtUnixMs) },
          { label: '操作', render: (user) => <div className="laRowActions"><button type="button" onClick={() => setUpdateForm({ userId: user.id, role: user.role })}>填入改角色</button></div> },
        ]}
      />}
    </>}
  </section>;
}

function NonAdminAccountNotice({ currentUser }: { currentUser?: User | null }) {
  return <>
    <section className="laStatsGrid">
      <StudioMetricCard icon={<ShieldCheck />} label="当前后台账号" value={currentUser?.username || '未同步'} trend={currentUser ? userRoleMeta(currentUser.role).label : 'AuthSession'} tone="info" />
      <StudioMetricCard icon={<ShieldAlert />} label="账号管理权限" value="仅管理员" trend="ANCHOR / OPERATOR 不能创建或改角色" tone="danger" />
    </section>
    <StudioCard title="仅管理员可管理账号" subtitle="Admin only" padding="md">
      <StudioEmptyState compact icon={<ShieldAlert size={28} />} title="仅管理员可管理账号" description="当前账号只能查看后台业务功能，不能看到或提交创建用户、修改角色表单。" />
    </StudioCard>
  </>;
}
