package biz

import "time"

type Settlement struct {
	LotID        string    `json:"lotId"`
	WinnerUserID string    `json:"winnerUserId"`
	FinalPrice   Money     `json:"finalPrice"`
	SettledAt    time.Time `json:"settledAt"`
}
