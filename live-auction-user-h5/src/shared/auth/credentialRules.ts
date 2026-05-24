export const BUYER_USERNAME_MIN_LENGTH = 6;
export const BUYER_USERNAME_MAX_LENGTH = 64;
export const BUYER_PASSWORD_MIN_LENGTH = 8;
export const BUYER_PASSWORD_MAX_LENGTH = 128;
export const BUYER_NICKNAME_MAX_LENGTH = 128;
export const BUYER_USERNAME_PATTERN = '[A-Za-z0-9_-]{6,64}';

const USERNAME_RE = /^[a-zA-Z0-9_-]{6,64}$/;

export function normalizeBuyerUsername(username: string): string {
  return username.trim().toLowerCase();
}

export function validateBuyerCredentials({
  username,
  password,
  nickname,
  requireNickname,
}: {
  username: string;
  password: string;
  nickname?: string;
  requireNickname?: boolean;
}): string {
  const normalizedUsername = username.trim();
  const trimmedNickname = nickname?.trim() ?? '';

  if (!USERNAME_RE.test(normalizedUsername)) {
    return '账号需为 6-64 位，只能包含字母、数字、下划线或中横线';
  }
  if (password.length < BUYER_PASSWORD_MIN_LENGTH || password.length > BUYER_PASSWORD_MAX_LENGTH) {
    return '密码需为 8-128 位';
  }
  if (requireNickname && !trimmedNickname) {
    return '请输入昵称';
  }
  if (trimmedNickname.length > BUYER_NICKNAME_MAX_LENGTH) {
    return '昵称最多 128 个字符';
  }
  return '';
}
