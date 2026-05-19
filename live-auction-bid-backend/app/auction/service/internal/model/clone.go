package model

import "google.golang.org/protobuf/proto"

func CloneLot(lot *Lot) *Lot {
	if lot == nil {
		return nil
	}
	return proto.Clone(lot).(*Lot)
}
