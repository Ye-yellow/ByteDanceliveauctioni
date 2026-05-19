package biz

import "time"

type BidRule struct {
	StartPrice             Money         `json:"startPrice"`
	MinIncrement           Money         `json:"minIncrement"`
	ReservePrice           Money         `json:"reservePrice,omitempty"`
	AntiSnipeExtend        time.Duration `json:"antiSnipeExtend"`
	MaxBidPerUserPerSecond int           `json:"maxBidPerUserPerSecond,omitempty"`
}
