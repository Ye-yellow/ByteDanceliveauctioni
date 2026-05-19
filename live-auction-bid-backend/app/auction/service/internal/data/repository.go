package data

import (
	"context"
	"errors"
	"sync"

	biz "live-auction-bid/backend/app/auction/service/internal/biz"
)

type LotRepository struct {
	mu   sync.RWMutex
	lots map[string]*biz.Lot
}

func NewLotRepository() *LotRepository { return &LotRepository{lots: map[string]*biz.Lot{}} }

func (r *LotRepository) Save(ctx context.Context, lot *biz.Lot) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *lot
	cp.Bids = append([]biz.Bid(nil), lot.Bids...)
	r.lots[lot.ID] = &cp
	return nil
}

func (r *LotRepository) FindByID(ctx context.Context, id string) (*biz.Lot, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	lot, ok := r.lots[id]
	if !ok {
		return nil, errors.New("lot not found")
	}
	cp := *lot
	cp.Bids = append([]biz.Bid(nil), lot.Bids...)
	return &cp, nil
}

func (r *LotRepository) FindLiveByRoom(ctx context.Context, roomID string) (*biz.Lot, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, lot := range r.lots {
		if lot.RoomID == roomID && lot.Status == biz.LotLive {
			cp := *lot
			cp.Bids = append([]biz.Bid(nil), lot.Bids...)
			return &cp, nil
		}
	}
	return nil, errors.New("live lot not found")
}

func (r *LotRepository) List(ctx context.Context) ([]*biz.Lot, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*biz.Lot, 0, len(r.lots))
	for _, lot := range r.lots {
		cp := *lot
		cp.Bids = append([]biz.Bid(nil), lot.Bids...)
		out = append(out, &cp)
	}
	return out, nil
}
