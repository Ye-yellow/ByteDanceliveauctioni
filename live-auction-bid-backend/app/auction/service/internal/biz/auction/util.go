package auction

import (
	"google.golang.org/protobuf/proto"
	v1 "live-auction-bid/backend/api/auction/service/v1"
)

func CNY(amount int64) *v1.Money {
	return &v1.Money{Amount: amount, Currency: "CNY"}
}

func CloneLot(lot *v1.Lot) *v1.Lot {
	if lot == nil {
		return nil
	}
	return proto.Clone(lot).(*v1.Lot)
}
