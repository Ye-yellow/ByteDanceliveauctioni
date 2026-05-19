package data

import (
	"context"
	"errors"
	"sort"
	"sync"

	"live-auction-bid/backend/app/auction/service/internal/biz"
)

type MemoryLotRepo struct { mu sync.RWMutex; lots map[string]*biz.Lot }
func NewMemoryLotRepo() *MemoryLotRepo { return &MemoryLotRepo{lots: map[string]*biz.Lot{}} }
func (r *MemoryLotRepo) Create(ctx context.Context, lot *biz.Lot) error { r.mu.Lock(); defer r.mu.Unlock(); r.lots[lot.ID] = clone(lot); return nil }
func (r *MemoryLotRepo) Save(ctx context.Context, lot *biz.Lot) error { r.mu.Lock(); defer r.mu.Unlock(); if _, ok := r.lots[lot.ID]; !ok { return errors.New("拍品不存在") }; r.lots[lot.ID] = clone(lot); return nil }
func (r *MemoryLotRepo) FindByID(ctx context.Context, lotID string) (*biz.Lot, error) { r.mu.RLock(); defer r.mu.RUnlock(); lot, ok := r.lots[lotID]; if !ok { return nil, errors.New("拍品不存在") }; return clone(lot), nil }
func (r *MemoryLotRepo) List(ctx context.Context, roomID string, status biz.LotStatus) ([]*biz.Lot, error) { r.mu.RLock(); defer r.mu.RUnlock(); out := []*biz.Lot{}; for _, lot := range r.lots { if roomID != "" && lot.RoomID != roomID { continue }; if status != "" && lot.Status != status { continue }; out = append(out, clone(lot)) }; sort.Slice(out, func(i,j int) bool { return out[i].ID < out[j].ID }); return out, nil }
func clone(l *biz.Lot) *biz.Lot { if l == nil { return nil }; cp := *l; cp.TrustCards = append([]biz.TrustRevealCard(nil), l.TrustCards...); return &cp }
