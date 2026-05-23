import { useEffect, useState } from 'react';
import { AlertTriangle, Package, RefreshCw, Search, ShoppingBag } from 'lucide-react';
import { listAdminLots, type AdminLotsQuery } from '../auction/api/auctionApi';
import { LIBRARY_LOT_STATUS_FILTERS, lotStatusLabel, lotStatusTone } from '../../entities/auction/model/auctionStatus';
import type { Lot } from '../../shared/api/types';
import { resultMessage } from '../../shared/api/result';
import { formatMoneyText } from '../../shared/lib/format';
import { ADMIN_ROOM } from '../../shared/config/studio';
import { StudioBadge, StudioButton, StudioCard, StudioEmptyState, StudioField, StudioPageHeader, StudioTableSkeleton } from '../../pages/host-console/components/studio-ui';

export function ProductLibraryPage({ roomId = ADMIN_ROOM.id }: { roomId?: string }) {
  const [query, setQuery] = useState<AdminLotsQuery>({ page: 1, pageSize: 20, roomId, view: 'library' });
  const [lots, setLots] = useState<Lot[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  const syncLots = async (nextQuery = query) => {
    setLoading(true);
    setError('');
    try {
      const page = await listAdminLots({ ...nextQuery, roomId, view: 'library' });
      setLots(page.lots);
      setTotal(page.total);
      setQuery((current) => ({ ...current, view: 'library', page: page.page, pageSize: page.pageSize }));
    } catch (e) {
      setError(resultMessage(e));
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { void syncLots(); }, []);
  useEffect(() => { void syncLots(query); }, [query.page, query.status]);

  return <section className="productLibraryPage">
    <StudioCard padding="lg" className="productLibraryHero"><StudioPageHeader eyebrow="Product library" title="拍品库" description="只保留草稿和准备中的可复用拍品资料；取消、异常、成交结果统一进入拍品历史。" actions={<><a className="studioButton studioButton-secondary studioButton-md" href="/admin/auctions/history">拍品历史</a><StudioButton type="button" variant="secondary" icon={<RefreshCw size={15} />} loading={loading} onClick={() => void syncLots()}>同步拍品</StudioButton><a className="studioButton studioButton-primary studioButton-md" href="/admin/auctions/create">添加拍品</a></>} /></StudioCard>
    {error ? <div className="auctionMgmtNotice danger"><AlertTriangle size={16} />{error}</div> : null}
    <section className="productLibraryStats"><div><span>可复用资产</span><b>{total}</b><small>草稿 / 准备中</small></div><div><span>本页拍品</span><b>{lots.length}</b><small>分页结果</small></div><div><span>讲解卡</span><b>{lots.reduce((sum, item) => sum + (item.trustCards?.length || 0), 0)}</b><small>来自真实 lot</small></div></section>
    <StudioCard padding="md"><div className="productLibraryFilters"><label><Search size={17} /><input value={query.keyword || ''} onChange={(e) => setQuery((current) => ({ ...current, view: 'library', keyword: e.target.value, page: 1 }))} onKeyDown={(e) => { if (e.key === 'Enter') void syncLots({ ...query, view: 'library', page: 1 }); }} placeholder="搜索拍品名 / 竞拍 ID" /></label><StudioField label="资产状态"><select value={query.status || ''} onChange={(e) => setQuery((current) => ({ ...current, view: 'library', status: e.target.value as AdminLotsQuery['status'], page: 1 }))}>{LIBRARY_LOT_STATUS_FILTERS.map((item) => <option key={item.label} value={item.value}>{item.label}</option>)}</select></StudioField><StudioButton type="button" variant="primary" icon={<Search size={15} />} onClick={() => void syncLots({ ...query, view: 'library', page: 1 })}>查询</StudioButton></div></StudioCard>
    <section className="productLibraryList" aria-label="拍品库列表">{loading ? <StudioTableSkeleton rows={5} columns={4} /> : lots.length ? lots.map((lot, index) => <article className="productLibraryRow" key={lot.id}><div className="productIdentity"><span className="productLibraryNo">#{String(index + 1).padStart(2, '0')}</span><div className="productLibraryThumb">{lot.imageUrl ? <img src={lot.imageUrl} alt={lot.title} /> : <ShoppingBag size={28} />}</div><div><h3>{lot.title}</h3><div className="productLibraryTags"><StudioBadge tone={lotStatusTone(lot.status)}>{lotStatusLabel(lot.status)}</StudioBadge><span>{lot.id}</span><span>{roomId}</span></div></div></div><div className="productLibraryMetrics"><span><b>起拍价</b>{formatMoneyText(lot.rule.startPrice)}</span><span><b>加价</b>{formatMoneyText(lot.rule.minIncrement)}</span><span><b>封顶</b>{formatMoneyText(lot.rule.capPrice)}</span><span><b>讲解卡</b>{lot.trustCards?.length || 0}</span></div><div className="productLibraryActions"><a className="primary" href={`/admin/auctions/create?lotId=${encodeURIComponent(lot.id)}`}>编辑资料</a><a href="/admin/auctions">查看本场队列</a></div></article>) : <StudioEmptyState icon={<Package size={34} />} title="暂无可复用拍品" description="取消、异常和成交记录不进入拍品库，可到拍品历史查看。" action={<a className="studioButton studioButton-primary studioButton-md" href="/admin/auctions/create">添加拍品</a>} />}</section>
  </section>;
}
