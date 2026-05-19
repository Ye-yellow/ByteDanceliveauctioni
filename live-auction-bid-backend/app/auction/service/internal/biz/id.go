package biz

import (
	"fmt"
	"time"
)

func NewID(prefix string) string {
	return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
}

func NowMs() int64 {
	return time.Now().UnixMilli()
}
