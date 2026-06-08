import { useEffect, useMemo, useRef, useState } from 'react';
import './shop-replica.css';

type VisualKind = 'socket' | 'tv' | 'sweater' | 'socks' | 'storage' | 'beauty' | 'snack' | 'phone';
type DetailTab = '商品' | '评价' | '详情' | '推荐';

type DetailProduct = {
  id: string;
  name: string;
  price: string;
  realPrice: string;
  sold: string;
  visual: VisualKind;
  coverLabel: string;
  specs: string[];
};

type RecommendProduct = {
  id: string;
  name: string;
  price: string;
  sold: string;
  tag?: string;
  visual: VisualKind;
  coverLabel: string;
};

const PRODUCTS: DetailProduct[] = [
  {
    id: 'socket',
    name: '多功能电源插座 家用宿舍办公插排 直播热卖同款',
    price: '39.9',
    realPrice: '29.9',
    sold: '12.3万',
    visual: 'socket',
    coverLabel: '插座',
    specs: ['【10A】五孔插排1.8米', '【10A】三孔插排3米', '白色独立开关款'],
  },
  {
    id: 'tv-oled',
    name: '小米电视6 65英寸 OLED 超薄全面屏官方补贴',
    price: '470',
    realPrice: '429',
    sold: '2.7万',
    visual: 'tv',
    coverLabel: 'OLED',
    specs: ['65英寸 OLED', '官方标配', '含上门安装'],
  },
  {
    id: 'stripe-knit',
    name: '红白撞色条纹软糯针织上衣女秋季新款',
    price: '69.9',
    realPrice: '59.9',
    sold: '8.8万',
    visual: 'sweater',
    coverLabel: '针织',
    specs: ['红白条纹 S', '红白条纹 M', '红白条纹 L'],
  },
  {
    id: 'sp-socks',
    name: '男士SP中筒袜十双装 透气耐穿不易滑落',
    price: '8.8',
    realPrice: '8.8',
    sold: '10万+',
    visual: 'socks',
    coverLabel: '袜子',
    specs: ['【10双】男士SP中筒袜', '【5双】男士SP中筒袜', '黑白混色装'],
  },
  {
    id: 'makeup-set',
    name: '水光持妆粉底液套装 清透遮瑕自然妆感',
    price: '88',
    realPrice: '79',
    sold: '6.4万',
    visual: 'beauty',
    coverLabel: '美妆',
    specs: ['自然色', '亮肤色', '粉底液+散粉套装'],
  },
  {
    id: 'storage-box',
    name: '桌面透明收纳盒 多层防尘展示柜带抽屉',
    price: '29.9',
    realPrice: '24.9',
    sold: '4.2万',
    visual: 'storage',
    coverLabel: '收纳',
    specs: ['两层透明款', '三层透明款', '加宽抽屉款'],
  },
  {
    id: 'snack-box',
    name: '休闲零食大礼包 组合装办公室下午茶',
    price: '19.9',
    realPrice: '16.9',
    sold: '9.6万',
    visual: 'snack',
    coverLabel: '零食',
    specs: ['经典组合', '辣味组合', '甜口组合'],
  },
  {
    id: 'phone-card',
    name: '50元话费充值 即充即到账 全国通用',
    price: '49.8',
    realPrice: '49.5',
    sold: '18.5万',
    visual: 'phone',
    coverLabel: '话费',
    specs: ['50元话费', '100元话费', '200元话费'],
  },
];

const COMMENT_TAGS = [
  ['物美价廉', '29'],
  ['物流很好', '26'],
  ['推荐', '18'],
  ['商家服务好', '15'],
];

const COMMENTS = [
  {
    user: '花***栽',
    text: '东西不错质量也很好，性价比很高，良心商家，就冲这图必须给好评。',
    sku: 'china款/超值【买3双+送2双】共5双',
    visual: 'socks' as VisualKind,
    coverLabel: '实拍',
  },
  {
    user: '橘***海',
    text: '包装完整，页面里展示的细节都对得上，发货也很快。',
    sku: '升级款/家用办公室多场景',
    visual: 'socket' as VisualKind,
    coverLabel: '晒单',
  },
];

const SHOP_RECOMMENDS: RecommendProduct[] = [
  { id: 'tv-oled', name: '小米电视6 65英寸 OLED 超薄全面屏', price: '470', sold: '2.7万', visual: 'tv', coverLabel: 'OLED' },
  { id: 'stripe-knit', name: '红白撞色条纹软糯针织上衣女秋季新款', price: '69.9', sold: '8.8万', visual: 'sweater', coverLabel: '针织' },
  { id: 'sp-socks', name: '男士SP中筒袜十双装 透气耐穿', price: '8.8', sold: '10万+', visual: 'socks', coverLabel: '袜子' },
];

const WATERFALL: RecommendProduct[] = [
  { id: 'storage-box', name: '桌面透明收纳盒 多层防尘展示柜带抽屉', price: '29.9', sold: '4.2万', tag: '限时抢', visual: 'storage', coverLabel: '收纳' },
  { id: 'makeup-set', name: '水光持妆粉底液套装 清透遮瑕自然妆感', price: '88', sold: '6.4万', tag: '38节补贴', visual: 'beauty', coverLabel: '美妆' },
  { id: 'snack-box', name: '休闲零食大礼包 组合装办公室下午茶', price: '19.9', sold: '9.6万', tag: '近30天低价', visual: 'snack', coverLabel: '零食' },
  { id: 'phone-card', name: '50元话费充值 即充即到账 全国通用', price: '49.8', sold: '18.5万', tag: '极速到账', visual: 'phone', coverLabel: '话费' },
];

const DETAIL_NOTES = ['五孔间距升级', '儿童保护门', '阻燃外壳', '一体铜芯'];

function backToShop() {
  if (window.history.length > 1) {
    window.history.back();
    return;
  }
  window.location.assign('/shop');
}

function ArrowIcon() {
  return <span className="dyShopArrow" aria-hidden="true">›</span>;
}

function Price({ value }: { value: string }) {
  const [intPart, decimalPart = ''] = value.split('.');
  return (
    <span className="dyShopPrice">
      <i>￥</i>
      <b>{intPart}</b>
      {decimalPart ? <em>.{decimalPart}</em> : null}
    </span>
  );
}

function ProductPoster({
  visual,
  label,
  className = '',
}: {
  visual: VisualKind;
  label: string;
  className?: string;
}) {
  return (
    <div className={`dyShopPoster dyShopPoster-${visual} ${className}`}>
      <span className="dyShopPosterObject" aria-hidden="true">
        <i />
        <i />
        <b />
      </span>
      <strong>{label}</strong>
    </div>
  );
}

function findProduct() {
  const id = new URLSearchParams(window.location.search).get('id');
  return PRODUCTS.find((product) => product.id === id) ?? PRODUCTS[0];
}

export function ShopDetailPage() {
  const pageRef = useRef<HTMLElement | null>(null);
  const slidesRef = useRef<HTMLDivElement | null>(null);
  const productRef = useRef<HTMLElement | null>(null);
  const commentsRef = useRef<HTMLElement | null>(null);
  const detailRef = useRef<HTMLElement | null>(null);
  const recommendRef = useRef<HTMLElement | null>(null);
  const toastTimerRef = useRef<number | null>(null);
  const product = useMemo(() => findProduct(), []);
  const [headerProgress, setHeaderProgress] = useState(0);
  const [slideIndex, setSlideIndex] = useState(0);
  const [activeTab, setActiveTab] = useState<DetailTab>('商品');
  const [selectedSpec, setSelectedSpec] = useState(product.specs[0]);
  const [openIndexes, setOpenIndexes] = useState<number[]>([]);
  const [toast, setToast] = useState('');

  const heroSlides = useMemo(() => ([
    { visual: product.visual, label: product.coverLabel },
    { visual: product.visual, label: '细节' },
    { visual: 'storage' as VisualKind, label: '包装' },
    { visual: 'phone' as VisualKind, label: '保障' },
  ]), [product.coverLabel, product.visual]);

  useEffect(() => () => {
    if (toastTimerRef.current) {
      window.clearTimeout(toastTimerRef.current);
    }
  }, []);

  const showToast = (message: string) => {
    if (toastTimerRef.current) {
      window.clearTimeout(toastTimerRef.current);
    }
    setToast(message);
    toastTimerRef.current = window.setTimeout(() => setToast(''), 1600);
  };

  const handlePageScroll = () => {
    const top = pageRef.current?.scrollTop ?? 0;
    const nextProgress = Math.max(0, Math.min(1, top / 200));
    setHeaderProgress((current) => (Math.abs(current - nextProgress) < 0.015 ? current : nextProgress));
  };

  const handleSlideScroll = () => {
    const node = slidesRef.current;
    if (!node) return;
    const next = Math.round(node.scrollLeft / Math.max(1, node.clientWidth));
    setSlideIndex(Math.max(0, Math.min(heroSlides.length - 1, next)));
  };

  const toggleSection = (index: number) => {
    setOpenIndexes((current) => (
      current.includes(index) ? current.filter((item) => item !== index) : [...current, index]
    ));
  };

  const scrollToSection = (tab: DetailTab) => {
    const refMap = {
      商品: productRef,
      评价: commentsRef,
      详情: detailRef,
      推荐: recommendRef,
    };
    setActiveTab(tab);
    refMap[tab].current?.scrollIntoView({ behavior: 'smooth', block: 'start' });
  };

  return (
    <main ref={pageRef} className="mobileShell dyShopDetailShell" onScroll={handlePageScroll}>
      <header className="dyShopDetailTop dyShopDetailTopFloat" style={{ opacity: 1 - headerProgress }}>
        <button className="dyShopRoundButton" type="button" aria-label="返回" onClick={backToShop}>‹</button>
        <div className="dyShopDetailTopRight">
          <div className="dyShopDetailSearch dyShopDetailSearchGhost">
            <span>⌕</span>
            <b>{product.coverLabel}</b>
          </div>
          <button type="button" aria-label="搜索">⌕</button>
          <button type="button" aria-label="收藏">☆</button>
          <button type="button" aria-label="分享">↗</button>
        </div>
      </header>

      <header className="dyShopDetailTop dyShopDetailTopShadow" style={{ opacity: headerProgress }}>
        <div className="dyShopDetailTopRow">
          <button className="dyShopPlainButton" type="button" aria-label="返回" onClick={backToShop}>‹</button>
          <div className="dyShopDetailSearch">
            <span>⌕</span>
            <b>{product.coverLabel}</b>
          </div>
          <button type="button" aria-label="收藏">☆</button>
          <button type="button" aria-label="分享">↗</button>
        </div>
        <nav className="dyShopDetailTabs" aria-label="商品详情锚点">
          {(['商品', '评价', '详情', '推荐'] as DetailTab[]).map((tab) => (
            <button
              className={tab === activeTab ? 'isActive' : ''}
              type="button"
              key={tab}
              onClick={() => scrollToSection(tab)}
            >
              {tab}
            </button>
          ))}
        </nav>
      </header>

      <section ref={productRef} className="dyShopDetailHero" aria-label="商品图片">
        <div ref={slidesRef} className="dyShopDetailSlides" onScroll={handleSlideScroll}>
          {heroSlides.map((slide, index) => (
            <ProductPoster
              className="dyShopDetailHeroPoster"
              key={`${slide.label}-${index}`}
              label={slide.label}
              visual={slide.visual}
            />
          ))}
        </div>
        <span className="dyShopDetailIndex">{slideIndex + 1}/{heroSlides.length}</span>
      </section>

      <section className="dyShopDetailContent">
        <section className="dyShopDetailInfo">
          <div className="dyShopDetailPriceWrap">
            <Price value={product.price} />
            <span className="dyShopDetailCoupon">
              <small>热销款券后</small>
              <Price value={product.realPrice} />
            </span>
          </div>
          <h1>{product.name}</h1>
          <p>已售{product.sold}</p>
        </section>

        <section className="dyShopDetailCard dyShopDescCard">
          <article className="dyShopSpecRow">
            <b>保障</b>
            <p>假一赔四 · 运费险 · 极速退款</p>
            <ArrowIcon />
          </article>
          <article className="dyShopSpecRow">
            <b>选择</b>
            <div className="dyShopSpecScroller">
              {product.specs.map((spec) => (
                <button
                  className={spec === selectedSpec ? 'isActive' : ''}
                  type="button"
                  key={spec}
                  onClick={() => setSelectedSpec(spec)}
                >
                  {spec}
                </button>
              ))}
              <em>共{product.specs.length}种规格可选</em>
            </div>
            <ArrowIcon />
          </article>
          <article className="dyShopSpecRow dyShopSpecRowTall">
            <b>物流</b>
            <div>
              <p>发货 四川成都 <i>|</i> 免运费</p>
              <p>48小时内发货</p>
              <p className="dyShopMuted">送至 四川省成都市</p>
            </div>
            <ArrowIcon />
          </article>
          <article className="dyShopSpecRow">
            <b>参数</b>
            <p className="dyShopEllipsis">优惠新人券 立减4 额定功率 线长 适用场景 售后服务</p>
            <ArrowIcon />
          </article>
        </section>

        <section ref={commentsRef} className="dyShopDetailCard dyShopComments">
          <header>
            <h2>商品评论(507)</h2>
            <ArrowIcon />
          </header>
          <div className="dyShopCommentTags">
            {COMMENT_TAGS.map(([tag, count]) => (
              <span key={tag}>{tag} <i>{count}</i></span>
            ))}
          </div>
          {COMMENTS.map((comment) => (
            <article className="dyShopComment" key={comment.user}>
              <header>
                <span>{comment.user.slice(0, 1)}</span>
                <b>{comment.user}</b>
              </header>
              <div className="dyShopCommentBody">
                <div>
                  <p>{comment.text}</p>
                  <small>{comment.sku}</small>
                </div>
                <ProductPoster className="dyShopCommentPoster" visual={comment.visual} label={comment.coverLabel} />
              </div>
            </article>
          ))}
        </section>

        <section className="dyShopDetailCard dyShopStoreCard">
          <header>
            <span className="dyShopStoreLogo">店</span>
            <div>
              <h2>店铺名</h2>
              <p><em>金牌店铺</em><em>好评过千</em><em>销量超10万</em></p>
              <small>店铺口碑4.90分</small>
            </div>
            <button type="button">进店</button>
          </header>
          <div className="dyShopStoreScores">
            {['商品质量|商品评价一般', '物流速度|平均24小时发货', '服务体验|售后响应稳定'].map((item) => {
              const [label, value] = item.split('|');
              return (
                <span key={item}>
                  <small>{label}</small>
                  <b>{value}</b>
                </span>
              );
            })}
          </div>
          <section className="dyShopStoreRecommend">
            <header>
              <b>店铺推荐</b>
              <a href="/shop">查看全部 <ArrowIcon /></a>
            </header>
            <div>
              {SHOP_RECOMMENDS.map((item) => (
                <a href={`/shop/detail?id=${item.id}`} key={item.id}>
                  <ProductPoster className="dyShopStorePoster" visual={item.visual} label={item.coverLabel} />
                  <b>{item.name}</b>
                  <Price value={item.price} />
                </a>
              ))}
            </div>
          </section>
        </section>
      </section>

      <section ref={detailRef} className="dyShopDetailImages">
        <header><span />商品详情<span /></header>
        {DETAIL_NOTES.map((note, index) => (
          <ProductPoster
            className="dyShopDetailImagePoster"
            key={note}
            label={note}
            visual={index % 2 === 0 ? product.visual : 'storage'}
          />
        ))}
      </section>

      <section className="dyShopDetailPad">
        <section className="dyShopDetailCard dyShopAccordion">
          {[0, 1, 2].map((index) => (
            <article className={openIndexes.includes(index) ? 'isOpen' : ''} key={index}>
              <button type="button" onClick={() => toggleSection(index)}>
                <span>价格说明</span>
                <ArrowIcon />
              </button>
              <p>页面展示价格、券后价和活动价会随库存、优惠券、补贴活动发生变化，实际成交价以提交订单页展示为准。</p>
            </article>
          ))}
        </section>

        <section ref={recommendRef} className="dyShopOtherRecommend">
          <h2>你可能还会喜欢</h2>
          <div>
            {WATERFALL.map((item) => (
              <a href={`/shop/detail?id=${item.id}`} className="dyShopRecommendGoods" key={item.id}>
                <ProductPoster visual={item.visual} label={item.coverLabel} />
                <section>
                  <h3>{item.name}</h3>
                  {item.tag ? <em>{item.tag}</em> : null}
                  <p><b>￥{item.price}</b><span>已售{item.sold}件</span></p>
                </section>
              </a>
            ))}
          </div>
        </section>
      </section>

      <footer className="dyShopBuyToolbar">
        <div className="dyShopToolbarOptions">
          <button type="button"><span>⌂</span><b>进店</b></button>
          <button type="button"><span>☻</span><b>客服</b></button>
          <button type="button"><span>▱</span><b>购物车</b></button>
        </div>
        <div className="dyShopBuyButtons">
          <button type="button" onClick={() => showToast(`已加入购物车：${selectedSpec}`)}>加入购物车</button>
          <button type="button" onClick={() => showToast('已领取优惠券，准备下单')}>领券购买</button>
        </div>
      </footer>

      {toast ? <div className="dyShopToast" role="status">{toast}</div> : null}
    </main>
  );
}
