package config

import (
	"errors"
	"fmt"
	"github.com/v3danth/free-chat/internal/avatar"
	"log"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	ServerPort string
	// AvatarStyle draws society marks as "poly" (SVG) or "pixel" (PNG).
	AvatarStyle string
	// BehindProxy trusts X-Forwarded-For; enable only behind your own proxy.
	BehindProxy bool

	MySQLHost     string
	MySQLPort     string
	MySQLUser     string
	MySQLPassword string
	MySQLDatabase string

	JWTSecret string

	// Optional: when both are set, this member is created or promoted to
	// admin at startup.
	AdminEmail    string
	AdminPassword string
	AdminName     string

	// GeoIPPath is a MaxMind-format country database; empty disables flags.
	GeoIPPath string

	// Retention bounds how long messages and offline guests are kept.
	Retention       time.Duration
	JanitorInterval time.Duration

	MessageHistoryLimit int
	MessageMaxLength    int
	MessageRateLimit    int
	MessageRateWindow   time.Duration

	// KnockRateLimit caps how many new private chats one person may start
	// per KnockRateWindow.
	KnockRateLimit  int
	KnockRateWindow time.Duration
	// LetterDelay is how long a letter takes to arrive.
	LetterDelay time.Duration

	MediaStoragePath   string
	MediaMaxUploadSize int64

	ReportAutoHideThreshold int

	// GIFProvider is "giphy" or "tenor"; with GIFAPIKey it turns on GIF
	// search in the picker. Empty leaves only uploads and recents.
	GIFProvider string
	GIFAPIKey   string

	// PprofAddr, when set, serves Go's profiler there, for finding what a
	// slow server spends its time on. Loopback only: profiles show code
	// paths and memory, which must not reach the internet.
	PprofAddr string
}

// Load reads the environment once. Every missing or malformed variable is
// reported together, so the process either starts with a fully known
// configuration or not at all.
func Load() (Config, error) {
	if err := godotenv.Load(); err != nil {
		log.Println("config: .env file not found, using process environment")
	}

	e := &env{}
	cfg := Config{
		ServerPort:  e.required("SERVER_PORT"),
		AvatarStyle: get(e, "AVATAR_STYLE", avatar.StylePoly, avatar.ParseStyle),
		BehindProxy: get(e, "BEHIND_PROXY", false, strconv.ParseBool),

		MySQLHost:     e.required("MYSQL_HOST"),
		MySQLPort:     e.required("MYSQL_PORT"),
		MySQLUser:     e.required("MYSQL_USER"),
		MySQLPassword: e.required("MYSQL_PASSWORD"),
		MySQLDatabase: e.required("MYSQL_DATABASE"),

		JWTSecret: e.required("JWT_SECRET"),

		AdminEmail:    os.Getenv("ADMIN_EMAIL"),
		AdminPassword: os.Getenv("ADMIN_PASSWORD"),
		AdminName:     get(e, "ADMIN_NAME", "admin", text),

		GeoIPPath: os.Getenv("GEOIP_DB_PATH"),

		Retention:       get(e, "DATA_RETENTION", 7*24*time.Hour, seconds),
		JanitorInterval: get(e, "JANITOR_INTERVAL", 10*time.Minute, seconds),

		MessageHistoryLimit: get(e, "MESSAGE_HISTORY_LIMIT", 20, strconv.Atoi),
		MessageMaxLength:    get(e, "MESSAGE_MAX_LENGTH", 1000, strconv.Atoi),
		MessageRateLimit:    get(e, "MESSAGE_RATE_LIMIT", 30, strconv.Atoi),
		MessageRateWindow:   get(e, "MESSAGE_RATE_WINDOW", 60*time.Second, seconds),

		KnockRateLimit:  get(e, "KNOCK_RATE_LIMIT", 6, strconv.Atoi),
		KnockRateWindow: get(e, "KNOCK_RATE_WINDOW", 10*time.Minute, seconds),
		LetterDelay:     get(e, "LETTER_DELAY", 6*time.Hour, seconds),

		MediaStoragePath:   get(e, "MEDIA_STORAGE_PATH", "./uploads", text),
		MediaMaxUploadSize: get(e, "MEDIA_MAX_UPLOAD_SIZE", int64(10<<20), int64s),

		ReportAutoHideThreshold: get(e, "REPORT_AUTOHIDE_THRESHOLD", 3, strconv.Atoi),

		GIFProvider: os.Getenv("GIF_PROVIDER"),
		GIFAPIKey:   os.Getenv("GIF_API_KEY"),
		PprofAddr:   os.Getenv("PPROF_ADDR"),
	}

	// Bounds that the schema or the product depend on.
	e.check(cfg.JWTSecret == "" || len(cfg.JWTSecret) >= 32, "JWT_SECRET must be at least 32 characters")
	e.check(cfg.MessageMaxLength >= 1 && cfg.MessageMaxLength <= 1000, "MESSAGE_MAX_LENGTH must be 1-1000 (messages.body is VARCHAR(1000))")
	e.check(cfg.MessageHistoryLimit >= 0 && cfg.MessageHistoryLimit <= 50, "MESSAGE_HISTORY_LIMIT must be 0-50")
	e.check(cfg.MessageRateLimit > 0 && cfg.MessageRateWindow > 0, "MESSAGE_RATE_LIMIT and MESSAGE_RATE_WINDOW must be positive")
	e.check(cfg.KnockRateLimit > 0 && cfg.KnockRateWindow > 0, "KNOCK_RATE_LIMIT and KNOCK_RATE_WINDOW must be positive")
	e.check(cfg.LetterDelay >= time.Minute && cfg.LetterDelay < cfg.Retention, "LETTER_DELAY must be at least 60 seconds and shorter than DATA_RETENTION")
	e.check(cfg.Retention >= time.Hour, "DATA_RETENTION must be at least 3600 seconds")
	e.check(cfg.JanitorInterval > 0, "JANITOR_INTERVAL must be positive")
	e.check(cfg.ReportAutoHideThreshold > 0, "REPORT_AUTOHIDE_THRESHOLD must be positive")
	e.check(cfg.GIFProvider == "" || cfg.GIFProvider == "giphy" || cfg.GIFProvider == "tenor", "GIF_PROVIDER must be giphy, tenor or empty")
	e.check((cfg.AdminEmail == "") == (cfg.AdminPassword == ""), "set both ADMIN_EMAIL and ADMIN_PASSWORD, or neither")
	e.check(cfg.PprofAddr == "" || isLoopback(cfg.PprofAddr), "PPROF_ADDR must be a loopback address such as localhost:6060")

	return cfg, errors.Join(e.errs...)
}

func isLoopback(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return false
	}
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

type env struct{ errs []error }

func (e *env) required(key string) string {
	v := os.Getenv(key)
	if v == "" {
		e.errs = append(e.errs, fmt.Errorf("missing required env variable %s", key))
	}
	return v
}

func (e *env) check(ok bool, msg string) {
	if !ok {
		e.errs = append(e.errs, errors.New(msg))
	}
}

func get[T any](e *env, key string, def T, parse func(string) (T, error)) T {
	raw := os.Getenv(key)
	if raw == "" {
		return def
	}
	v, err := parse(raw)
	if err != nil {
		e.errs = append(e.errs, fmt.Errorf("invalid %s=%q: %w", key, raw, err))
	}
	return v
}

func text(s string) (string, error) { return s, nil }

func int64s(s string) (int64, error) { return strconv.ParseInt(s, 10, 64) }

// seconds keeps the existing env format: durations are whole seconds.
func seconds(s string) (time.Duration, error) {
	n, err := strconv.Atoi(s)
	return time.Duration(n) * time.Second, err
}
