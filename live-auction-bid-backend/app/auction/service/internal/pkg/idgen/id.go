package idgen

import (
	"fmt"
	"time"
)

func New(prefix string) string {
	return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
}
