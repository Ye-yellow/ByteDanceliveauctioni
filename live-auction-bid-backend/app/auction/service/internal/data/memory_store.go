package data

import (
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
