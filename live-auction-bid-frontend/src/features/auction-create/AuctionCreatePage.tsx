import { useEffect, useMemo, useState, type ChangeEvent, type ReactNode } from 'react';
import { AlertTriangle, ArrowDown, ArrowUp, CheckCircle2, ChevronLeft, ChevronRight, ImagePlus, Trash2, UploadCloud } from 'lucide-react';
import { createDraftLot, deleteUploadedImage, patchDraftLot, queueLot, uploadImage } from '../auction/api/auctionApi';
import type { CreateLotRequest, Money, TrustCardType, UploadedAsset } from '../../shared/api/types';
import { resultMessage } from '../../shared/api/result';
import { formatMoneyText } from '../../shared/lib/format';
import { StudioButton, StudioCard, StudioField, StudioPageHeader, StudioToastViewport, useStudioToast } from '../../pages/host-console/components/studio-ui';

type UploadedImage = {
  assetId?: string;
  imageUrl: string;
  fileName?: string;
  sizeBytes?: number | string;
};

type TrustCardKey = 'certificate' | 'flaw' | 'detail' | 'service';

type TrustCardDraft = {
  content: string;
  imageUrl: string;
  assetId?: string;
};

type FormState = {
  title: string;
  description: string;
  imageUrl: string;
  mainImageAssetId?: string;
  gallery: UploadedImage[];
  categoryMode: 'preset' | 'custom';
  category: string;
  tags: string;
  estimatePrice: number | '';
  stock: number;
  afterSaleNotes: string;
  startPrice: number;
  minIncrement: number;
  capPrice: number | '';
  depositAmount: number;
  durationSeconds: number;
  antiSnipeWindowSeconds: number;
  antiSnipeExtendSeconds: number;
  maxExtendCount: number;
  trustCards: Record<TrustCardKey, TrustCardDraft>;
};

type StepKey = 'product' | 'rules' | 'briefing' | 'review';

type FormIssue = { level: 'error' | 'warning' | 'success'; step: StepKey; text: string };

const STEP_DEFS: Array<{ key: StepKey; label: string; hint: string }> = [
  { key: 'product', label: '拍品资料', hint: '图片 / 基础信息' },
  { key: 'rules', label: '竞拍规则', hint: '价格 / 延时机制' },
  { key: 'briefing', label: '主播讲解', hint: '证书 / 瑕疵 / 售后' },
  { key: 'review', label: '确认发布', hint: '检查后入队' },
];

const TRUST_CARD_DEFS: Array<{ key: TrustCardKey; type: TrustCardType; title: string; label: string; placeholder: string }> = [
  { key: 'certificate', type: 'TRUST_CARD_TYPE_CERTIFICATE', title: '证书卡', label: '证书信息', placeholder: '证书编号、鉴定机构、材质证明等' },
  { key: 'flaw', type: 'TRUST_CARD_TYPE_FLAW', title: '瑕疵说明卡', label: '瑕疵说明', placeholder: '如实记录磨损、划痕、缺件或使用痕迹' },
  { key: 'detail', type: 'TRUST_CARD_TYPE_DETAIL', title: '细节展示卡', label: '细节说明', placeholder: '工艺、材质、尺寸、佩戴/使用细节' },
  { key: 'service', type: 'TRUST_CARD_TYPE_SERVICE', title: '售后说明卡', label: '售后说明', placeholder: '退换、支付、发货、客服承诺等' },
];

const CUSTOM_CATEGORY_VALUE = '__custom__';

const CATEGORY_OPTIONS = [
  '翡翠玉石',
  '珠宝彩宝',
  '黄金贵金属',
  '腕表配饰',
  '文玩收藏',
  '字画艺术',
  '陶瓷紫砂',
  '潮玩手办',
  '奢侈品',
  '酒水茶叶',
  '数码家电',
  '服饰箱包',
];

const initialForm: FormState = {
  title: '',
  description: '',
  imageUrl: '',
  gallery: [],
  categoryMode: 'preset',
  category: '',
  tags: '',
  estimatePrice: '',
  stock: 1,
  afterSaleNotes: '',
  startPrice: 0,
  minIncrement: 50,
  capPrice: '',
  depositAmount: 0,
  durationSeconds: 300,
  antiSnipeWindowSeconds: 10,
  antiSnipeExtendSeconds: 15,
  maxExtendCount: 5,
  trustCards: {
    certificate: { content: '', imageUrl: '' },
    flaw: { content: '', imageUrl: '' },
    detail: { content: '', imageUrl: '' },
    service: { content: '', imageUrl: '' },
  },
};

type AuctionCreatePageProps = {
  roomId: string;
  roomName?: string;
};

export function AuctionCreatePage({ roomId, roomName = roomId }: AuctionCreatePageProps) {
  const [form, setForm] = useState<FormState>(initialForm);
  const [activeStep, setActiveStep] = useState<StepKey>('product');
  const [previewImageIndex, setPreviewImageIndex] = useState(0);
  const [submitting, setSubmitting] = useState(false);
  const [uploading, setUploading] = useState<Record<string, boolean>>({});
  const [error, setError] = useState('');
  const { toasts, showToast } = useStudioToast();

  const issues = useMemo(() => validate(form), [form]);
  const blockingIssues = issues.filter(isBlockingIssue);
  const hasError = blockingIssues.length > 0;
  const isUploading = Object.values(uploading).some(Boolean);
  const tagList = useMemo(() => parseTags(form.tags), [form.tags]);
  const trustCards = useMemo(() => buildTrustCards(form), [form]);
  const previewImages = useMemo(() => [
    ...(form.imageUrl ? [{ imageUrl: form.imageUrl, label: '主图' }] : []),
    ...form.gallery.map((image, index) => ({ imageUrl: image.imageUrl, label: `轮播 ${index + 1}` })),
  ], [form.gallery, form.imageUrl]);
  const previewImage = previewImages[previewImageIndex] || previewImages[0];
  const activeStepIndex = STEP_DEFS.findIndex((step) => step.key === activeStep);
  const currentStepBlocking = activeStep === 'review' ? blockingIssues : blockingIssues.filter((issue) => issue.step === activeStep);
  const currentStepHasError = currentStepBlocking.length > 0;
  const canGoBack = activeStepIndex > 0;

  const canEnterStep = (stepKey: StepKey) => {
    const targetIndex = STEP_DEFS.findIndex((step) => step.key === stepKey);
    // Backward: always allowed — return to any previous step freely.
    if (targetIndex <= activeStepIndex) return true;
    // Forward: only allow the immediate next step (no skipping).
    if (targetIndex !== activeStepIndex + 1) return false;
    return !blockingIssues.some((issue) => stepIndex(issue.step) < targetIndex);
  };

  const goToStep = (stepKey: StepKey) => {
    if (canEnterStep(stepKey)) setActiveStep(stepKey);
  };

  const goNext = () => {
    if (currentStepHasError || isUploading) return;
    const nextStep = STEP_DEFS[activeStepIndex + 1];
    if (nextStep) setActiveStep(nextStep.key);
  };

  const goBack = () => {
    const previousStep = STEP_DEFS[activeStepIndex - 1];
    if (previousStep) setActiveStep(previousStep.key);
  };

  useEffect(() => {
    setPreviewImageIndex((index) => Math.min(index, Math.max(previewImages.length - 1, 0)));
  }, [previewImages.length]);

  const update = (patch: Partial<FormState>) => setForm((current) => ({ ...current, ...patch }));
  const updateCategory = (value: string) => {
    if (value === CUSTOM_CATEGORY_VALUE) {
      update({ categoryMode: 'custom', category: '' });
      return;
    }
    update({ categoryMode: 'preset', category: value });
  };
  const updateTrustCard = (key: TrustCardKey, patch: Partial<TrustCardDraft>) => setForm((current) => ({
    ...current,
    trustCards: { ...current.trustCards, [key]: { ...current.trustCards[key], ...patch } },
  }));

  const uploadFile = async (file: File, target: 'main' | 'gallery' | TrustCardKey) => {
    const uploadKey = target === 'main' || target === 'gallery' ? target : `trust-${target}`;
    if (!file.type.startsWith('image/')) {
      showToast({ tone: 'danger', title: '上传失败', description: '请选择图片文件。' });
      return;
    }
    if (target === 'gallery' && form.gallery.length >= 6) {
      showToast({ tone: 'warning', title: '轮播图已满', description: '最多上传 6 张轮播图。' });
      return;
    }
    setUploading((current) => ({ ...current, [uploadKey]: true }));
    try {
      const asset = await uploadImage(file, { roomId, bizType: target === 'main' ? 'lot_image' : target === 'gallery' ? 'lot_gallery' : 'trust_card' });
      if (target === 'main') {
        if (form.mainImageAssetId) void deleteUploadedImage(form.mainImageAssetId, { silent: true });
        update({ imageUrl: asset.imageUrl, mainImageAssetId: asset.id });
        setPreviewImageIndex(0);
      } else if (target === 'gallery') {
        setForm((current) => ({ ...current, gallery: [...current.gallery, imageFromAsset(asset, file.name)] }));
      } else {
        const previous = form.trustCards[target].assetId;
        if (previous) void deleteUploadedImage(previous, { silent: true });
        updateTrustCard(target, { imageUrl: asset.imageUrl, assetId: asset.id });
      }
      showToast({ tone: 'success', title: '图片已上传', description: file.name });
    } catch (e) {
      showToast({ tone: 'danger', title: '上传失败', description: resultMessage(e) });
    } finally {
      setUploading((current) => ({ ...current, [uploadKey]: false }));
    }
  };

  const handleFile = (event: ChangeEvent<HTMLInputElement>, target: 'main' | 'gallery' | TrustCardKey) => {
    const file = event.currentTarget.files?.[0];
    event.currentTarget.value = '';
    if (file) void uploadFile(file, target);
  };

  const removeMainImage = () => {
    if (form.mainImageAssetId) void deleteUploadedImage(form.mainImageAssetId, { silent: true });
    update({ imageUrl: '', mainImageAssetId: undefined });
  };

  const removeGalleryImage = (index: number) => {
    const image = form.gallery[index];
    if (image?.assetId) void deleteUploadedImage(image.assetId, { silent: true });
    setForm((current) => ({ ...current, gallery: current.gallery.filter((_, itemIndex) => itemIndex !== index) }));
  };

  const moveGalleryImage = (index: number, direction: -1 | 1) => {
    setForm((current) => {
      const nextIndex = index + direction;
      if (nextIndex < 0 || nextIndex >= current.gallery.length) return current;
      const gallery = [...current.gallery];
      [gallery[index], gallery[nextIndex]] = [gallery[nextIndex], gallery[index]];
      return { ...current, gallery };
    });
  };

  const removeTrustImage = (key: TrustCardKey) => {
    const assetId = form.trustCards[key].assetId;
    if (assetId) void deleteUploadedImage(assetId, { silent: true });
    updateTrustCard(key, { imageUrl: '', assetId: undefined });
  };

  const clearForm = () => {
    const assetIds = [form.mainImageAssetId, ...form.gallery.map((image) => image.assetId), ...TRUST_CARD_DEFS.map((card) => form.trustCards[card.key].assetId)].filter(Boolean) as string[];
    assetIds.forEach((assetId) => void deleteUploadedImage(assetId, { silent: true }));
    setForm(initialForm);
    setActiveStep('product');
    setPreviewImageIndex(0);
    setError('');
  };

  const submit = async () => {
    if (hasError || isUploading) return;
    setSubmitting(true);
    setError('');
    try {
      const draft = await createDraftLot({ roomId });
      const saved = await patchDraftLot(draft.id, toRequest(form, roomId));
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
    <StudioCard padding="lg" className="auctionCreateTitleBar auctionCreateHeader">
      <StudioPageHeader eyebrow="Create lot" title="添加拍品" actions={<a className="studioButton studioButton-secondary studioButton-md" href="/admin/auctions">返回队列</a>} />
    </StudioCard>
    {error ? <div className="auctionMgmtNotice danger"><AlertTriangle size={16} />{error}</div> : null}
    <nav className="publishStepper auctionCreateStepper" aria-label="添加拍品步骤">
      {STEP_DEFS.map((step, index) => {
        const stepIssues = issues.filter((issue) => issue.step === step.key);
        const hasStepError = stepIssues.some((issue) => issue.level === 'error');
        const hasStepWarning = stepIssues.some((issue) => issue.level === 'warning');
        const isActive = activeStep === step.key;
        const isDone = index < activeStepIndex && !hasStepError;
        const isRisk = hasStepWarning && !hasStepError;
        const isLocked = !canEnterStep(step.key);
        const lockedIssue = blockingIssues.find((issue) => stepIndex(issue.step) < index);
        const stepStatus = isDone ? '已完成' : isRisk ? '需检查' : isActive ? '当前' : '待完成';
        return <button key={step.key} type="button" disabled={isLocked || submitting} title={lockedIssue ? `请先完成：${lockedIssue.text}` : undefined} aria-current={isActive ? 'step' : undefined} className={`${isActive ? 'active' : ''} ${isDone ? 'done' : ''} ${isRisk ? 'risk' : ''} ${isLocked ? 'locked' : ''}`.trim()} onClick={() => goToStep(step.key)}>
          <b>{isDone ? <CheckCircle2 size={16} /> : index + 1}</b>
          <span>{step.label}</span>
          <small>{step.hint}</small>
          <em className="stepStatus">{stepStatus}</em>
        </button>;
      })}
    </nav>
    <div className="auctionCreateLayout">
      <main className="auctionCreateMain">
        {activeStep === 'product' ? <section className="auctionStepCard productInfoWorkbench">
          <header><div><p>Product assets</p><h3>拍品资料与素材</h3></div><div className="auctionStepActions"><StudioButton type="button" variant="secondary" onClick={clearForm} disabled={submitting || isUploading}>清空</StudioButton><StudioButton type="button" variant="primary" icon={<ChevronRight size={15} />} disabled={currentStepHasError || isUploading || submitting} onClick={goNext}>{isUploading ? '等待图片上传完成' : '下一步'}</StudioButton></div></header>
          <div className="productCockpitGrid">
            <section className="productPanel mediaPanel">
              <h4>拍品图片</h4>
              <AuctionField label="拍品主图" error={issueText(issues, '主图')} className="fieldMainImage">
                <div className={`auctionUpload mainImageUpload ${form.imageUrl ? 'hasImage' : ''} ${uploading.main ? 'isUploading' : ''}`}>
                  {form.imageUrl ? <img src={form.imageUrl} alt={form.title || '拍品主图'} /> : <ImagePlus size={34} />}
                  <b>{form.imageUrl ? '主图已上传' : '点击上传主图'}</b>
                  {uploading.main ? <span>上传中</span> : null}
                  <input type="file" accept="image/*" disabled={uploading.main || submitting} onChange={(event) => handleFile(event, 'main')} />
                </div>
                {form.imageUrl ? <div className="mainImageControl"><span>{shortURL(form.imageUrl)}</span><button type="button" disabled={submitting} onClick={removeMainImage}>移除</button></div> : null}
              </AuctionField>
              <AuctionField label="轮播图" help="最多 6 张，按当前顺序展示。" className="fieldCarousel">
                <div className={`auctionUpload carouselUpload ${uploading.gallery ? 'isUploading' : ''}`}>
                  <UploadCloud size={18} /><span>{uploading.gallery ? '上传中' : '上传轮播图'}</span>
                  <input type="file" accept="image/*" disabled={uploading.gallery || submitting || form.gallery.length >= 6} onChange={(event) => handleFile(event, 'gallery')} />
                </div>
                <div className="galleryThumbList">
                  {form.gallery.map((image, index) => <div key={`${image.imageUrl}-${index}`}><img src={image.imageUrl} alt={`轮播图 ${index + 1}`} /><span>#{index + 1}</span><button type="button" disabled={index === 0 || submitting} onClick={() => moveGalleryImage(index, -1)} aria-label="上移轮播图"><ArrowUp size={14} /></button><button type="button" disabled={index === form.gallery.length - 1 || submitting} onClick={() => moveGalleryImage(index, 1)} aria-label="下移轮播图"><ArrowDown size={14} /></button><button type="button" disabled={submitting} onClick={() => removeGalleryImage(index)} aria-label="删除轮播图"><Trash2 size={14} /></button></div>)}
                </div>
              </AuctionField>
            </section>
            <section className="productPanel basePanel">
              <h4>基础资料</h4>
              <div className="baseInfoGrid">
                <AuctionField label="拍品名称" error={issueText(issues, '名称')} className="fieldTitle"><input value={form.title} onChange={(e) => update({ title: e.target.value })} placeholder="请输入竞拍拍品名称" /></AuctionField>
                <AuctionField label="分类" error={issueText(issues, '分类')} className="fieldCategory">
                  <select value={form.categoryMode === 'custom' ? CUSTOM_CATEGORY_VALUE : form.category} onChange={(e) => updateCategory(e.target.value)}>
                    <option value="">请选择拍品分类</option>
                    {CATEGORY_OPTIONS.map((category) => <option key={category} value={category}>{category}</option>)}
                    <option value={CUSTOM_CATEGORY_VALUE}>其他</option>
                  </select>
                  {form.categoryMode === 'custom' ? <input value={form.category} onChange={(e) => update({ category: e.target.value })} placeholder="请输入自定义分类" /> : null}
                </AuctionField>
                <AuctionField label="标签" help="用逗号分隔。" className="fieldTags"><input value={form.tags} onChange={(e) => update({ tags: e.target.value })} placeholder="保真, 稀缺, 福利场" /></AuctionField>
                <AuctionField label="参考估价（元）" className="fieldEstimate"><input type="number" value={form.estimatePrice} min={0} placeholder="可选" onChange={(e) => update({ estimatePrice: e.target.value === '' ? '' : Number(e.target.value) })} /></AuctionField>
                <AuctionField label="库存" error={issueText(issues, '库存')} className="fieldStock"><input type="number" value={form.stock} min={1} onChange={(e) => update({ stock: Number(e.target.value) })} /></AuctionField>
                <AuctionField label="拍品介绍" error={issueText(issues, '介绍')} className="fieldDescription"><textarea value={form.description} onChange={(e) => update({ description: e.target.value })} rows={5} placeholder="描述材质、成色、亮点和竞拍价值" /></AuctionField>
                <AuctionField label="售后说明" className="fieldService"><textarea value={form.afterSaleNotes} onChange={(e) => update({ afterSaleNotes: e.target.value })} rows={3} placeholder="成交后确认、发货、保价、客服承诺等" /></AuctionField>
              </div>
            </section>
          </div>
        </section> : null}
        {activeStep === 'rules' ? <section className="auctionStepCard ruleWorkbench">
          <header><div><p>Auction rules</p><h3>竞拍规则</h3></div><div className="auctionStepActions"><StudioButton type="button" variant="secondary" onClick={clearForm} disabled={submitting || isUploading}>清空</StudioButton><StudioButton type="button" variant="secondary" icon={<ChevronLeft size={15} />} onClick={goBack} disabled={!canGoBack || submitting}>上一步</StudioButton><StudioButton type="button" variant="primary" icon={<ChevronRight size={15} />} disabled={currentStepHasError || isUploading || submitting} onClick={goNext}>{isUploading ? '等待图片上传完成' : '下一步'}</StudioButton></div></header>
          <div className="ruleFieldGrid">
            <StudioField label="起拍价（元）"><input type="number" value={form.startPrice} min={0} onChange={(e) => update({ startPrice: Number(e.target.value) })} /></StudioField>
            <StudioField label="加价幅度（元）" error={issueText(issues, '加价')}><input type="number" value={form.minIncrement} min={1} onChange={(e) => update({ minIncrement: Number(e.target.value) })} /></StudioField>
            <StudioField label="封顶价（元）" error={issueText(issues, '封顶')}><input type="number" value={form.capPrice} placeholder="可选" onChange={(e) => update({ capPrice: e.target.value === '' ? '' : Number(e.target.value) })} /></StudioField>
            <StudioField label="保证金（元）" error={issueText(issues, '保证金')}><input type="number" value={form.depositAmount} min={0} onChange={(e) => update({ depositAmount: Number(e.target.value) })} /></StudioField>
            <StudioField label="竞拍时长（秒）" error={issueText(issues, '时长')}><input type="number" value={form.durationSeconds} min={60} onChange={(e) => update({ durationSeconds: Number(e.target.value) })} /></StudioField>
            <StudioField label="延时窗口（秒）" error={issueText(issues, '延时窗口')}><input type="number" value={form.antiSnipeWindowSeconds} min={1} onChange={(e) => update({ antiSnipeWindowSeconds: Number(e.target.value) })} /></StudioField>
            <StudioField label="每次延长（秒）" error={issueText(issues, '每次延长')}><input type="number" value={form.antiSnipeExtendSeconds} min={10} max={30} onChange={(e) => update({ antiSnipeExtendSeconds: Number(e.target.value) })} /></StudioField>
            <StudioField label="最大延时次数" error={issueText(issues, '最大延时')}><input type="number" value={form.maxExtendCount} min={1} onChange={(e) => update({ maxExtendCount: Number(e.target.value) })} /></StudioField>
          </div>
        </section> : null}
        {activeStep === 'briefing' ? <section className="auctionStepCard liveBriefingWorkbench">
          <header><div><p>Briefing cards</p><h3>主播讲解</h3></div><div className="auctionStepActions"><StudioButton type="button" variant="secondary" onClick={clearForm} disabled={submitting || isUploading}>清空</StudioButton><StudioButton type="button" variant="secondary" icon={<ChevronLeft size={15} />} onClick={goBack} disabled={!canGoBack || submitting}>上一步</StudioButton><StudioButton type="button" variant="primary" icon={<ChevronRight size={15} />} disabled={currentStepHasError || isUploading || submitting} onClick={goNext}>{isUploading ? '等待图片上传完成' : '下一步'}</StudioButton></div></header>
          <section className="productPanel trustPanel">
            <div className="trustCardGrid">
              {TRUST_CARD_DEFS.map((card) => {
                const item = form.trustCards[card.key];
                return <AuctionField key={card.key} label={card.label} className="trustCardField">
                  <textarea value={item.content} onChange={(e) => updateTrustCard(card.key, { content: e.target.value })} rows={3} placeholder={card.placeholder} />
                  <div className={`trustImageSlot ${item.imageUrl ? 'hasImage' : ''} ${uploading[`trust-${card.key}`] ? 'isUploading' : ''}`}>
                    {item.imageUrl ? <img src={item.imageUrl} alt={`${card.label}图片`} /> : <UploadCloud size={16} />}
                    <span>{item.imageUrl ? '图片已上传' : '上传图片'}</span>
                    <input type="file" accept="image/*" disabled={uploading[`trust-${card.key}`] || submitting} onChange={(event) => handleFile(event, card.key)} />
                  </div>
                  {item.imageUrl ? <button className="trustImageRemove" type="button" disabled={submitting} onClick={() => removeTrustImage(card.key)}>移除图片</button> : null}
                </AuctionField>;
              })}
            </div>
          </section>
        </section> : null}
        {activeStep === 'review' ? <section className="auctionStepCard publishReviewWorkbench">
          <header><div><p>Final review</p><h3>确认发布</h3></div><div className="auctionStepActions"><StudioButton type="button" variant="secondary" onClick={clearForm} disabled={submitting || isUploading}>清空</StudioButton><StudioButton type="button" variant="secondary" icon={<ChevronLeft size={15} />} onClick={goBack} disabled={!canGoBack || submitting}>上一步</StudioButton><StudioButton type="button" variant="primary" loading={submitting} disabled={hasError || isUploading} onClick={() => void submit()}>{submitting ? '正在加入本场队列...' : isUploading ? '等待图片上传完成' : '加入本场队列'}</StudioButton></div></header>
          <div className="publishReviewGrid">
            <div className="publishSummaryBlock">
              <h4>入队摘要</h4>
              <div><span>直播间</span><b>{roomName}</b></div>
              <div><span>拍品</span><b>{form.title || '未填写'}</b></div>
              <div><span>分类</span><b>{form.category.trim() || '未选择'}</b></div>
              <div><span>图片素材</span><b>{Number(Boolean(form.imageUrl)) + form.gallery.length} 张</b></div>
              <div><span>讲解卡</span><b>{trustCards.length} 张</b></div>
            </div>
            <div className="publishIssueBox">
              <h4>发布检查</h4>
              {issues.length ? issues.map((issue) => <div key={issue.text} className={issue.level}><AlertTriangle size={15} /><span>{issue.text}</span></div>) : <div className="success"><CheckCircle2 size={15} /><span>核心配置已通过</span></div>}
            </div>
          </div>
        </section> : null}

      </main>
      <aside className="stickyPreviewPanel">
        <div className="mobilePreviewWrap">
          <div className="mobileAuctionPhone h5AuctionPreview">
            <section className="phoneLotCard">
              <div className="phoneImage">
                {previewImage ? <img src={previewImage.imageUrl} alt={`${previewImage.label}预览`} /> : <ImagePlus size={28} />}
                {previewImages.length > 1 ? <>
                  <button className="phoneImageNav prev" type="button" aria-label="上一张预览图" onClick={() => setPreviewImageIndex((index) => (index + previewImages.length - 1) % previewImages.length)}><ChevronLeft size={15} /></button>
                  <button className="phoneImageNav next" type="button" aria-label="下一张预览图" onClick={() => setPreviewImageIndex((index) => (index + 1) % previewImages.length)}><ChevronRight size={15} /></button>
                </> : null}
              </div>
              {previewImages.length > 1 ? <div className="phoneCarouselStrip">{previewImages.map((image, index) => <button key={`${image.imageUrl}-${index}`} type="button" className={index === previewImageIndex ? 'active' : ''} onClick={() => setPreviewImageIndex(index)}><img src={image.imageUrl} alt={image.label} /><span>{index + 1}</span></button>)}</div> : null}
              <div className="phoneLotInfo">
                <h4>{form.title || '拍品名称待填写'}</h4>
              </div>
              {tagList.length ? <div className="phoneRanking">{tagList.slice(0, 3).map((tag) => <span key={tag}>{tag}</span>)}</div> : null}
              <div className="phonePriceGrid">
                <div><span>当前价</span><b>{activeStep === 'product' ? '待配置' : formatMoneyText(auctionMoney(form.startPrice))}</b></div>
                <div><span>倒计时</span><b>{activeStep === 'product' ? '等待开拍' : `${form.durationSeconds}s`}</b></div>
                <div><span>起拍价</span><b>{activeStep === 'product' ? '待配置' : formatMoneyText(auctionMoney(form.startPrice))}</b></div>
                <div><span>加价幅度</span><b>{activeStep === 'product' ? '待配置' : formatMoneyText(auctionMoney(form.minIncrement))}</b></div>
                <div><span>保证金</span><b>{activeStep === 'product' ? '待配置' : formatMoneyText(auctionMoney(form.depositAmount))}</b></div>
              </div>
              <button type="button">立即出价</button>
            </section>
          </div>
        </div>
        {activeStep === 'rules' || activeStep === 'review' ? <StudioCard title="规则摘要" subtitle="Summary" padding="md"><div className="ruleSnapshotGrid"><div><span>起拍价</span><b>{formatMoneyText(auctionMoney(form.startPrice))}</b></div><div><span>加价幅度</span><b>{formatMoneyText(auctionMoney(form.minIncrement))}</b></div><div><span>封顶价</span><b>{form.capPrice === '' ? '未设置' : formatMoneyText(auctionMoney(form.capPrice))}</b></div><div><span>保证金</span><b>{formatMoneyText(auctionMoney(form.depositAmount))}</b></div><div><span>库存</span><b>{form.stock}</b></div><div><span>最后出价延时</span><b>{form.antiSnipeWindowSeconds}s / +{form.antiSnipeExtendSeconds}s</b></div></div></StudioCard> : null}
      </aside>
    </div>
  </section>;
}

function AuctionField({ label, help, error, children, className = '' }: { label: string; help?: string; error?: string; children: ReactNode; className?: string }) {
  return <div className={`auctionField ${className}`.trim()}><span>{label}</span>{children}{help ? <small>{help}</small> : null}{error ? <em>{error}</em> : null}</div>;
}

function imageFromAsset(asset: UploadedAsset, fileName: string): UploadedImage {
  return { assetId: asset.id, imageUrl: asset.imageUrl, fileName, sizeBytes: asset.sizeBytes };
}

function auctionMoney(amount: number | ''): Money {
  return { amount: Math.max(0, Math.round(Number(amount || 0) * 100)), currency: 'CNY' };
}

function optionalMoney(amount: number | ''): Money | undefined {
  if (amount === '') return undefined;
  return auctionMoney(amount);
}

function buildTrustCards(form: FormState): CreateLotRequest['trustCards'] {
  return TRUST_CARD_DEFS.flatMap((card) => {
    const draft = form.trustCards[card.key];
    if (!draft.content.trim() && !draft.imageUrl.trim()) return [];
    return [{
      id: `${card.key}-card`,
      type: card.type,
      title: card.title,
      content: draft.content.trim(),
      ...(draft.imageUrl.trim() ? { imageUrl: draft.imageUrl.trim() } : {}),
    }];
  });
}

function toRequest(form: FormState, roomId: string): CreateLotRequest {
  const request: CreateLotRequest = {
    roomId,
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
    trustCards: buildTrustCards(form),
    galleryImageUrls: form.gallery.map((image) => image.imageUrl),
    category: form.category.trim(),
    tags: parseTags(form.tags),
    stock: form.stock,
    afterSaleNotes: form.afterSaleNotes.trim(),
    depositAmount: auctionMoney(form.depositAmount),
  };
  const estimatePrice = optionalMoney(form.estimatePrice);
  if (estimatePrice) request.estimatePrice = estimatePrice;
  return request;
}

function validate(form: FormState): FormIssue[] {
  const issues: FormIssue[] = [];
  if (!form.title.trim()) issues.push({ level: 'error', step: 'product', text: '拍品名称必填' });
  if (!form.category.trim()) issues.push({ level: 'error', step: 'product', text: form.categoryMode === 'custom' ? '请填写自定义分类' : '请选择拍品分类' });
  if (!form.imageUrl.trim()) issues.push({ level: 'error', step: 'product', text: '主图必须上传' });
  if (form.imageUrl && !isHTTPImageURL(form.imageUrl)) issues.push({ level: 'error', step: 'product', text: '主图必须是 TOS 返回的 http/https URL' });
  form.gallery.forEach((image, index) => {
    if (!isHTTPImageURL(image.imageUrl)) issues.push({ level: 'error', step: 'product', text: `轮播图 ${index + 1} 不是稳定 URL` });
  });
  if (form.gallery.length > 6) issues.push({ level: 'error', step: 'product', text: '轮播图最多 6 张' });
  for (const card of TRUST_CARD_DEFS) {
    const imageURL = form.trustCards[card.key].imageUrl;
    if (imageURL && !isHTTPImageURL(imageURL)) issues.push({ level: 'error', step: 'briefing', text: `${card.label}图片不是稳定 URL` });
  }
  if (!form.description.trim()) issues.push({ level: 'error', step: 'product', text: '拍品介绍必填' });
  if (form.stock < 1) issues.push({ level: 'error', step: 'product', text: '库存必须大于等于 1' });
  if (form.minIncrement <= 0) issues.push({ level: 'error', step: 'rules', text: '加价幅度必须大于 0' });
  if (form.depositAmount < 0) issues.push({ level: 'error', step: 'rules', text: '保证金不能小于 0' });
  if (form.durationSeconds < 60) issues.push({ level: 'error', step: 'rules', text: '竞拍时长必须大于等于 60 秒' });
  if (form.antiSnipeWindowSeconds <= 0) issues.push({ level: 'error', step: 'rules', text: '延时窗口必须大于 0 秒' });
  if (form.antiSnipeExtendSeconds < 10 || form.antiSnipeExtendSeconds > 30) issues.push({ level: 'error', step: 'rules', text: '每次延长必须在 10-30 秒之间' });
  if (form.maxExtendCount <= 0) issues.push({ level: 'error', step: 'rules', text: '最大延时次数必须大于 0' });
  if (form.capPrice !== '' && form.capPrice <= form.startPrice) issues.push({ level: 'error', step: 'rules', text: '封顶价必须大于起拍价' });
  if (!buildTrustCards(form).length) issues.push({ level: 'warning', step: 'briefing', text: '建议至少补充一张讲解卡' });
  return issues;
}

function isHTTPImageURL(value: string) {
  if (value.startsWith('blob:') || value.startsWith('data:')) return false;
  try {
    const url = new URL(value);
    return url.protocol === 'http:' || url.protocol === 'https:';
  } catch {
    return false;
  }
}

function parseTags(value: string) {
  return value.split(/[,，]/).map((tag) => tag.trim()).filter(Boolean);
}

function stepIndex(stepKey: StepKey) {
  return STEP_DEFS.findIndex((step) => step.key === stepKey);
}

function isBlockingIssue(issue: FormIssue) {
  return issue.level === 'error';
}

function issueText(issues: FormIssue[], keyword: string) {
  return issues.find((issue) => issue.text.includes(keyword))?.text;
}

function shortURL(value: string) {
  if (value.length <= 54) return value;
  return `${value.slice(0, 28)}...${value.slice(-18)}`;
}
