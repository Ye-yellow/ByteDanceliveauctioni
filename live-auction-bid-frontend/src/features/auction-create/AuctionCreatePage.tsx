import { useMemo, useState } from 'react';
import { AlertTriangle, CheckCircle2, Gavel, Package, Radio, Sparkles } from 'lucide-react';
import { createDraftLot, patchDraftLot, queueLot } from '../auction/api/auctionApi';
import type { CreateLotRequest, Money } from '../../shared/api/types';
import { resultMessage } from '../../shared/api/result';
import { formatMoneyText } from '../../shared/lib/format';
import { ADMIN_ROOM } from '../../shared/config/studio';
import { StudioBadge, StudioButton, StudioCard, StudioEmptyState, StudioField, StudioMetricCard, StudioPageHeader, StudioToastViewport, useStudioToast } from '../../pages/host-console/components/studio-ui';

type FormState = {
  title: string;
  description: string;
  imageUrl: string;
  startPrice: number;
  minIncrement: number;
  capPrice: number | '';
  durationSeconds: number;
  antiSnipeWindowSeconds: number;
  antiSnipeExtendSeconds: number;
  maxExtendCount: number;
  certificate: string;
  flaw: string;
  service: string;
};

const initialForm: FormState = {
  title: '',
  description: '',
  imageUrl: '',
  startPrice: 0,
  minIncrement: 50,
  capPrice: '',
  durationSeconds: 300,
  antiSnipeWindowSeconds: 10,
  antiSnipeExtendSeconds: 15,
  maxExtendCount: 5,
  certificate: '',
  flaw: '',
  service: '',
};

export function AuctionCreatePage() {
  const [form, setForm] = useState<FormState>(initialForm);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState('');
  const { toasts, showToast } = useStudioToast();

  const issues = useMemo(() => validate(form), [form]);
  const hasError = issues.some((issue) => issue.level === 'error');

  const update = (patch: Partial<FormState>) => setForm((current) => ({ ...current, ...patch }));

  const submit = async () => {
    if (hasError) return;
    setSubmitting(true);
    setError('');
    try {
      const draft = await createDraftLot({ roomId: ADMIN_ROOM.id });
      const saved = await patchDraftLot(draft.id, toRequest(form));
      const queued = await queueLot(saved.id);
      showToast({ tone: 'success', title: '拍品已加入本场队列', description: `${queued.lot.title} · #${queued.queuePosition || queued.lot.queuePosition || '-'}` });
      window.setTimeout(() => { location.href = '/admin/auctions?queued=1'; }, 350);
    } catch (e) {
      const message = resultMessage(e);
      setError(message);
      showToast({ tone: 'danger', title: '加入队列失败', description: message });
      setSubmitting(false);
    }
  };

  return <section className="auctionCreatePage">
    <StudioToastViewport toasts={toasts} className="auctionCreateToastViewport" />
    <StudioCard padding="lg" className="auctionCreateTitleBar">
      <StudioPageHeader eyebrow="Create lot" title="添加拍品" description="P2 保留真实创建链路：创建草稿、保存规则、加入当前直播间队列。" actions={<a className="studioButton studioButton-secondary studioButton-md" href="/admin/auctions">返回队列</a>} />
    </StudioCard>
    {error ? <div className="auctionMgmtNotice danger"><AlertTriangle size={16} />{error}</div> : null}
    <section className="auctionMgmtStats">
      <StudioMetricCard icon={<Radio />} label="固定直播间" value={ADMIN_ROOM.id} trend={ADMIN_ROOM.name} tone="info" />
      <StudioMetricCard icon={<Gavel />} label="起拍价" value={formatMoneyText(auctionMoney(form.startPrice))} trend={`加价 ${formatMoneyText(auctionMoney(form.minIncrement))}`} tone="success" />
      <StudioMetricCard icon={<Sparkles />} label="讲解卡" value={trustCards(form).length} trend="证书 / 瑕疵 / 售后" tone="purple" />
      <StudioMetricCard icon={<Package />} label="校验" value={issues.length ? issues.length : '通过'} trend={hasError ? '存在阻断项' : '可加入队列'} tone={hasError ? 'danger' : 'success'} />
    </section>
    <div className="auctionCreateLayout">
      <main className="auctionCreateMain">
        <StudioCard title="拍品资料" subtitle="Product" padding="md">
          <div className="productBaseGrid">
            <StudioField label="拍品名称" error={issueText(issues, 'lot title')}><input value={form.title} onChange={(e) => update({ title: e.target.value })} placeholder="请输入竞拍拍品名称" /></StudioField>
            <StudioField label="主图 URL" help="必须使用后端上传接口返回的稳定 URL。" error={issueText(issues, 'image url')}><input value={form.imageUrl} onChange={(e) => update({ imageUrl: e.target.value })} placeholder="https://..." /></StudioField>
            <StudioField label="拍品介绍" error={issueText(issues, '介绍')}><textarea value={form.description} onChange={(e) => update({ description: e.target.value })} rows={5} placeholder="描述材质、成色、亮点和竞拍价值" /></StudioField>
          </div>
        </StudioCard>
        <StudioCard title="竞拍规则" subtitle="Rules" padding="md">
          <div className="ruleFieldGrid">
            <StudioField label="起拍价"><input type="number" value={form.startPrice} min={0} onChange={(e) => update({ startPrice: Number(e.target.value) })} /></StudioField>
            <StudioField label="加价幅度" error={issueText(issues, 'min increment')}><input type="number" value={form.minIncrement} min={1} onChange={(e) => update({ minIncrement: Number(e.target.value) })} /></StudioField>
            <StudioField label="封顶价" error={issueText(issues, 'cap price')}><input type="number" value={form.capPrice} placeholder="可选" onChange={(e) => update({ capPrice: e.target.value === '' ? '' : Number(e.target.value) })} /></StudioField>
            <StudioField label="竞拍时长（秒）" error={issueText(issues, 'duration seconds')}><input type="number" value={form.durationSeconds} min={60} onChange={(e) => update({ durationSeconds: Number(e.target.value) })} /></StudioField>
            <StudioField label="延时窗口（秒）" error={issueText(issues, 'anti-snipe window')}><input type="number" value={form.antiSnipeWindowSeconds} min={1} onChange={(e) => update({ antiSnipeWindowSeconds: Number(e.target.value) })} /></StudioField>
            <StudioField label="每次延长（秒）" error={issueText(issues, 'anti-snipe extend')}><input type="number" value={form.antiSnipeExtendSeconds} min={10} max={30} onChange={(e) => update({ antiSnipeExtendSeconds: Number(e.target.value) })} /></StudioField>
            <StudioField label="最大延时次数" error={issueText(issues, 'max extend')}><input type="number" value={form.maxExtendCount} min={1} onChange={(e) => update({ maxExtendCount: Number(e.target.value) })} /></StudioField>
          </div>
        </StudioCard>
        <StudioCard title="主播讲解卡" subtitle="Trust cards" padding="md">
          <div className="productDetailGrid">
            <StudioField label="证书信息"><textarea value={form.certificate} onChange={(e) => update({ certificate: e.target.value })} rows={3} placeholder="证书编号、鉴定机构、材质证明等" /></StudioField>
            <StudioField label="瑕疵说明"><textarea value={form.flaw} onChange={(e) => update({ flaw: e.target.value })} rows={3} placeholder="如实记录磨损、划痕、缺件或使用痕迹" /></StudioField>
            <StudioField label="售后说明"><textarea value={form.service} onChange={(e) => update({ service: e.target.value })} rows={3} placeholder="退换、支付、发货、客服承诺等" /></StudioField>
          </div>
        </StudioCard>
        <div className="auctionStepNav"><StudioButton type="button" variant="secondary" onClick={() => setForm(initialForm)} disabled={submitting}>清空</StudioButton><StudioButton type="button" variant="primary" loading={submitting} disabled={hasError} onClick={() => void submit()}>{submitting ? '正在加入本场队列...' : '加入本场队列'}</StudioButton></div>
      </main>
      <aside className="stickyPreviewPanel">
        <StudioCard title="发布检查" subtitle="Review" padding="md">
          {issues.length ? <div className="publishIssueBox">{issues.map((issue) => <div key={issue.text} className={issue.level}><AlertTriangle size={15} /><span>{issue.text}</span></div>)}</div> : <StudioEmptyState compact tone="success" icon={<CheckCircle2 size={22} />} title="核心配置已通过" description="提交后会创建草稿、保存规则并加入队列。" />}
        </StudioCard>
        <StudioCard title="规则摘要" subtitle="Summary" padding="md"><div className="ruleSnapshotGrid"><div><span>起拍价</span><b>{formatMoneyText(auctionMoney(form.startPrice))}</b></div><div><span>加价幅度</span><b>{formatMoneyText(auctionMoney(form.minIncrement))}</b></div><div><span>封顶价</span><b>{form.capPrice === '' ? '未设置' : formatMoneyText(auctionMoney(form.capPrice))}</b></div><div><span>竞拍时长</span><b>{form.durationSeconds}s</b></div><div><span>最后出价延时</span><b>{form.antiSnipeWindowSeconds}s / +{form.antiSnipeExtendSeconds}s</b></div></div></StudioCard>
        <StudioCard title="状态" subtitle="Contract" padding="md"><div className="laRulePreview"><StudioBadge tone="info">LOT_STATUS_DRAFT</StudioBadge><StudioBadge tone="warning">QUEUE</StudioBadge><StudioBadge tone="success">HTTP only</StudioBadge></div></StudioCard>
      </aside>
    </div>
  </section>;
}

function auctionMoney(amount: number | ''): Money {
  return { amount: Number(amount || 0), currency: 'CNY' };
}

function trustCards(form: FormState): CreateLotRequest['trustCards'] {
  return [
    { id: 'certificate-card', type: 'TRUST_CARD_TYPE_CERTIFICATE', title: '证书卡', content: form.certificate.trim() || '证书信息待补充' },
    { id: 'flaw-card', type: 'TRUST_CARD_TYPE_FLAW', title: '瑕疵说明卡', content: form.flaw.trim() || '瑕疵说明待补充' },
    { id: 'service-card', type: 'TRUST_CARD_TYPE_SERVICE', title: '售后说明卡', content: form.service.trim() || '售后说明待补充' },
  ];
}

function toRequest(form: FormState): CreateLotRequest {
  return {
    roomId: ADMIN_ROOM.id,
    title: form.title.trim(),
    description: form.description.trim(),
    imageUrl: form.imageUrl.trim(),
    rule: {
      startPrice: auctionMoney(form.startPrice),
      minIncrement: auctionMoney(form.minIncrement),
      ...(form.capPrice !== '' ? { capPrice: auctionMoney(form.capPrice) } : {}),
      durationSeconds: form.durationSeconds,
      antiSnipeWindowSeconds: form.antiSnipeWindowSeconds,
      antiSnipeExtendSeconds: form.antiSnipeExtendSeconds,
      maxExtendCount: form.maxExtendCount,
    },
    trustCards: trustCards(form),
  };
}

function validate(form: FormState) {
  const issues: Array<{ level: 'error' | 'warning'; text: string }> = [];
  if (!form.title.trim()) issues.push({ level: 'error', text: 'lot title is required' });
  if (!form.imageUrl.trim()) issues.push({ level: 'error', text: 'lot image url is required' });
  if (!form.description.trim()) issues.push({ level: 'warning', text: '拍品介绍较弱，建议补充材质、成色和竞拍亮点' });
  if (form.minIncrement <= 0) issues.push({ level: 'error', text: 'min increment amount must be > 0' });
  if (form.durationSeconds < 60) issues.push({ level: 'error', text: 'duration seconds must be >= 60' });
  if (form.antiSnipeWindowSeconds <= 0) issues.push({ level: 'error', text: 'anti-snipe window seconds must be > 0' });
  if (form.antiSnipeExtendSeconds < 10 || form.antiSnipeExtendSeconds > 30) issues.push({ level: 'error', text: 'anti-snipe extend seconds must be between 10 and 30' });
  if (form.maxExtendCount <= 0) issues.push({ level: 'error', text: 'max extend count must be > 0' });
  if (form.capPrice !== '' && form.capPrice <= form.startPrice) issues.push({ level: 'error', text: 'cap price amount must be greater than start price amount' });
  return issues;
}

function issueText(issues: Array<{ text: string }>, keyword: string) {
  return issues.find((issue) => issue.text.includes(keyword))?.text;
}
