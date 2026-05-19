package data

import (
	"context"

	"live-auction-bid/backend/app/auction/service/internal/model"
)

func (s *MemoryStore) Append(ctx context.Context, bid model.Bid) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.bidsByLot[bid.LotID] = append(s.bidsByLot[bid.LotID], bid)
	return nil
}

func (s *MemoryStore) ListByLot(ctx context.Context, lotID string) ([]model.Bid, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	bids := s.bidsByLot[lotID]
	return append([]model.Bid(nil), bids...), nil
}

func (s *MemoryStore) FindByIdempotencyKey(ctx context.Context, lotID, key string) (model.Bid, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.idemByLot[lotID] == nil {
		return model.Bid{}, false, nil
	}
	bid, ok := s.idemByLot[lotID][key]
	return bid, ok, nil
}

func (s *MemoryStore) SaveIdempotencyKey(ctx context.Context, lotID, key string, bid model.Bid) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.idemByLot[lotID] == nil {
		s.idemByLot[lotID] = make(map[string]model.Bid)
	}
	s.idemByLot[lotID][key] = bid
	return nil
}
