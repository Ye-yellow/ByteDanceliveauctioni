package data

import (
	"context"
	"errors"
	"sort"

	"live-auction-bid/backend/app/auction/service/internal/model"
)

func (s *MemoryStore) Create(ctx context.Context, lot *model.Lot) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.lots[lot.Id] = model.CloneLot(lot)
	return nil
}

func (s *MemoryStore) Save(ctx context.Context, lot *model.Lot) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.lots[lot.Id]; !ok {
		return errors.New("拍品不存在")
	}
	s.lots[lot.Id] = model.CloneLot(lot)
	return nil
}

func (s *MemoryStore) FindByID(ctx context.Context, lotID string) (*model.Lot, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	lot, ok := s.lots[lotID]
	if !ok {
		return nil, errors.New("拍品不存在")
	}
	return model.CloneLot(lot), nil
}

func (s *MemoryStore) List(ctx context.Context, roomID string, status model.LotStatus) ([]*model.Lot, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	lots := make([]*model.Lot, 0, len(s.lots))
	for _, lot := range s.lots {
		if roomID != "" && lot.RoomId != roomID {
			continue
		}
		if status != 0 && lot.Status != status {
			continue
		}
		lots = append(lots, model.CloneLot(lot))
	}

	sort.Slice(lots, func(i, j int) bool {
		return lots[i].Id < lots[j].Id
	})
	return lots, nil
}
