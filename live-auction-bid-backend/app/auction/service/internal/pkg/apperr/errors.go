package apperr

import "errors"

// ErrLotVersionConflict is the stable sentinel for optimistic-lock conflicts on a lot aggregate.
// Repository implementations should wrap or return this value so service adapters can expose a
// consistent user-facing retry semantic instead of leaking storage-specific errors.
var ErrLotVersionConflict = errors.New("lot version conflict")

func IsLotVersionConflict(err error) bool {
	return errors.Is(err, ErrLotVersionConflict)
}
