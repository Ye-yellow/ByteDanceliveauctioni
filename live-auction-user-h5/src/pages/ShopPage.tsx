import { useMemo, useState } from 'react';
import { DouyinTabBar } from '../shared/ui/DouyinTabBar';
import './shop-replica.css';

type VisualKind = 'socket' | 'tv' | 'sweater' | 'socks' | 'storage' | 'beauty' | 'snack' | 'phone';

type ShopProduct = {
  id: string;
  category: string;
  name: string;
  price: string;
  sold: string;
  discount?: string;
  isLowPrice?: boolean;
  visual: VisualKind;
  coverLabel: string;
};

const SHOP_SHORTCUTS = [
  { label: '我的订单', icon: 'order', href: '/m/history?from=shop' },
  { label: '手机充值', icon: 'charge', href: '/shop?entry=recharge' },
  { label: '购物消息', icon: 'message', href: '/shop?entry=message' },
  { label: '小时达', icon: 'nearby', href: '/shop?entry=nearby' },
  { label: '退款/售后', icon: 'refund', href: '/shop?entry=refund' },
  { label: '潮流服饰', icon: 'fashion', href: '/shop?entry=fashion' },
];

const SHOP_CATEGORIES = ['精选', '38节补贴', '手机数码', '潮流服饰', '日用百货'];

const SHOP_PRODUCTS: ShopProduct[] = [
  {
    id: 'socket',
    category: '精选',
    name: '多功能电源插座 家用宿舍办公插排 直播热卖同款',
    price: '39.9',
    sold: '12.3万',
    discount: '券后价',
    isLowPrice: true,
    visual: 'socket',
    coverLabel: '插座',
  },
  {
    id: 'tv-oled',
    category: '手机数码',
    name: '小米电视6 65英寸 OLED 超薄全面屏官方补贴',
    price: '470',
    sold: '2.7万',
    discount: '官方立减',
    visual: 'tv',
    coverLabel: 'OLED',
  },
  {
    id: 'stripe-knit',
    category: '潮流服饰',
    name: '红白撞色条纹软糯针织上衣女秋季新款',
    price: '69.9',
    sold: '8.8万',
    discount: '满减',
    visual: 'sweater',
    coverLabel: '针织',
  },
  {
    id: 'sp-socks',
    category: '日用百货',
    name: '男士SP中筒袜十双装 透气耐穿不易滑落',
    price: '8.8',
    sold: '10万+',
    isLowPrice: true,
    visual: 'socks',
    coverLabel: '袜子',
  },
  {
    id: 'makeup-set',
    category: '38节补贴',
    name: '水光持妆粉底液套装 清透遮瑕自然妆感',
    price: '88',
    sold: '6.4万',
    discount: '38节补贴',
    visual: 'beauty',
    coverLabel: '美妆',
  },
  {
    id: 'storage-box',
    category: '日用百货',
    name: '桌面透明收纳盒 多层防尘展示柜带抽屉',
    price: '29.9',
    sold: '4.2万',
    discount: '限时抢',
    visual: 'storage',
    coverLabel: '收纳',
  },
  {
    id: 'snack-box',
    category: '38节补贴',
    name: '休闲零食大礼包 组合装办公室下午茶',
    price: '19.9',
    sold: '9.6万',
    isLowPrice: true,
    visual: 'snack',
    coverLabel: '零食',
  },
  {
    id: 'phone-card',
    category: '手机数码',
    name: '50元话费充值 即充即到账 全国通用',
    price: '49.8',
    sold: '18.5万',
    discount: '极速到账',
    visual: 'phone',
    coverLabel: '话费',
  },
];

const SUBSIDY_PRODUCTS = [
  SHOP_PRODUCTS[1],
  SHOP_PRODUCTS[4],
  SHOP_PRODUCTS[3],
  SHOP_PRODUCTS[6],
];

function ProductPoster({ product, compact = false }: { product: ShopProduct; compact?: boolean }) {
  return (
    <div className={`dyShopPoster dyShopPoster-${product.visual}${compact ? ' dyShopPosterCompact' : ''}`}>
      <span className="dyShopPosterObject" aria-hidden="true">
        <i />
        <i />
        <b />
      </span>
      <strong>{product.coverLabel}</strong>
    </div>
  );
}

export function ShopPage() {
  const [activeCategory, setActiveCategory] = useState('精选');
  const [query, setQuery] = useState('50元话费充值');
  const [submittedQuery, setSubmittedQuery] = useState('');

  const visibleProducts = useMemo(() => {
    const keyword = submittedQuery.trim();
    return SHOP_PRODUCTS.filter((product) => {
      const matchCategory = activeCategory === '精选' || product.category === activeCategory;
      const matchKeyword = !keyword || product.name.includes(keyword) || product.category.includes(keyword);
      return matchCategory && matchKeyword;
    });
  }, [activeCategory, submittedQuery]);

  const handleSearch = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setSubmittedQuery(query.trim() === '50元话费充值' ? '话费' : query.trim());
    setActiveCategory('精选');
  };

  return (
    <main className="mobileShell dyShopShell">
      <header className="dyShopSearchBar">
        <form className="dyShopSearchBox" onSubmit={handleSearch}>
          <span className="dyShopSearchGlyph" aria-hidden="true">⌕</span>
          <input
            aria-label="搜索商品"
            value={query}
            onChange={(event) => setQuery(event.target.value)}
          />
          <button className="dyShopCameraButton" type="button" aria-label="拍照搜索">▧</button>
          <button className="dyShopSearchButton" type="submit">搜索</button>
        </form>
        <a className="dyShopCartButton" href="/shop?panel=cart" aria-label="购物车">⌑</a>
      </header>

      <nav className="dyShopCategoryTabs" aria-label="商城分类">
        {SHOP_CATEGORIES.map((category) => (
          <button
            className={category === activeCategory ? 'isActive' : ''}
            type="button"
            key={category}
            onClick={() => {
              setActiveCategory(category);
              setSubmittedQuery('');
            }}
          >
            {category}
          </button>
        ))}
      </nav>

      <section className="dyShopCard dyShopShortcutCard" aria-label="商城快捷入口">
        <div className="dyShopShortcutScroller">
          {SHOP_SHORTCUTS.map((item) => (
            <a href={item.href} key={item.label}>
              <span className={`dyShopShortcutIcon dyShopShortcutIcon-${item.icon}`} aria-hidden="true" />
              <b>{item.label}</b>
            </a>
          ))}
        </div>
      </section>

      <section className="dyShopCard dyShopSubsidyCard" aria-label="38节补贴">
        <div className="dyShopSubsidyTitle">
          <span>38</span>
          <b>节补贴</b>
        </div>
        {SUBSIDY_PRODUCTS.map((product) => (
          <a href={`/shop/detail?id=${product.id}`} key={product.id}>
            <ProductPoster product={product} compact />
            <span className="dyShopSubsidyPrice">￥{product.price}</span>
          </a>
        ))}
      </section>

      <section className="dyShopWaterfall" aria-label="商品推荐">
        {visibleProducts.map((product) => (
          <a href={`/shop/detail?id=${product.id}`} className="dyShopGoodsCard" key={product.id}>
            <ProductPoster product={product} />
            <section>
              <h2>{product.name}</h2>
              {product.discount ? <em>{product.discount}</em> : null}
              <p>
                <b>￥{product.price}</b>
                <span>已售{product.sold}件</span>
              </p>
              {product.isLowPrice ? <small>近30天低价</small> : null}
            </section>
          </a>
        ))}
      </section>

      {visibleProducts.length === 0 ? (
        <section className="dyShopEmpty">换个关键词试试</section>
      ) : null}

      <DouyinTabBar active="shop" />
    </main>
  );
}
