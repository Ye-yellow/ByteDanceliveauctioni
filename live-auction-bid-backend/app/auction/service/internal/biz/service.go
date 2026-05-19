package biz

import (
	"context"
	"fmt"
	"time"
)

type DomainService struct {
	repo      LotRepository
	pub       EventPublisher
	ai        AtmosphereAI
	extendDur time.Duration
}

func NewDomainService(repo LotRepository, pub EventPublisher, ai AtmosphereAI, extendDur time.Duration) *DomainService {
	return &DomainService{repo: repo, pub: pub, ai: ai, extendDur: extendDur}
}

func (s *DomainService) PlaceBid(ctx context.Context, lotID, userID, nickname string, amount Money) (*Lot, error) {
	lot, err := s.repo.FindByID(ctx, lotID)
	if err != nil {
		return nil, err
	}
	bid := Bid{ID: fmt.Sprintf("bid_%d", time.Now().UnixNano()), LotID: lotID, UserID: userID, Nickname: nickname, Amount: amount, CreatedAt: time.Now()}
	if err := lot.PlaceBid(bid, s.extendDur); err != nil {
		return nil, err
	}
	if s.ai != nil {
		lot.AtmosphereText = s.ai.OnBid(ctx, lot, bid)
	}
	if err := s.repo.Save(ctx, lot); err != nil {
		return nil, err
	}
	_ = s.pub.PublishBidAccepted(ctx, lot, bid)
	_ = s.pub.PublishLotUpdated(ctx, lot)
	return lot, nil
}

func (s *DomainService) Settle(ctx context.Context, lotID string) (*Lot, error) {
	lot, err := s.repo.FindByID(ctx, lotID)
	if err != nil {
		return nil, err
	}
	lot.Status = LotSettled
	lot.Version++
	if err := s.repo.Save(ctx, lot); err != nil {
		return nil, err
	}
	_ = s.pub.PublishLotSettled(ctx, lot)
	return lot, nil
}
