package memory

import (
	"context"
	"errors"
	"sync"

	domain "live-auction-bid/backend/internal/domain/auction"
)

type LotRepository struct {
	mu   sync.RWMutex
	lots map[string]*domain.Lot
}

func NewLotRepository() *LotRepository { return &LotRepository{lots: map[string]*domain.Lot{}} }

func (r *LotRepository) Save(ctx context.Context, lot *domain.Lot) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *lot
	cp.Bids = append([]domain.Bid(nil), lot.Bids...)
	r.lots[lot.ID] = &cp
	return nil
}

func (r *LotRepository) FindByID(ctx context.Context, id string) (*domain.Lot, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	lot, ok := r.lots[id]
	if !ok {
		return nil, errors.New("lot not found")
	}
	cp := *lot
	cp.Bids = append([]domain.Bid(nil), lot.Bids...)
	return &cp, nil
}

func (r *LotRepository) FindLiveByRoom(ctx context.Context, roomID string) (*domain.Lot, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, lot := range r.lots {
		if lot.RoomID == roomID && lot.Status == domain.LotLive {
			cp := *lot
			cp.Bids = append([]domain.Bid(nil), lot.Bids...)
			return &cp, nil
		}
	}
	return nil, errors.New("live lot not found")
}

func (r *LotRepository) List(ctx context.Context) ([]*domain.Lot, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*domain.Lot, 0, len(r.lots))
	for _, lot := range r.lots {
		cp := *lot
		cp.Bids = append([]domain.Bid(nil), lot.Bids...)
		out = append(out, &cp)
	}
	return out, nil
}
