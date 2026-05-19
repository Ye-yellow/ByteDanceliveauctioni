package data

import (
	"sync"

	v1 "live-auction-bid/backend/api/auction/service/v1"
)

// MemoryStore 是 V1 内存存储实现。
//
// 该实现只服务本地演示；接口已经按仓储边界拆好，后续可分别替换为：
// - LotRepository -> MySQL
// - BidRepository -> Redis Stream / MySQL
// - 幂等键 -> Redis SETNX
type MemoryStore struct {
	mu sync.RWMutex

	lots      map[string]*v1.Lot
	bidsByLot map[string][]v1.Bid
	idemByLot map[string]map[string]v1.Bid
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		lots:      make(map[string]*v1.Lot),
		bidsByLot: make(map[string][]v1.Bid),
		idemByLot: make(map[string]map[string]v1.Bid),
	}
}
