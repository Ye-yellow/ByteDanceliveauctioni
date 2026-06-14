package data

import (
	"context"
	"time"

	"gorm.io/gorm/clause"
)

type shopSeed struct {
	ID                  string
	Title               string
	Subtitle            string
	Description         string
	Category            string
	ShopName            string
	MainImageURL        string
	Tags                []string
	Badges              []string
	PriceAmount         int64
	OriginalPriceAmount int64
	SoldLabel           string
	Live                bool
	SKUs                []shopSeedSKU
}

type shopSeedSKU struct {
	ID          string
	Name        string
	PriceAmount int64
	Stock       int64
}

var shopSeeds = []shopSeed{
	{
		ID: "imperial-green-jade-bangle", Title: "冰阳绿翡翠手镯 正圈饱满细腻起光", Subtitle: "直播间同款，支持上手看货",
		Description: "精选冰阳绿翡翠手镯，圈口细腻，适合送礼和自戴。", Category: "珠宝玉石", ShopName: "Ggboy珠宝严选",
		MainImageURL: "/shop-assets/auction-lots/imperial-green-jade-bangle.png", Tags: []string{"翡翠手镯", "正在直播", "包邮"}, Badges: []string{"直播中", "旗舰"}, PriceAmount: 47000, OriginalPriceAmount: 59900, SoldLabel: "3.8万+", Live: true,
		SKUs: []shopSeedSKU{{ID: "sku-jade-bangle-54", Name: "54圈口", PriceAmount: 47000, Stock: 88}, {ID: "sku-jade-bangle-56", Name: "56圈口", PriceAmount: 49900, Stock: 65}},
	},
	{
		ID: "white-hetian-jade-bangle", Title: "白月光和田玉手镯 温润通透日常款", Subtitle: "低价开拍，同城优先发货",
		Description: "温润白玉手镯，颜色干净，适合日常搭配。", Category: "珠宝玉石", ShopName: "Yexieer珠宝店",
		MainImageURL: "/shop-assets/auction-lots/white-hetian-jade-bangle.png", Tags: []string{"和田玉", "低价开拍"}, Badges: []string{"补贴"}, PriceAmount: 8800, OriginalPriceAmount: 12900, SoldLabel: "6.4万+", Live: false,
		SKUs: []shopSeedSKU{{ID: "sku-white-jade-55", Name: "55圈口", PriceAmount: 8800, Stock: 120}, {ID: "sku-white-jade-57", Name: "57圈口", PriceAmount: 9300, Stock: 80}},
	},
	{
		ID: "carved-jade-pendant", Title: "天然翡翠平安扣项链 冰润飘花吊坠", Subtitle: "送礼好物，附精美礼盒",
		Description: "冰润翡翠平安扣，搭配简洁项链，适合作为礼物。", Category: "项链吊坠", ShopName: "Ggboy珠宝严选",
		MainImageURL: "/shop-assets/auction-lots/carved-jade-pendant-necklace.png", Tags: []string{"项链", "适合送礼"}, Badges: []string{"热卖"}, PriceAmount: 19900, OriginalPriceAmount: 25900, SoldLabel: "1.9万+", Live: true,
		SKUs: []shopSeedSKU{{ID: "sku-jade-pendant-gold", Name: "金色链条", PriceAmount: 19900, Stock: 76}, {ID: "sku-jade-pendant-silver", Name: "银色链条", PriceAmount: 18900, Stock: 72}},
	},
	{
		ID: "freshwater-pearl-necklace", Title: "淡水珍珠项链 近圆强光通勤气质款", Subtitle: "新娘礼物与日常通勤都合适",
		Description: "淡水珍珠项链，强光近圆，经典百搭。", Category: "项链吊坠", ShopName: "珍珠小姐旗舰店",
		MainImageURL: "/shop-assets/auction-lots/freshwater-pearl-necklace.png", Tags: []string{"珍珠", "送礼"}, Badges: []string{"新品"}, PriceAmount: 36800, OriginalPriceAmount: 42900, SoldLabel: "2.6万+", Live: false,
		SKUs: []shopSeedSKU{{ID: "sku-pearl-42", Name: "42cm", PriceAmount: 36800, Stock: 45}, {ID: "sku-pearl-45", Name: "45cm", PriceAmount: 38900, Stock: 38}},
	},
	{
		ID: "ruby-diamond-bracelet", Title: "红宝石钻石手链 18K金精致叠戴款", Subtitle: "小众设计，节日礼物",
		Description: "红宝石与钻石点缀，适合叠戴和节日送礼。", Category: "手链手串", ShopName: "璀璨宝石馆",
		MainImageURL: "/shop-assets/auction-lots/ruby-diamond-gold-bracelet.png", Tags: []string{"红宝石", "18K金"}, Badges: []string{"限时"}, PriceAmount: 69900, OriginalPriceAmount: 89900, SoldLabel: "9800+", Live: false,
		SKUs: []shopSeedSKU{{ID: "sku-ruby-bracelet-s", Name: "15cm", PriceAmount: 69900, Stock: 18}, {ID: "sku-ruby-bracelet-m", Name: "16.5cm", PriceAmount: 72900, Stock: 16}},
	},
	{
		ID: "sapphire-diamond-necklace", Title: "蓝宝石钻石项链 锁骨链高级感礼盒装", Subtitle: "直播精选，支持保价",
		Description: "蓝宝石钻石锁骨链，细节精致，礼盒装发货。", Category: "项链吊坠", ShopName: "璀璨宝石馆",
		MainImageURL: "/shop-assets/auction-lots/sapphire-diamond-necklace.png", Tags: []string{"蓝宝石", "礼盒"}, Badges: []string{"直播中"}, PriceAmount: 75900, OriginalPriceAmount: 99900, SoldLabel: "1.2万+", Live: true,
		SKUs: []shopSeedSKU{{ID: "sku-sapphire-necklace", Name: "礼盒装", PriceAmount: 75900, Stock: 22}},
	},
	{
		ID: "gift-jewelry-box", Title: "珠宝收纳盒 旅行便携首饰盒多层分区", Subtitle: "百元低价，收纳不打结",
		Description: "便携首饰盒，多层分区，适合项链、耳饰和戒指。", Category: "收纳配饰", ShopName: "好物研究所",
		MainImageURL: "/douyin-assets/images/6V2vxN6FH4QiuCQ3KMbP7.png", Tags: []string{"收纳", "百元低价"}, Badges: []string{"补贴"}, PriceAmount: 3990, OriginalPriceAmount: 5900, SoldLabel: "12.3万+", Live: false,
		SKUs: []shopSeedSKU{{ID: "sku-jewelry-box-pink", Name: "樱花粉", PriceAmount: 3990, Stock: 300}, {ID: "sku-jewelry-box-cream", Name: "奶油白", PriceAmount: 3990, Stock: 280}},
	},
	{
		ID: "silver-bracelet-stack", Title: "银手镯叠戴套装 轻奢百搭开口款", Subtitle: "低价开拍，适合通勤",
		Description: "轻奢银手镯套装，开口设计，适合日常叠戴。", Category: "手链手串", ShopName: "好物研究所",
		MainImageURL: "/douyin-assets/images/9K9Ioxl-NqRtr7Tx1L6mH.png", Tags: []string{"银饰", "低价开拍"}, Badges: []string{"新"}, PriceAmount: 12900, OriginalPriceAmount: 19900, SoldLabel: "8600+", Live: false,
		SKUs: []shopSeedSKU{{ID: "sku-silver-stack", Name: "三件套", PriceAmount: 12900, Stock: 150}},
	},
}

func (s *Store) EnsureShopSeeds(ctx context.Context) error {
	nowMs := time.Now().UnixMilli()
	for _, seed := range shopSeeds {
		product := ShopProductModel{
			ID:                  seed.ID,
			Title:               seed.Title,
			Subtitle:            seed.Subtitle,
			Description:         seed.Description,
			Category:            seed.Category,
			ShopName:            seed.ShopName,
			MainImageURL:        seed.MainImageURL,
			DetailImageURLs:     jsonText([]string{seed.MainImageURL}),
			Tags:                jsonText(seed.Tags),
			Badges:              jsonText(seed.Badges),
			PriceAmount:         seed.PriceAmount,
			OriginalPriceAmount: seed.OriginalPriceAmount,
			Currency:            "CNY",
			SoldLabel:           seed.SoldLabel,
			Live:                seed.Live,
			Status:              "active",
			CreatedAtUnixMs:     nowMs,
			UpdatedAtUnixMs:     nowMs,
		}
		if err := s.db.WithContext(ctx).Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"title", "subtitle", "description", "category", "shop_name", "main_image_url",
				"detail_image_urls", "tags", "badges", "price_amount", "original_price_amount",
				"currency", "sold_label", "live", "status", "updated_at_unix_ms",
			}),
		}).Create(&product).Error; err != nil {
			return err
		}
		for _, seedSKU := range seed.SKUs {
			sku := ShopSKUModel{
				ID:          seedSKU.ID,
				ProductID:   seed.ID,
				Name:        seedSKU.Name,
				PriceAmount: seedSKU.PriceAmount,
				Currency:    "CNY",
				Stock:       seedSKU.Stock,
			}
			if err := s.db.WithContext(ctx).Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "id"}},
				DoUpdates: clause.AssignmentColumns([]string{"product_id", "name", "price_amount", "currency", "stock"}),
			}).Create(&sku).Error; err != nil {
				return err
			}
		}
	}
	return nil
}
