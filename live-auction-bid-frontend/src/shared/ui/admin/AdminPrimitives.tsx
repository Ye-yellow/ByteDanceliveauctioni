import type { ReactNode } from 'react';

type NavItem = {
  id: string;
  label: string;
  description?: string;
  icon?: ReactNode;
  active?: boolean;
  disabled?: boolean;
  onClick?: () => void;
};

export function AdminLayout({
  title,
  subtitle,
  navItems,
  userSlot,
  actions,
  children,
}: {
  title: string;
  subtitle?: string;
  navItems: NavItem[];
  userSlot?: ReactNode;
  actions?: ReactNode;
  children: ReactNode;
}) {
  return (
    <main className="adminShell">
      <aside className="adminSidebar" aria-label="后台导航">
        <a className="adminBrand" href="/host" aria-label="直播竞拍后台首页">
          <span className="adminBrandMark">拍</span>
          <span className="adminBrandText">
            <strong>Live Auction</strong>
            <small>Management Center</small>
          </span>
        </a>
        <nav className="adminNav">
          {navItems.map((item) => (
            <button
              key={item.id}
              className={`adminNavItem ${item.active ? 'active' : ''}`}
              disabled={item.disabled}
              onClick={item.onClick}
              title={item.disabled ? `${item.label} · 待后端契约扩展` : item.label}
            >
              <span className="adminNavIcon">{item.icon ?? item.label.slice(0, 1)}</span>
              <span>
                <strong>{item.label}</strong>
                {item.description && <small>{item.description}</small>}
              </span>
            </button>
          ))}
        </nav>
        <div className="adminSidebarFooter">
          <a href="/home" className="adminGhostLink">返回 Home</a>
          <span>后台 Web 专用 · 真实后端契约 · 无 mock</span>
        </div>
      </aside>

      <section className="adminMain">
        <header className="adminHeader">
          <div>
            <p className="adminBreadcrumb">Admin / Auction / {title}</p>
            <h1>{title}</h1>
            {subtitle && <p>{subtitle}</p>}
          </div>
          <div className="adminHeaderActions">
            {actions}
            {userSlot}
          </div>
        </header>
        {children}
      </section>
    </main>
  );
}

export function StatCard({
  label,
  value,
  hint,
  tone = 'neutral',
  icon,
}: {
  label: string;
  value: ReactNode;
  hint?: string;
  tone?: 'neutral' | 'success' | 'warning' | 'danger' | 'primary';
  icon?: ReactNode;
}) {
  return (
    <article className={`adminStat stat-${tone}`}>
      <div className="adminStatIcon">{icon ?? '•'}</div>
      <div>
        <span>{label}</span>
        <strong>{value}</strong>
        {hint && <small>{hint}</small>}
      </div>
    </article>
  );
}

export function StatusBadge({ label, tone = 'neutral' }: { label: string; tone?: 'neutral' | 'success' | 'warning' | 'danger' | 'info' | 'purple' }) {
  return <span className={`adminStatusBadge badge-${tone}`}><i />{label}</span>;
}

export function EmptyState({ title, description, action }: { title: string; description?: string; action?: ReactNode }) {
  return (
    <div className="adminEmptyState">
      <div className="adminEmptyIcon">⌁</div>
      <h3>{title}</h3>
      {description && <p>{description}</p>}
      {action && <div>{action}</div>}
    </div>
  );
}

type Column<T> = {
  key: string;
  label: string;
  className?: string;
  render: (row: T, index: number) => ReactNode;
};

export function DataTable<T>({
  rows,
  columns,
  rowKey,
  empty,
}: {
  rows: T[];
  columns: Column<T>[];
  rowKey: (row: T, index: number) => string;
  empty?: ReactNode;
}) {
  if (!rows.length) return <>{empty ?? <EmptyState title="暂无数据" />}</>;
  return (
    <div className="adminTableWrap">
      <table className="adminTable">
        <thead>
          <tr>{columns.map((column) => <th className={column.className} key={column.key}>{column.label}</th>)}</tr>
        </thead>
        <tbody>
          {rows.map((row, index) => (
            <tr key={rowKey(row, index)}>
              {columns.map((column) => <td className={column.className} key={column.key}>{column.render(row, index)}</td>)}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
