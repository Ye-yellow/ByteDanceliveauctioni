package data

import "time"

func normalizeLeaseTTL(ttl time.Duration) time.Duration {
	if ttl <= 0 {
		return 15 * time.Second
	}
	return ttl
}

func normalizeRenewInterval(interval time.Duration) time.Duration {
	if interval <= 0 {
		return 5 * time.Second
	}
	return interval
}

func minWorkerInterval(a, b time.Duration) time.Duration {
	if a <= 0 {
		return normalizeRenewInterval(b)
	}
	if b <= 0 || a < b {
		return a
	}
	return b
}

func formatWorkerTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Format(time.RFC3339)
}
