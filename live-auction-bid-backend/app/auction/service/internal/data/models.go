package data

import "time"

type AuctionLotModel struct {
	ID                     string `gorm:"column:id;type:varchar(64);primaryKey"`
	RoomID                 string `gorm:"column:room_id;type:varchar(64);not null;index:idx_room_status,priority:1;index:idx_room_updated,priority:1"`
	Title                  string `gorm:"column:title;type:varchar(255);not null"`
	Description            string `gorm:"column:description;type:text;not null"`
	ImageURL               string `gorm:"column:image_url;type:varchar(1024);not null"`
	Status                 int32  `gorm:"column:status;type:int;not null;index:idx_room_status,priority:2;index:idx_status_ends_at,priority:1"`
	QueueStatus            int32  `gorm:"column:queue_status;type:int;not null;default:1;index:idx_room_queue,priority:2"`
	QueuePosition          int32  `gorm:"column:queue_position;type:int;not null;default:0;index:idx_room_queue,priority:3"`
	StartPriceAmount       int64  `gorm:"column:start_price_amount;not null"`
	StartPriceCurrency     string `gorm:"column:start_price_currency;type:varchar(16);not null"`
	MinIncrementAmount     int64  `gorm:"column:min_increment_amount;not null"`
	MinIncrementCurrency   string `gorm:"column:min_increment_currency;type:varchar(16);not null"`
	CapPriceAmount         *int64 `gorm:"column:cap_price_amount"`
	CapPriceCurrency       string `gorm:"column:cap_price_currency;type:varchar(16)"`
	DurationSeconds        int32  `gorm:"column:duration_seconds;type:int;not null"`
	AntiSnipeWindowSeconds int32  `gorm:"column:anti_snipe_window_seconds;type:int;not null"`
	AntiSnipeExtendSeconds int32  `gorm:"column:anti_snipe_extend_seconds;type:int;not null"`
	MaxExtendCount         int32  `gorm:"column:max_extend_count;type:int;not null"`
	CurrentPriceAmount     int64  `gorm:"column:current_price_amount;not null"`
	CurrentPriceCurrency   string `gorm:"column:current_price_currency;type:varchar(16);not null"`
	LeadingUserID          string `gorm:"column:leading_user_id;type:varchar(64);not null;default:''"`
	LeadingNickname        string `gorm:"column:leading_nickname;type:varchar(128);not null;default:''"`
	StartedAtUnixMs        int64  `gorm:"column:started_at_unix_ms;not null;default:0"`
	EndsAtUnixMs           int64  `gorm:"column:ends_at_unix_ms;not null;default:0;index:idx_status_ends_at,priority:2"`
	SettledAtUnixMs        int64  `gorm:"column:settled_at_unix_ms;not null;default:0"`
	CancelReason           string `gorm:"column:cancel_reason;type:varchar(512);not null;default:''"`
	CancelledAtUnixMs      int64  `gorm:"column:cancelled_at_unix_ms;not null;default:0"`
	WinnerUserID           string `gorm:"column:winner_user_id;type:varchar(64);not null;default:''"`
	WinnerNickname         string `gorm:"column:winner_nickname;type:varchar(128);not null;default:''"`
	FinalPriceAmount       int64  `gorm:"column:final_price_amount;not null;default:0"`
	FinalPriceCurrency     string `gorm:"column:final_price_currency;type:varchar(16);not null;default:''"`
	Version                int64  `gorm:"column:version;not null"`
	PlaybookStage          int32  `gorm:"column:playbook_stage;type:int;not null"`
	Payload                string `gorm:"column:payload;type:json;not null"`
	CreatedAt              time.Time
	UpdatedAt              time.Time `gorm:"index:idx_room_updated,priority:2"`
}

func (AuctionLotModel) TableName() string { return "auction_lots" }

type AuctionBidModel struct {
	ID              string  `gorm:"column:id;type:varchar(64);primaryKey"`
	LotID           string  `gorm:"column:lot_id;type:varchar(64);not null;index:idx_lot_created,priority:1;index:idx_lot_amount,priority:1;index:idx_lot_user,priority:1;uniqueIndex:idx_lot_idem,priority:1"`
	UserID          string  `gorm:"column:user_id;type:varchar(64);not null;index:idx_lot_user,priority:2"`
	Nickname        string  `gorm:"column:nickname;type:varchar(128);not null"`
	Amount          int64   `gorm:"column:amount;not null;index:idx_lot_amount,priority:2"`
	Currency        string  `gorm:"column:currency;type:varchar(16);not null"`
	IdempotencyKey  *string `gorm:"column:idempotency_key;type:varchar(128);uniqueIndex:idx_lot_idem,priority:2"`
	CreatedAtUnixMs int64   `gorm:"column:created_at_unix_ms;not null;index:idx_lot_created,priority:2"`
	Payload         string  `gorm:"column:payload;type:json;not null"`
	CreatedAt       time.Time
}

func (AuctionBidModel) TableName() string { return "auction_bids" }

type AuctionEventModel struct {
	ID               string `gorm:"column:id;type:varchar(64);primaryKey"`
	RoomID           string `gorm:"column:room_id;type:varchar(64);not null;index:idx_room_occurred,priority:1"`
	LotID            string `gorm:"column:lot_id;type:varchar(64);not null;default:'';index:idx_lot_occurred,priority:1"`
	Type             int32  `gorm:"column:type;type:int;not null;index:idx_type_occurred,priority:1"`
	OccurredAtUnixMs int64  `gorm:"column:occurred_at_unix_ms;not null;index:idx_room_occurred,priority:2;index:idx_lot_occurred,priority:2;index:idx_type_occurred,priority:2"`
	Reason           string `gorm:"column:reason;type:varchar(512);not null;default:''"`
	Payload          string `gorm:"column:payload;type:json;not null"`
	StreamID         string `gorm:"column:stream_id;type:varchar(64);not null;default:''"`
	StreamedAtUnixMs int64  `gorm:"column:streamed_at_unix_ms;not null;default:0;index:idx_streamed_at"`
	LastStreamError  string `gorm:"column:last_stream_error;type:varchar(512);not null;default:''"`
	CreatedAt        time.Time
}

func (AuctionEventModel) TableName() string { return "auction_events" }

type AuctionUserModel struct {
	ID              string `gorm:"column:id;type:varchar(64);primaryKey"`
	Username        string `gorm:"column:username;type:varchar(64);not null;uniqueIndex:idx_username"`
	Nickname        string `gorm:"column:nickname;type:varchar(128);not null"`
	PasswordHash    string `gorm:"column:password_hash;type:varchar(255);not null"`
	Role            int32  `gorm:"column:role;type:int;not null;index:idx_role"`
	CreatedAtUnixMs int64  `gorm:"column:created_at_unix_ms;not null"`
	UpdatedAtUnixMs int64  `gorm:"column:updated_at_unix_ms;not null"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (AuctionUserModel) TableName() string { return "auction_users" }

type AuctionUserSessionModel struct {
	ID                     string `gorm:"column:id;type:varchar(64);primaryKey"`
	UserID                 string `gorm:"column:user_id;type:varchar(64);not null;index:idx_user_sessions"`
	RefreshTokenHash       string `gorm:"column:refresh_token_hash;type:varchar(64);not null;uniqueIndex:idx_refresh_token_hash"`
	RefreshExpiresAtUnixMs int64  `gorm:"column:refresh_expires_at_unix_ms;not null;index:idx_session_expiry"`
	RevokedAtUnixMs        int64  `gorm:"column:revoked_at_unix_ms;not null;default:0"`
	CreatedAtUnixMs        int64  `gorm:"column:created_at_unix_ms;not null"`
	CreatedAt              time.Time
	UpdatedAt              time.Time
}

func (AuctionUserSessionModel) TableName() string { return "auction_user_sessions" }

type AssetFileModel struct {
	ID               string `gorm:"column:id;type:varchar(64);primaryKey"`
	OwnerUserID      string `gorm:"column:owner_user_id;type:varchar(64);not null;index:idx_asset_owner"`
	RoomID           string `gorm:"column:room_id;type:varchar(64);not null;default:'';index:idx_asset_room"`
	BizType          string `gorm:"column:biz_type;type:varchar(64);not null;index:idx_asset_biz_type"`
	Status           string `gorm:"column:status;type:varchar(32);not null;default:'temporary';index:idx_asset_status"`
	AttachedLotID    string `gorm:"column:attached_lot_id;type:varchar(64);not null;default:'';index:idx_asset_attached_lot"`
	StorageProvider  string `gorm:"column:storage_provider;type:varchar(32);not null"`
	Bucket           string `gorm:"column:bucket;type:varchar(128);not null"`
	ObjectKey        string `gorm:"column:object_key;type:varchar(512);not null;uniqueIndex:idx_asset_object_key"`
	PublicURL        string `gorm:"column:public_url;type:varchar(1024);not null"`
	OriginalName     string `gorm:"column:original_name;type:varchar(255);not null;default:''"`
	MimeType         string `gorm:"column:mime_type;type:varchar(64);not null"`
	SizeBytes        int64  `gorm:"column:size_bytes;not null"`
	SHA256           string `gorm:"column:sha256;type:char(64);not null;index:idx_asset_sha256"`
	AttachedAtUnixMs int64  `gorm:"column:attached_at_unix_ms;not null;default:0"`
	DeletedAtUnixMs  int64  `gorm:"column:deleted_at_unix_ms;not null;default:0;index:idx_asset_deleted_at"`
	ExpiresAtUnixMs  int64  `gorm:"column:expires_at_unix_ms;not null;default:0;index:idx_asset_expiry"`
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func (AssetFileModel) TableName() string { return "asset_files" }
