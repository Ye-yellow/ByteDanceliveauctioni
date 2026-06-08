package data

import (
	"testing"

	v1 "live-auction-bid/backend/api/auction/service/v1"
	"live-auction-bid/backend/app/auction/service/internal/biz/auction"
)

func TestCanFastForwardRuntimeProjection(t *testing.T) {
	tests := []struct {
		name       string
		current    AuctionLotModel
		offset     AuctionRuntimeProjectionOffsetModel
		projection auction.RuntimeProjectionEvent
		want       bool
	}{
		{
			name:    "already projected offset",
			current: AuctionLotModel{Version: 12, Status: int32(v1.LotStatus_LOT_STATUS_LIVE)},
			offset:  AuctionRuntimeProjectionOffsetModel{LastProjectedVersion: 12},
			projection: auction.RuntimeProjectionEvent{
				PreviousLotVersion: 3,
				LotVersion:         4,
			},
			want: true,
		},
		{
			name:    "terminal lot advanced to event version",
			current: AuctionLotModel{Version: 181, Status: int32(v1.LotStatus_LOT_STATUS_CANCELLED)},
			offset:  AuctionRuntimeProjectionOffsetModel{LastProjectedVersion: 180},
			projection: auction.RuntimeProjectionEvent{
				PreviousLotVersion: 180,
				LotVersion:         181,
			},
			want: true,
		},
		{
			name:    "real gap is not fast forwarded",
			current: AuctionLotModel{Version: 179, Status: int32(v1.LotStatus_LOT_STATUS_LIVE)},
			offset:  AuctionRuntimeProjectionOffsetModel{LastProjectedVersion: 179},
			projection: auction.RuntimeProjectionEvent{
				PreviousLotVersion: 180,
				LotVersion:         181,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := canFastForwardRuntimeProjection(tt.current, tt.offset, tt.projection); got != tt.want {
				t.Fatalf("canFastForwardRuntimeProjection() = %v, want %v", got, tt.want)
			}
		})
	}
}
