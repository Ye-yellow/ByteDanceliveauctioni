package conf

type Bootstrap struct {
	Server  ServerConfig  `yaml:"server"`
	Auction AuctionConfig `yaml:"auction"`
}

type ServerConfig struct {
	HTTP HTTPConfig `yaml:"http"`
}

type HTTPConfig struct {
	Addr string `yaml:"addr"`
}

type AuctionConfig struct {
	DefaultRoomID          string `yaml:"default_room_id"`
	AntiSnipeExtendSeconds int    `yaml:"anti_snipe_extend_seconds"`
	MaxClockSkewMS         int    `yaml:"max_clock_skew_ms"`
}
