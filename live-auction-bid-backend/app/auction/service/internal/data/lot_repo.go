package data

import (
	"context"
	"errors"
	"sort"

	v1 "live-auction-bid/backend/api/auction/service/v1"
)

func (s *MemoryStore) Create(ctx context.Context, lot *v1.Lot) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.lots[lot.Id] = cloneLot(lot)
	return nil
}

func (s *MemoryStore) Save(ctx context.Context, lot *v1.Lot) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.lots[lot.Id]; !ok {
		return errors.New("拍品不存在")
	}
	s.lots[lot.Id] = cloneLot(lot)
	return nil
}

func (s *MemoryStore) FindByID(ctx context.Context, lotID string) (*v1.Lot, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	lot, ok := s.lots[lotID]
	if !ok {
		return nil, errors.New("拍品不存在")
	}
	return cloneLot(lot), nil
}

func (s *MemoryStore) List(ctx context.Context, roomID string, status v1.LotStatus) ([]*v1.Lot, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	lots := make([]*v1.Lot, 0, len(s.lots))
	for _, lot := range s.lots {
		if roomID != "" && lot.RoomId != roomID {
			continue
		}
		if status != 0 && lot.Status != status {
			continue
		}
		lots = append(lots, cloneLot(lot))
	}

	sort.Slice(lots, func(i, j int) bool {
		return lots[i].Id < lots[j].Id
	})
	return lots, nil
}
