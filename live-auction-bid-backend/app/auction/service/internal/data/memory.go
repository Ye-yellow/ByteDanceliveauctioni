package data

import (
	"context"
	"errors"
	"sort"
	"sync"

	"live-auction-bid/backend/app/auction/service/internal/model"
)

// MemoryStore 是 V1 内存存储实现。
//
// 该实现只服务本地演示；接口已经按仓储边界拆好，后续可分别替换为：
// - LotRepository -> MySQL
// - BidRepository -> Redis Stream / MySQL
// - 幂等键 -> Redis SETNX
type MemoryStore struct {
	mu sync.RWMutex

	lots      map[string]*model.Lot
	bidsByLot map[string][]model.Bid
	idemByLot map[string]map[string]model.Bid
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		lots:      make(map[string]*model.Lot),
		bidsByLot: make(map[string][]model.Bid),
		idemByLot: make(map[string]map[string]model.Bid),
	}
}

func (s *MemoryStore) Create(ctx context.Context, lot *model.Lot) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.lots[lot.ID] = model.CloneLot(lot)
	return nil
}

func (s *MemoryStore) Save(ctx context.Context, lot *model.Lot) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.lots[lot.ID]; !ok {
		return errors.New("拍品不存在")
	}
	s.lots[lot.ID] = model.CloneLot(lot)
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
		if roomID != "" && lot.RoomID != roomID {
			continue
		}
		if status != "" && lot.Status != status {
			continue
		}
		lots = append(lots, model.CloneLot(lot))
	}

	sort.Slice(lots, func(i, j int) bool {
		return lots[i].ID < lots[j].ID
	})
	return lots, nil
}

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
