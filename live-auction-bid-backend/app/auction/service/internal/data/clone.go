package data

import (
	"google.golang.org/protobuf/proto"
	v1 "live-auction-bid/backend/api/auction/service/v1"
)

func cloneLot(lot *v1.Lot) *v1.Lot {
	if lot == nil {
		return nil
	}
	return proto.Clone(lot).(*v1.Lot)
}
