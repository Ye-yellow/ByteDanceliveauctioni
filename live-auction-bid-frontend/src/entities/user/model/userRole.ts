import type { UserRole } from '../../../shared/api/types';
import { USER_ROLE } from '../../../shared/api/types';
import type { StudioTone } from '../../../pages/host-console/components/studio-ui';

export const USER_ROLE_FILTERS: Array<{ label: string; value: UserRole | '' }> = [
  { label: '全部后台账号', value: '' },
  { label: '主播主账号', value: USER_ROLE.ADMIN },
  { label: '主播 / 场控', value: USER_ROLE.ANCHOR },
  { label: '运营子账号', value: USER_ROLE.OPERATOR },
];

export const USER_ROLE_OPTIONS: Array<{ role: UserRole; label: string; hint: string; tone: StudioTone }> = [
  { role: USER_ROLE.ADMIN, label: '主播主账号', hint: '由平台发放，可管理当前商家空间和团队子账号', tone: 'purple' },
  { role: USER_ROLE.ANCHOR, label: '主播 / 场控', hint: '可进入工作台并执行开拍、控场、落锤等操作', tone: 'success' },
  { role: USER_ROLE.OPERATOR, label: '运营子账号', hint: '可进入工作台并协助拍品、队列和成交处理', tone: 'info' },
  { role: USER_ROLE.BUYER, label: '买家（H5）', hint: '只属于用户 H5，不允许进入或由 Admin 工作台创建', tone: 'warning' },
];

export const TEAM_ACCOUNT_ROLE_OPTIONS = [
  { role: USER_ROLE.ANCHOR, label: '主播 / 场控', hint: '协助直播中开拍、控场、落锤和异常处理', tone: 'success' as StudioTone },
  { role: USER_ROLE.OPERATOR, label: '运营子账号', hint: '协助拍品准备、队列维护和成交处理', tone: 'info' as StudioTone },
];

export function userRoleMeta(role?: UserRole | string | null) {
  return USER_ROLE_OPTIONS.find((item) => item.role === role) ?? {
    role: String(role || USER_ROLE.UNSPECIFIED),
    label: String(role || '未知角色'),
    hint: '后端返回的未识别角色',
    tone: 'neutral' as StudioTone,
  };
}
