package auction

import (
	"testing"
	"time"

	v1 "live-auction-bid/backend/api/auction/service/v1"
)

func TestIsStalePreStartLotUsesBusinessDayAndKeepsDrafts(t *testing.T) {
	loc := time.FixedZone("Asia/Shanghai", 8*60*60)
	nowMs := time.Date(2026, 6, 3, 0, 10, 0, 0, loc).UnixMilli()

	cases := []struct {
		name string
		lot  *v1.Lot
		want bool
	}{
		{
			name: "queued previous day under 24 hours",
			lot: &v1.Lot{
				Status:          v1.LotStatus_LOT_STATUS_QUEUED,
				CreatedAtUnixMs: time.Date(2026, 6, 2, 23, 50, 0, 0, loc).UnixMilli(),
			},
			want: true,
		},
		{
			name: "queued same day",
			lot: &v1.Lot{
				Status:          v1.LotStatus_LOT_STATUS_QUEUED,
				CreatedAtUnixMs: time.Date(2026, 6, 3, 0, 5, 0, 0, loc).UnixMilli(),
			},
			want: false,
		},
		{
			name: "ready older than 24 hours",
			lot: &v1.Lot{
				Status:          v1.LotStatus_LOT_STATUS_READY,
				CreatedAtUnixMs: nowMs - int64(25*time.Hour/time.Millisecond),
			},
			want: true,
		},
		{
			name: "draft previous day",
			lot: &v1.Lot{
				Status:          v1.LotStatus_LOT_STATUS_DRAFT,
				CreatedAtUnixMs: time.Date(2026, 6, 2, 23, 50, 0, 0, loc).UnixMilli(),
			},
			want: false,
		},
		{
			name: "already started",
			lot: &v1.Lot{
				Status:          v1.LotStatus_LOT_STATUS_QUEUED,
				CreatedAtUnixMs: time.Date(2026, 6, 2, 23, 50, 0, 0, loc).UnixMilli(),
				StartedAtUnixMs: nowMs - 1000,
			},
			want: false,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsStalePreStartLot(tt.lot, nowMs); got != tt.want {
				t.Fatalf("IsStalePreStartLot() = %v, want %v", got, tt.want)
			}
		})
	}
}
