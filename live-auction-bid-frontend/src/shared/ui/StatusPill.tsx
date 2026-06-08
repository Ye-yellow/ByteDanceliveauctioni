type Props = { ok: boolean; okText: string; pendingText: string };

export function StatusPill({ ok, okText, pendingText }: Props) {
  return <div className={ok ? 'status ok' : 'status'}>{ok ? okText : pendingText}</div>;
}
