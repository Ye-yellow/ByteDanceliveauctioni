import type { UserRole } from '../../../shared/api/types';
import { USER_ROLE } from '../../../shared/api/types';
import type { StudioTone } from '../../../pages/host-console/components/studio-ui';

export const USER_ROLE_FILTERS: Array<{ label: string; value: UserRole | '' }> = [
  { label: '全部角色', value: '' },
  { label: '管理员', value: USER_ROLE.ADMIN },
  { label: '主播', value: USER_ROLE.ANCHOR },
  { label: '运营', value: USER_ROLE.OPERATOR },
  { label: '买家', value: USER_ROLE.BUYER },
];

export const USER_ROLE_OPTIONS: Array<{ role: UserRole; label: string; hint: string; tone: StudioTone }> = [
  { role: USER_ROLE.ADMIN, label: '管理员', hint: '可管理账号和后台配置', tone: 'purple' },
  { role: USER_ROLE.ANCHOR, label: '主播', hint: '可进入后台并管理竞拍', tone: 'success' },
  { role: USER_ROLE.OPERATOR, label: '运营', hint: '可进入后台并协助控场', tone: 'info' },
  { role: USER_ROLE.BUYER, label: '买家', hint: '只能进入 H5，不能进入后台', tone: 'warning' },
];

export function userRoleMeta(role?: UserRole | string | null) {
  return USER_ROLE_OPTIONS.find((item) => item.role === role) ?? {
    role: String(role || USER_ROLE.UNSPECIFIED),
    label: String(role || '未知角色'),
    hint: '后端返回的未识别角色',
    tone: 'neutral' as StudioTone,
  };
}
