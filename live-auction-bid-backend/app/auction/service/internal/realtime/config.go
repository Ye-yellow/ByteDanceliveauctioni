package realtime

import (
	"errors"
	"net"
	"net/url"
	"strings"
	"time"
)

const (
	ScopePublic = "public"
	ScopeAdmin  = "admin"

	defaultTicketTTL = 60 * time.Second
)

type Config struct {
	Environment        string
	AllowedOrigins     []string
	AllowMissingOrigin bool
	TicketTTL          time.Duration
	TicketSecret       string
}

func DefaultConfig() Config {
	return Config{
		Environment:        "dev",
		AllowMissingOrigin: true,
		TicketTTL:          defaultTicketTTL,
	}
}

func ConfigFromEnv(getenv func(string) string) (Config, error) {
	cfg := DefaultConfig()
	cfg.Environment = strings.TrimSpace(getenv("AUCTION_ENV"))
	cfg.AllowedOrigins = splitCSV(getenv("AUCTION_WS_ALLOWED_ORIGINS"))
	cfg.TicketSecret = strings.TrimSpace(getenv("AUCTION_JWT_SECRET"))
	if value := strings.TrimSpace(getenv("AUCTION_WS_TICKET_TTL")); value != "" {
		duration, err := time.ParseDuration(value)
		if err != nil {
			return Config{}, err
		}
		cfg.TicketTTL = duration
	}
	if value := strings.TrimSpace(getenv("AUCTION_WS_ALLOW_MISSING_ORIGIN")); value != "" {
		cfg.AllowMissingOrigin = parseBool(value)
	} else {
		cfg.AllowMissingOrigin = !isProdEnv(cfg.Environment)
	}
	return NormalizeConfig(cfg)
}

func NormalizeConfig(cfg Config) (Config, error) {
	if strings.TrimSpace(cfg.Environment) == "" {
		cfg.Environment = "dev"
	}
	cfg.Environment = strings.ToLower(strings.TrimSpace(cfg.Environment))
	if cfg.TicketTTL <= 0 {
		cfg.TicketTTL = defaultTicketTTL
	}
	origins := make([]string, 0, len(cfg.AllowedOrigins))
	for _, origin := range cfg.AllowedOrigins {
		normalized, ok := normalizeOrigin(origin)
		if ok {
			origins = append(origins, normalized)
		}
	}
	cfg.AllowedOrigins = origins
	if isProdEnv(cfg.Environment) && len(cfg.AllowedOrigins) == 0 {
		return Config{}, errors.New("AUCTION_WS_ALLOWED_ORIGINS is required in prod")
	}
	if isProdEnv(cfg.Environment) && strings.TrimSpace(cfg.TicketSecret) == "" {
		return Config{}, errors.New("AUCTION_JWT_SECRET is required for websocket tickets in prod")
	}
	return cfg, nil
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func parseBool(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func isProdEnv(env string) bool {
	switch strings.ToLower(strings.TrimSpace(env)) {
	case "prod", "production":
		return true
	default:
		return false
	}
}

func normalizeOrigin(origin string) (string, bool) {
	parsed, err := url.Parse(strings.TrimSpace(origin))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", false
	}
	return strings.ToLower(parsed.Scheme) + "://" + strings.ToLower(parsed.Host), true
}

func isLocalhostOrigin(origin string) bool {
	parsed, err := url.Parse(origin)
	if err != nil {
		return false
	}
	host := parsed.Hostname()
	if host == "" {
		host = parsed.Host
		if splitHost, _, err := net.SplitHostPort(parsed.Host); err == nil {
			host = splitHost
		}
	}
	switch strings.ToLower(strings.Trim(host, "[]")) {
	case "localhost", "127.0.0.1", "::1":
		return true
	default:
		return false
	}
}

func normalizeScope(scope string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(scope)) {
	case "", ScopePublic:
		return ScopePublic, true
	case ScopeAdmin:
		return ScopeAdmin, true
	default:
		return "", false
	}
}
