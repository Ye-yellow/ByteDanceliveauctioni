export function createIdempotencyKey(prefix = 'bid', ...parts: Array<string | number | undefined>) {
  const stableParts = parts.filter((part) => part !== undefined && part !== '').join('-');
  const random = crypto.randomUUID?.() || Math.random().toString(36).slice(2);
  return [prefix, stableParts, Date.now(), random].filter(Boolean).join('-');
}
