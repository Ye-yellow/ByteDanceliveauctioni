package model

func CloneLot(lot *Lot) *Lot {
	if lot == nil {
		return nil
	}
	cp := *lot
	cp.TrustCards = append([]TrustRevealCard(nil), lot.TrustCards...)
	return &cp
}
