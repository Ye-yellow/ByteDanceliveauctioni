package apperr

import "errors"

// ErrLotVersionConflict is the stable sentinel for optimistic-lock conflicts on a lot aggregate.
// Repository implementations should wrap or return this value so service adapters can expose a
// consistent user-facing retry semantic instead of leaking storage-specific errors.
var ErrLotVersionConflict = errors.New("lot version conflict")

var (
	ErrInvalidArgument    = errors.New("invalid argument")
	ErrUnauthenticated    = errors.New("unauthenticated")
	ErrPermissionDenied   = errors.New("permission denied")
	ErrUsernameTaken      = errors.New("username already exists")
	ErrInvalidCredentials = errors.New("invalid username or password")
	ErrInvalidToken       = errors.New("invalid token")
	ErrTokenExpired       = errors.New("token expired")
	ErrSessionExpired     = errors.New("session expired")
	ErrAccountDisabled    = errors.New("account disabled")
	ErrUserNotFound       = errors.New("user not found")
	ErrNotFound           = errors.New("not found")
)

func IsLotVersionConflict(err error) bool {
	return errors.Is(err, ErrLotVersionConflict)
}

func IsInvalidArgument(err error) bool {
	return errors.Is(err, ErrInvalidArgument)
}

func IsUnauthenticated(err error) bool {
	return errors.Is(err, ErrUnauthenticated)
}

func IsPermissionDenied(err error) bool {
	return errors.Is(err, ErrPermissionDenied)
}

func IsUsernameTaken(err error) bool {
	return errors.Is(err, ErrUsernameTaken)
}

func IsInvalidCredentials(err error) bool {
	return errors.Is(err, ErrInvalidCredentials)
}

func IsInvalidToken(err error) bool {
	return errors.Is(err, ErrInvalidToken)
}

func IsTokenExpired(err error) bool {
	return errors.Is(err, ErrTokenExpired)
}

func IsSessionExpired(err error) bool {
	return errors.Is(err, ErrSessionExpired)
}

func IsAccountDisabled(err error) bool {
	return errors.Is(err, ErrAccountDisabled)
}

func IsUserNotFound(err error) bool {
	return errors.Is(err, ErrUserNotFound)
}

func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound) || errors.Is(err, ErrUserNotFound)
}
