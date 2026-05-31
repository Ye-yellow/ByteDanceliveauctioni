import { useEffect, useMemo, useState } from 'react';
import { AlertTriangle, Ban, CheckCircle2, ChevronLeft, ChevronRight, Power, RefreshCw, Search, ShieldAlert, ShieldCheck, UserCog, Users } from 'lucide-react';
import { currentAuth } from '../auth/api/authApi';
import { adminCreateUser, adminUpdateUserRole, adminUpdateUserStatus, listAdminUsers, type AdminUsersQuery } from '../admin-user/api/adminUserApi';
import { hasPermission, isManagedTeamRole, PERMISSION_CODE, ROLE_CODE, USER_STATUS, type RoleCode, type User } from '../../shared/api/types';
import { resultMessage } from '../../shared/api/result';
import { formatDateTimeText } from '../../shared/lib/format';
import { primaryRoleCode, ROLE_CODE_FILTERS, roleCodeMeta, TEAM_ACCOUNT_ROLE_OPTIONS, userStatusMeta } from '../../entities/user/model/userRole';
import { StudioBadge, StudioButton, StudioCard, StudioEmptyState, StudioField, StudioMetricCard, StudioPageHeader, StudioTable, StudioTableSkeleton, StudioToastViewport, useStudioToast } from '../../pages/host-console/components/studio-ui';

const DEFAULT_PAGE_SIZE = 20;
const MANAGED_TEAM_ROLES: RoleCode[] = [ROLE_CODE.ANCHOR, ROLE_CODE.OPERATOR];

export function TeamAccountsPage() {
  const currentUser = currentAuth().user;
  const canListTeam = hasPermission(currentUser, PERMISSION_CODE.TEAM_USER_LIST);
  const canCreateTeam = hasPermission(currentUser, PERMISSION_CODE.TEAM_USER_CREATE);
  const canUpdateTeamRole = hasPermission(currentUser, PERMISSION_CODE.TEAM_USER_UPDATE_ROLE);
  const canUpdateTeamStatus = hasPermission(currentUser, PERMISSION_CODE.TEAM_USER_UPDATE_STATUS);
  const [query, setQuery] = useState<AdminUsersQuery>({ page: 1, pageSize: DEFAULT_PAGE_SIZE });
  const [users, setUsers] = useState<User[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [createForm, setCreateForm] = useState({ username: '', password: '', nickname: '', roleCode: ROLE_CODE.OPERATOR as RoleCode });
  const [updateForm, setUpdateForm] = useState({ userId: '', roleCode: ROLE_CODE.OPERATOR as RoleCode });
  const [creating, setCreating] = useState(false);
  const [updating, setUpdating] = useState(false);
  const [statusUpdatingId, setStatusUpdatingId] = useState('');
  const { toasts, showToast } = useStudioToast();

  const totalPages = Math.max(1, Math.ceil(total / (query.pageSize || DEFAULT_PAGE_SIZE)));
  const currentPage = query.page || 1;

  const goPrevPage = () => setQuery((c) => ({ ...c, page: Math.max(1, (c.page || 1) - 1) }));
  const goNextPage = () => setQuery((c) => ({ ...c, page: (c.page || 1) + 1 }));

  const syncUsers = async (nextQuery = query) => {
    if (!canListTeam) return;
    setLoading(true);
    setError('');
    try {
      const page = await listAdminUsers(nextQuery);
      const outOfScopeUser = page.users.find((user) => !isManagedTeamUser(user));
      if (outOfScopeUser) throw new Error(`团队账号接口返回了非子账号 ${outOfScopeUser.username}，违反主账号空间边界`);
      setUsers(page.users);
      setTotal(page.total);
      setQuery((current) => ({ ...current, page: page.page, pageSize: page.pageSize }));
    } catch (e) {
      const message = resultMessage(e);
      setError(message);
      showToast({ id: 'team-users-sync-failed', tone: 'danger', title: '账号列表同步失败', description: message });
    } finally {
      setLoading(false);
    }
  };

  const updateQuery = (patch: Partial<AdminUsersQuery>) => {
    setQuery((current) => ({ ...current, ...patch, page: patch.page ?? 1 }));
  };

  const submitCreate = async () => {
    if (!canCreateTeam || !createForm.username.trim() || !createForm.password || !createForm.nickname.trim()) return;
    if (!MANAGED_TEAM_ROLES.includes(createForm.roleCode)) {
      setError('主账号只能创建主播 / 场控或运营子账号，不能创建其他主账号或买家账号');
      return;
    }
    setCreating(true);
    setError('');
    try {
      const user = await adminCreateUser({ ...createForm, username: createForm.username.trim(), nickname: createForm.nickname.trim() });
      setCreateForm({ username: '', password: '', nickname: '', roleCode: ROLE_CODE.OPERATOR });
      showToast({ tone: 'success', title: '账号已创建', description: `${user.username} · ${roleCodeMeta(primaryRoleCode(user.roleCodes)).label}` });
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
    if (!canUpdateTeamRole || !updateForm.userId.trim()) return;
    if (!MANAGED_TEAM_ROLES.includes(updateForm.roleCode)) {
      setError('团队成员只能设置为主播 / 场控或运营子账号');
      return;
    }
    setUpdating(true);
    setError('');
    try {
      const user = await adminUpdateUserRole(updateForm.userId.trim(), updateForm.roleCode);
      setUpdateForm({ userId: '', roleCode: ROLE_CODE.OPERATOR });
      showToast({ tone: 'success', title: '账号角色已更新', description: `${user.username} · ${roleCodeMeta(primaryRoleCode(user.roleCodes)).label}` });
      await syncUsers(query);
    } catch (e) {
      const message = resultMessage(e);
      setError(message);
      showToast({ tone: 'danger', title: '更新角色失败', description: message });
    } finally {
      setUpdating(false);
    }
  };

  const submitStatus = async (user: User) => {
    if (!canUpdateTeamStatus || !isManagedTeamUser(user)) return;
    const nextStatus = user.status === USER_STATUS.ACTIVE ? USER_STATUS.DISABLED : USER_STATUS.ACTIVE;
    setStatusUpdatingId(user.id);
    setError('');
    try {
      const updated = await adminUpdateUserStatus(user.id, nextStatus);
      showToast({ tone: 'success', title: nextStatus === USER_STATUS.ACTIVE ? '账号已启用' : '账号已停用', description: `${updated.username} · ${userStatusMeta(updated.status).label}` });
      await syncUsers(query);
    } catch (e) {
      const message = resultMessage(e);
      setError(message);
      showToast({ tone: 'danger', title: '账号状态更新失败', description: message });
    } finally {
      setStatusUpdatingId('');
    }
  };

  useEffect(() => { void syncUsers(); }, [canListTeam]);
  useEffect(() => { void syncUsers(query); }, [query.page, query.roleCode]);

  const metrics = useMemo(() => ({
    subaccounts: users.filter(isManagedTeamUser).length,
    active: users.filter((user) => user.status === USER_STATUS.ACTIVE).length,
    disabled: users.filter((user) => user.status === USER_STATUS.DISABLED).length,
  }), [users]);

  return <section className="teamAccountPage">
    <StudioToastViewport toasts={toasts} />
    <section className="laSettingsHero laMerchantsHero"><StudioPageHeader eyebrow="Team accounts" title="团队成员" description="每个主账号只管理自己主播 / 商家空间下的主播、场控和运营子账号。" actions={<StudioButton type="button" variant="secondary" icon={<RefreshCw size={15} />} loading={loading} disabled={!canListTeam} onClick={() => void syncUsers()}>{loading ? '同步中' : '同步账号'}</StudioButton>} /></section>
    {error ? <div className="auctionMgmtNotice danger"><AlertTriangle size={16} />{error}</div> : null}
    {!canListTeam ? <NoTeamPermissionNotice currentUser={currentUser} /> : <>
      <section className="laStatsGrid">
        <StudioMetricCard icon={<ShieldCheck />} label="账号总数" value={total} trend="当前商家空间" tone="info" />
        <StudioMetricCard icon={<UserCog />} label="团队子账号" value={metrics.subaccounts} trend="主播 / 场控 / 运营" tone="success" />
        <StudioMetricCard icon={<Power />} label="启用中" value={metrics.active} trend="允许登录后台" tone="success" />
        <StudioMetricCard icon={<Ban />} label="已停用" value={metrics.disabled} trend="禁止登录并撤销会话" tone="danger" />
      </section>
      <StudioCard padding="md">
        <div className="auctionFilterBar queueFilters" aria-label="账号筛选">
          <label><Search size={15} /><input value={query.keyword || ''} onChange={(e) => updateQuery({ keyword: e.target.value })} onKeyDown={(e) => { if (e.key === 'Enter') void syncUsers({ ...query, page: 1 }); }} placeholder="搜索用户 ID / 用户名 / 昵称" /></label>
          <StudioField label="角色"><select value={query.roleCode || ''} onChange={(e) => updateQuery({ roleCode: e.target.value as AdminUsersQuery['roleCode'] })}>{ROLE_CODE_FILTERS.map((item) => <option key={item.label} value={item.value}>{item.label}</option>)}</select></StudioField>
          <StudioButton type="button" variant="primary" icon={<Search size={15} />} onClick={() => void syncUsers({ ...query, page: 1 })}>查询</StudioButton>
        </div>
      </StudioCard>
      <section className="laGrid laGrid-1-1">
        <StudioCard title="创建团队子账号" subtitle="/api/admin/users" padding="md">
          <div className="laFormGrid">
            <StudioField label="用户名"><input value={createForm.username} onChange={(e) => setCreateForm({ ...createForm, username: e.target.value })} placeholder="team_operator_01" autoComplete="off" /></StudioField>
            <StudioField label="昵称"><input value={createForm.nickname} onChange={(e) => setCreateForm({ ...createForm, nickname: e.target.value })} placeholder="运营账号昵称" /></StudioField>
            <StudioField label="初始密码"><input value={createForm.password} onChange={(e) => setCreateForm({ ...createForm, password: e.target.value })} placeholder="输入后端允许的密码" type="password" autoComplete="new-password" /></StudioField>
            <StudioField label="角色"><select value={createForm.roleCode} onChange={(e) => setCreateForm({ ...createForm, roleCode: e.target.value as RoleCode })}>{TEAM_ACCOUNT_ROLE_OPTIONS.map((item) => <option key={item.roleCode} value={item.roleCode}>{item.label}</option>)}</select></StudioField>
          </div>
          <div className="laRulePreview">{TEAM_ACCOUNT_ROLE_OPTIONS.map((item) => <span key={item.roleCode}>{item.label} · {item.hint}</span>)}</div>
          <div className="drawerActions"><StudioButton type="button" variant="primary" loading={creating} disabled={!canCreateTeam || !createForm.username.trim() || !createForm.password || !createForm.nickname.trim()} onClick={() => void submitCreate()}>创建账号</StudioButton></div>
        </StudioCard>
        <StudioCard title="调整用户角色" subtitle="/api/admin/users/{userId}/role" padding="md">
          <div className="laFormGrid">
            <StudioField label="用户 ID"><input value={updateForm.userId} onChange={(e) => setUpdateForm({ ...updateForm, userId: e.target.value })} placeholder="输入后端 user.id" /></StudioField>
            <StudioField label="新角色"><select value={updateForm.roleCode} onChange={(e) => setUpdateForm({ ...updateForm, roleCode: e.target.value as RoleCode })}>{TEAM_ACCOUNT_ROLE_OPTIONS.map((item) => <option key={item.roleCode} value={item.roleCode}>{item.label}</option>)}</select></StudioField>
          </div>
          <StudioEmptyState compact icon={<CheckCircle2 size={22} />} title="角色变更后立即刷新列表" description="后端按主账号空间做最终校验；页面不提供主账号或买家账号创建入口。" />
          <div className="drawerActions"><StudioButton type="button" variant="primary" loading={updating} disabled={!canUpdateTeamRole || !updateForm.userId.trim()} onClick={() => void submitUpdate()}>更新角色</StudioButton></div>
        </StudioCard>
      </section>
      {loading ? <StudioTableSkeleton rows={6} columns={7} /> : <StudioTable
        rows={users}
        rowKey={(user) => user.id}
        header={`共 ${total} 个账号 · 每页 ${DEFAULT_PAGE_SIZE} 条`}
        filters={<div className="orderPager">
            <button type="button" disabled={currentPage <= 1 || loading} onClick={goPrevPage}><ChevronLeft size={15} /><span>上一页</span></button>
            <span className="orderPagerIndex">第 {currentPage} / {totalPages} 页</span>
            <button type="button" disabled={currentPage >= totalPages || loading} onClick={goNextPage}><span>下一页</span><ChevronRight size={15} /></button>
          </div>}
        empty={<StudioEmptyState compact icon={<Users size={28} />} title="暂无账号" description="当前筛选条件下没有用户。" />}
        columns={[
          { label: '用户', render: (user) => <div><b>{user.username}</b><br /><span>{user.nickname || '未设置昵称'}</span></div> },
          { label: '用户 ID', render: (user) => <code>{user.id}</code> },
          { label: '角色', render: (user) => <StudioBadge tone={roleCodeMeta(primaryRoleCode(user.roleCodes)).tone}>{roleCodeMeta(primaryRoleCode(user.roleCodes)).label}</StudioBadge> },
          { label: '账号状态', render: (user) => <StudioBadge tone={userStatusMeta(user.status).tone}>{userStatusMeta(user.status).label}</StudioBadge> },
          { label: '创建时间', render: (user) => formatDateTimeText(user.createdAtUnixMs) },
          { label: '更新时间', render: (user) => formatDateTimeText(user.updatedAtUnixMs) },
          { label: '操作', render: (user) => <div className="laRowActions">
            <button type="button" disabled={!canUpdateTeamRole} onClick={() => setUpdateForm({ userId: user.id, roleCode: primaryRoleCode(user.roleCodes) as RoleCode })}>填入改角色</button>
            <button type="button" disabled={!canUpdateTeamStatus || statusUpdatingId === user.id} onClick={() => void submitStatus(user)}>{user.status === USER_STATUS.ACTIVE ? '停用账号' : '启用账号'}</button>
          </div> },
        ]}
      />}
    </>}
  </section>;
}

function NoTeamPermissionNotice({ currentUser }: { currentUser?: User | null }) {
  return <>
    <section className="laStatsGrid">
      <StudioMetricCard icon={<ShieldCheck />} label="当前后台账号" value={currentUser?.username || '未同步'} trend={currentUser ? roleCodeMeta(primaryRoleCode(currentUser.roleCodes)).label : 'AuthSession'} tone="info" />
      <StudioMetricCard icon={<ShieldAlert />} label="账号管理权限" value="未授权" trend="缺少 team.user.list 权限" tone="danger" />
    </section>
    <StudioCard title="仅主账号可管理团队成员" subtitle="Main account only" padding="md">
      <StudioEmptyState compact icon={<ShieldAlert size={28} />} title="仅主账号可管理团队成员" description="当前账号只能查看后台业务功能，不能看到或提交创建团队子账号、修改角色、停用账号表单。" />
    </StudioCard>
  </>;
}

function isManagedTeamUser(user: User) {
  return user.roleCodes.some((roleCode) => isManagedTeamRole(roleCode));
}
