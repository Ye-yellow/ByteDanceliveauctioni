package data

import (
	"context"

	v1 "live-auction-bid/backend/api/auction/service/v1"
)

func (s *MemoryStore) Append(ctx context.Context, bid v1.Bid) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.bidsByLot[bid.LotId] = append(s.bidsByLot[bid.LotId], bid)
	return nil
}

func (s *MemoryStore) ListByLot(ctx context.Context, lotID string) ([]v1.Bid, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	bids := s.bidsByLot[lotID]
	return append([]v1.Bid(nil), bids...), nil
}

func (s *MemoryStore) FindByIdempotencyKey(ctx context.Context, lotID, key string) (v1.Bid, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.idemByLot[lotID] == nil {
		return v1.Bid{}, false, nil
	}
	bid, ok := s.idemByLot[lotID][key]
	return bid, ok, nil
}

func (s *MemoryStore) SaveIdempotencyKey(ctx context.Context, lotID, key string, bid v1.Bid) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.idemByLot[lotID] == nil {
		s.idemByLot[lotID] = make(map[string]v1.Bid)
	}
	s.idemByLot[lotID][key] = bid
	return nil
}
