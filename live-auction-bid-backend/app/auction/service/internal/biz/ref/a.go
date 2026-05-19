//go:build ignore
// +build ignore

package biz

import (
	"context"
	"encoding/json"
	v2 "odin/api/loli/service/v2"
	"time"

	"github.com/awa/go-iap/appstore/api"
	"github.com/pkg/errors"
)

// RevertPurchaseFlow 状态常量
const (
	RevertPurchaseStatus_Pending    int32 = 0 // 待处理
	RevertPurchaseStatus_Processing int32 = 1 // 处理中
	RevertPurchaseStatus_Success    int32 = 2 // 成功
	RevertPurchaseStatus_Failed     int32 = 3 // 失败
)

type OrderRepo interface {
	CreateOrderRecord(ctx context.Context, record *OrderRecord) (int64, error)

	// 补偿
	RpcGetUsernameByYid(ctx context.Context, yid int64) (string, error)
	GetPreviousOrderRecordsByUsername(ctx context.Context, username string) ([]*PreviousOrderRecord, error)
	CheckPreviousUser(ctx context.Context, username string) (bool, error)

	CreateRevertPurchaseFlow(ctx context.Context, yid int64, revertData *v2.RevertPurchaseData) error
	UpdateRevertPurchaseFlow(ctx context.Context, yid int64, revertData *v2.RevertPurchaseData) error
	UpdateRevertPurchaseFlowStatus(ctx context.Context, yid int64, orderId string, status int32, moneyStoreKey, purchaseToken, failReason string) error
	GetRevertPurchaseFlowStatus(ctx context.Context, yid int64, orderId string) (*v2.RevertPurchaseFlowInfo, error)
	GetRevertPurchaseFlowList(ctx context.Context, yid int64) ([]*v2.RevertPurchaseFlowInfo, error)
	RetryRevertPurchaseFlow(ctx context.Context, yid int64, orderId string) error
	GetRevertPurchaseFlowData(ctx context.Context, yid int64, orderId string) (string, error)
}

type OrderRecord struct {
	Fid              string
	Yid              int64
	IsTest           bool
	StoreKey         string
	Channel          string
	TransactionID    string
	Receipt          string
	PlayStoreRespStr string // 解析后的数据
	AppStoreRespStr  string // 解析后的数据
}

type PreviousOrderRecord struct {
	Username   string
	StoreKey   string
	IsReturned bool
}

type Receipt struct {
	Payload       string `json:"Payload"`
	Store         string `json:"Store"`
	TransactionID string `json:"TransactionID"`
}

type ReceiptPayload struct {
	JSON       string   `json:"json"`
	Signature  string   `json:"signature"`
	SkuDetails []string `json:"skuDetails"`
}

type ReceiptPayloadJSON struct {
	OrderID       string `json:"orderId"`
	PackageName   string `json:"packageName"`
	ProductID     string `json:"productId"`
	PurchaseTime  int64  `json:"purchaseTime"`
	PurchaseState int    `json:"purchaseState"`
	PurchaseToken string `json:"purchaseToken"`
	Acknowledged  bool   `json:"acknowledged"`
}

type ReceiptPayloadSkuDetails struct {
	ProductID         string `json:"productId"`
	Type              string `json:"type"`
	Title             string `json:"title"`
	Name              string `json:"name"`
	Description       string `json:"description"`
	Price             string `json:"price"`
	PriceAmountMicros int    `json:"price_amount_micros"`
	PriceCurrencyCode string `json:"price_currency_code"`
	SkuDetailsToken   string `json:"skuDetailsToken"`
}

func (uc *UserUseCase) VerifyByOrderId(ctx context.Context, channel v2.AdminOrderChannel, OrderId string, productID string) (string, error) {

	switch channel {
	case v2.AdminOrderChannel_AdminOrderChannel_IOS:

		return "", nil

	case v2.AdminOrderChannel_AdminOrderChannel_Google:

		orderInfo, err := uc.playStore.GetOrder(ctx, OrderId)
		if err != nil {
			return "", errors.Wrapf(err, "playstore GetOrder error")
		}

		if orderInfo.State != "PROCESSED" {
			return "", errors.Errorf("订单状态异常, OrderState: %v (PROCESSED=已处理, REFUNDED=已退款, EXPIRED=已过期, CANCELED=已取消, DEFERRED=已推迟, PENDING=待处理)", orderInfo.State)
		}
		if orderInfo.LineItems[0].ProductId != productID {
			return "", errors.Errorf("商品不匹配, productID=%v, orderInfo.LineItems[0].ProductId=%v", productID, orderInfo.LineItems[0].ProductId)
		}

		return orderInfo.PurchaseToken, nil

	default:
		return "", errors.Errorf("不支持的平台类型: %v", channel)
	}
}

func (uc *UserUseCase) AcknowledgeOrder_Google(ctx context.Context, productID string, purchaseToken string) error {
	err := uc.playStore.AcknowledgeProduct_Google(ctx, productID, purchaseToken)
	if err != nil {
		return errors.Wrapf(err, "playstore AcknowledgeProduct_Google error")
	}

	return nil
}

func (uc *UserUseCase) VerifyOrder1(ctx context.Context, u *User, channel v2.OrderChannel, receiptBytes []byte, currencyType string, storeKey string) (int64, error) {
	return 0, errors.New("unsupported platform type")
}

func (uc *UserUseCase) VerifyOrder(ctx context.Context, u *User, channel v2.OrderChannel, receiptBytes []byte, currencyType string, storeKey string, afid string) (int64, error) {

	var (
		receipt      Receipt
		payloadInfo  ReceiptPayload
		payloadJson  ReceiptPayloadJSON
		skuDetails   ReceiptPayloadSkuDetails
		analysisData map[string]interface{}
		orderRecord  *OrderRecord
	)

	if err := json.Unmarshal(receiptBytes, &receipt); err != nil {
		return 0, errors.Wrapf(err, "json.Unmarshal receipt %v error", string(receiptBytes))
	}
	if receipt.Store == "fake" {
		return 0, nil
	}

	clientIp, err := GetClientIp(ctx)
	if err != nil {
		uc.log.Warnf("Error GetClientIp")
	}

	switch channel {
	case v2.OrderChannel_OrderChannel_IOS:
		if receipt.Store != "AppleAppStore" {
			return 0, errors.Errorf("Store not match %s", receipt.Store)
		}
		response, err := uc.appStore.GetTransactionInfo(ctx, receipt.TransactionID)
		if err != nil {
			return 0, errors.Wrapf(err, "iap GetTransactionInfo error")
		}

		transaction, err := uc.appStore.ParseSignedTransaction(response.SignedTransactionInfo)
		if err != nil {
			return 0, errors.Wrapf(err, "iap ParseSignedTransaction error")
		}

		bs, _ := json.MarshalIndent(transaction, "", "\t")
		uc.log.Debugf("AppleStore JWSTransaction: %v\n", string(bs))

		if transaction.TransactionID != receipt.TransactionID {
			return 0, errors.Wrapf(err, "iap validation failed: submission txid=%v, validation=%v", receipt.TransactionID, transaction.TransactionID)
		}

		// 判断是否为真实订单
		if transaction.Environment != api.Sandbox {
			analysisData = map[string]interface{}{
				"ip":            clientIp,
				"payment":       receipt.Store,
				"order_id":      receipt.TransactionID,
				"product":       storeKey,
				"currency_type": currencyType,
				"amount":        float64(transaction.Price) / 1000, // 该值表示您在 App Store Connect 中配置的应用内购买或订阅优惠的价格乘以 1000，得到一个整数值。
			}
		}
		orderRecord = &OrderRecord{
			Fid:             u.Fid,
			Yid:             u.Yid,
			IsTest:          transaction.Environment == api.Sandbox,
			StoreKey:        storeKey,
			Channel:         channel.String(),
			Receipt:         string(receiptBytes),
			TransactionID:   receipt.TransactionID,
			AppStoreRespStr: string(bs),
		}

	case v2.OrderChannel_OrderChannel_Google:
		if receipt.Store != "GooglePlay" {
			return 0, errors.Errorf("Store not match %s", receipt.Store)
		}
		if err := json.Unmarshal([]byte(receipt.Payload), &payloadInfo); err != nil {
			return 0, errors.Wrapf(err, "json.Unmarshal receipt.Payload %v error", receipt.Payload)
		}
		if err := json.Unmarshal([]byte(payloadInfo.JSON), &payloadJson); err != nil {
			return 0, errors.Wrapf(err, "json.Unmarshal receipt.Payload.JSON %v error", payloadInfo.JSON)
		}
		bs, _ := json.MarshalIndent(payloadJson, "", "\t")
		uc.log.Debugf("PlayStore PayloadJson: %v\n", string(bs))

		if err := json.Unmarshal([]byte(payloadInfo.SkuDetails[0]), &skuDetails); err != nil {
			return 0, errors.Wrapf(err, "json.Unmarshal receipt.Payload.SkuDetails[0] %v error", payloadInfo.SkuDetails[0])
		}
		bs, _ = json.MarshalIndent(skuDetails, "", "\t")
		uc.log.Debugf("PlayStore SkuDetails: %v\n", string(bs))

		if storeKey != payloadJson.ProductID {
			return 0, errors.Errorf("ProductId mismatch: excel=%v vs receipt=%v", storeKey, payloadJson.ProductID)
		}
		if currencyType != skuDetails.PriceCurrencyCode {
			return 0, errors.Errorf("CurrencyType mismatch: currencyType=%v vs skuDetails.PriceCurrencyCode=%v", currencyType, skuDetails.PriceCurrencyCode)
		}
		resp, err := uc.playStore.VerifyProduct(ctx, payloadJson.ProductID, receipt.TransactionID)
		if err != nil {
			return 0, errors.Wrapf(err, "playstore client VerifyProduct error")
		}
		bs, _ = json.MarshalIndent(resp, "", "\t")
		uc.log.Debugf("PlayStore client VerifyProduct resp: %v\n", string(bs))

		// 判断是否为真实订单
		if resp.PurchaseType == nil {
			analysisData = map[string]interface{}{
				"ip":            clientIp,
				"payment":       receipt.Store,
				"order_id":      payloadJson.OrderID,
				"product":       payloadJson.ProductID,
				"currency_type": skuDetails.PriceCurrencyCode,
				"amount":        float64(skuDetails.PriceAmountMicros) / 1000 / 1000,
			}
		}
		orderRecord = &OrderRecord{
			Fid:              u.Fid,
			Yid:              u.Yid,
			IsTest:           resp.PurchaseType != nil,
			StoreKey:         storeKey,
			Channel:          channel.String(),
			Receipt:          string(receiptBytes),
			TransactionID:    payloadJson.OrderID,
			PlayStoreRespStr: string(bs),
		}
	}
	moneyStoreItem, ok := ApolloConf.GetLubanStore().MoneyStore[storeKey]
	if !ok {
		uc.log.Errorf("StoreKey not exist %v in moneyStore table", storeKey)
	}
	if err := uc.repo.UpdateRmbCumulative(ctx, u.Yid, moneyStoreItem.MoneyPrice); err != nil {
		uc.log.Errorf("uc.repo.UpdateRmbCumulative error=%v", err)
	}
	recordId, err := uc.repo.CreateOrderRecord(ctx, orderRecord)
	if err != nil {
		uc.log.Errorf("uc.repo.CreateOrderRecord error=%v", err)
	}

	// 判断是否为fake：避免并发读写 analysisData，在启动 goroutine 前复制所需数据
	if _, ok := analysisData["payment"]; ok {
		trackPrice := moneyStoreItem.MoneyPrice
		// 复制埋点所需字段，供各 goroutine 只读使用，避免 concurrent map read and map write
		trackAmount := analysisData["amount"]
		trackCurrency := analysisData["currency_type"]
		trackProduct := analysisData["product"]
		analysisDataCopy := make(map[string]interface{}, len(analysisData))
		for k, v := range analysisData {
			analysisDataCopy[k] = v
		}

		GoTrack(ctx, time.Now().UnixMilli(), func(bgCtx context.Context, ts int64) {
			// TAP埋点：现实货币（使用副本，不写回共享 map）
			trackData, trackErr := uc.repo.GetTrackCommonPara(bgCtx, u.Yid, u.Ts, analysisDataCopy, u.BasicInfo)
			if trackErr != nil {
				panic(trackErr)
			}
			uc.tapdb.TrackBatchEvent(u.Yid, TAP_EVENT_CHARGE, trackData)
		})

		GoTrack(ctx, time.Now().UnixMilli(), func(bgCtx context.Context, ts int64) {
			afData := make(map[string]interface{})
			afData[AF_ParamRevenue] = trackAmount
			afData[AF_ParamCurrency] = trackCurrency
			afData[AF_ParamContentID] = trackProduct

			if err := uc.appsFlyer.TrackEvent(u.Yid, afid, AF_EventPurchase, afData); err != nil {
				panic(err)
			}
		})

		GoTrack(ctx, time.Now().UnixMilli(), func(bgCtx context.Context, ts int64) {
			rmb, err := uc.repo.GetRmbCumulative(bgCtx, u.Yid)
			if err != nil {
				panic(err)
			}
			if rmb == trackPrice {
				afData := make(map[string]interface{})
				afData[AF_ParamCurrency] = trackCurrency
				afData[AF_ParamContentID] = trackProduct

				if err := uc.appsFlyer.TrackEvent(u.Yid, afid, AF_EventFirstPurchase, afData); err != nil {
					panic(err)
				}
			}
		})
	}

	return recordId, err
}

func (uc *UserUseCase) SendReturnReward(ctx context.Context, u *User) error {
	// uc.RLock()
	// defer uc.RUnlock()

	// username, err := uc.repo.RpcGetUsernameByYid(ctx, u.Yid)
	// if err != nil {
	// 	return err
	// }

	// // 1. 充值补偿
	// orders, err := uc.repo.GetPreviousOrderRecordsByUsername(ctx, username)
	// if err != nil {
	// 	return err
	// }

	// if len(orders) > 0 {
	// 	var (
	// 		rMap        = make(map[int32]int32)
	// 		rTypes      []v1.RewardTypeEnum
	// 		rIds, rNums []int32

	// 		// TODO 配置
	// 		bonusPct int32 = 20
	// 		slotMax  int32 = 999
	// 	)

	// 	doubleMap := make(map[string]int32)
	// 	for _, v := range orders {
	// 		for i, itemId := range StoreKeyPackageDataMap[v.StoreKey].PackageItemID {
	// 			ratio := 2 - doubleMap[v.StoreKey]
	// 			itemNum := StoreKeyPackageDataMap[v.StoreKey].PackageItemCount[i]
	// 			rMap[itemId] += (itemNum*ratio*(100+bonusPct) + 99) / 100
	// 		}
	// 		doubleMap[v.StoreKey] = 1
	// 	}
	// 	for itemId, itemNum := range rMap {
	// 		for ; itemNum > slotMax; itemNum -= slotMax {
	// 			rTypes = append(rTypes, v1.RewardTypeEnum_RewardItemType)
	// 			rIds = append(rIds, itemId)
	// 			rNums = append(rNums, slotMax)
	// 		}

	// 		rTypes = append(rTypes, v1.RewardTypeEnum_RewardItemType)
	// 		rIds = append(rIds, itemId)
	// 		rNums = append(rNums, itemNum)
	// 	}

	// 	// 	// TODO 配置
	// 	// 	if err := uc.AddPersonalMail(ctx, u.Yid, &v1.Mail{
	// 	// 		MailType:   v1.Mailtype_Personal,
	// 	// 		Id:         time.Now().UnixMicro(),
	// 	// 		Title:      "充值补偿",
	// 	// 		Content:    "blabla",
	// 	// 		CanCollect: true,
	// 	// 		RewardType: rTypes,
	// 	// 		RewardId:   rIds,
	// 	// 		RewardNum:  rNums,
	// 	// 		CreateTime: int32(time.Now().Unix()),
	// 	// 		ExpireTime: int32(time.Now().AddDate(0, 0, 1).Unix()),
	// 	// 	}); err != nil {
	// 	// 		return err
	// 	// 	}
	// 	// }

	// 	// // 2. 二测补偿
	// 	// is, err := uc.repo.CheckPreviousUser(ctx, username)
	// 	// if err != nil {
	// 	// 	return err
	// 	// }
	// 	// if is {
	// 	// 	if err := uc.AddPersonalMail(ctx, u.Yid, &v1.Mail{
	// 	// 		MailType:   v1.Mailtype_Personal,
	// 	// 		Id:         time.Now().UnixMicro(),
	// 	// 		Title:      "二测补偿",
	// 	// 		Content:    "blabla",
	// 	// 		CanCollect: true,
	// 	// 		RewardType: []v1.RewardTypeEnum{v1.RewardTypeEnum_RewardItemType},
	// 	// 		RewardId:   []int32{1},
	// 	// 		RewardNum:  []int32{666},
	// 	// 		CreateTime: int32(time.Now().Unix()),
	// 	// 		ExpireTime: int32(time.Now().AddDate(0, 0, 1).Unix()),
	// 	// 	}); err != nil {
	// 	// 		return err
	// 	// 	}
	// }

	return nil
}
