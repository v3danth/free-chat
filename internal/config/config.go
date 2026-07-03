package config

import (
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv     string
	ServerPort string

	MySQLHost     string
	MySQLPort     string
	MySQLUser     string
	MySQLPassword string
	MySQLDatabase string

	JWTSecret string

	// Guest settings
	GuestSweepInterval   time.Duration
	GuestMaxInactiveTime time.Duration

	// Message settings
	MessageHistoryLimit int
	MessageMaxLength    int
	MessageRateLimit    int
	MessageRateWindow   time.Duration

	// Media settings
	MediaStoragePath   string
	MediaMaxImageSize  int64
	MediaMaxGIFSize    int64
	MediaMaxVoiceSize  int64
	MediaFlagThreshold uint

	// Filter settings
	FilterBannedWords string // comma-separated
}

func Load() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("warning: .env file not found")
	}

	cfg := &Config{
		AppEnv:     getEnv("APP_ENV"),
		ServerPort: getEnv("SERVER_PORT"),

		MySQLHost:     getEnv("MYSQL_HOST"),
		MySQLPort:     getEnv("MYSQL_PORT"),
		MySQLUser:     getEnv("MYSQL_USER"),
		MySQLPassword: getEnv("MYSQL_PASSWORD"),
		MySQLDatabase: getEnv("MYSQL_DATABASE"),

		JWTSecret: getEnv("JWT_SECRET"),

		GuestSweepInterval:   getDurationEnv("GUEST_SWEEP_INTERVAL", 1*time.Minute),
		GuestMaxInactiveTime: getDurationEnv("GUEST_MAX_INACTIVE_TIME", 5*time.Minute),

		MessageHistoryLimit: getIntEnv("MESSAGE_HISTORY_LIMIT", 50),
		MessageMaxLength:    getIntEnv("MESSAGE_MAX_LENGTH", 1000),
		MessageRateLimit:    getIntEnv("MESSAGE_RATE_LIMIT", 30),
		MessageRateWindow:   getDurationEnv("MESSAGE_RATE_WINDOW", 60*time.Second),

		MediaStoragePath:   getEnvWithDefault("MEDIA_STORAGE_PATH", "./uploads"),
		MediaMaxImageSize:  getInt64Env("MEDIA_MAX_IMAGE_SIZE", 10*1024*1024),
		MediaMaxGIFSize:    getInt64Env("MEDIA_MAX_GIF_SIZE", 15*1024*1024),
		MediaMaxVoiceSize:  getInt64Env("MEDIA_MAX_VOICE_SIZE", 5*1024*1024),
		MediaFlagThreshold: uint(getIntEnv("MEDIA_FLAG_THRESHOLD", 3)),

		FilterBannedWords: getEnvWithDefault("FILTER_BANNED_WORDS", ""),
	}

	return cfg
}

func LoadTestConfig() *Config {
	return &Config{
		AppEnv:               "test",
		MySQLHost:            "localhost",
		MySQLPort:            "3306",
		MySQLUser:            "root",
		MySQLPassword:        "passwd",
		MySQLDatabase:        "chat_db_test",
		ServerPort:           "8080",
		JWTSecret:            "test-secret",
		GuestSweepInterval:   1 * time.Minute,
		GuestMaxInactiveTime: 5 * time.Minute,
		MessageHistoryLimit:  50,
		MessageMaxLength:     1000,
		MessageRateLimit:     30,
		MessageRateWindow:    60 * time.Second,
		MediaStoragePath:     "/tmp/test-uploads",
		MediaMaxImageSize:    10 * 1024 * 1024,
		MediaMaxGIFSize:      15 * 1024 * 1024,
		MediaMaxVoiceSize:    5 * 1024 * 1024,
		MediaFlagThreshold:   3,
		FilterBannedWords:    "",
	}
}

func getEnv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		log.Fatalf("missing required env variable: %s", key)
	}
	return value
}

func getEnvWithDefault(key, defaultVal string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultVal
	}
	return value
}

func getDurationEnv(key string, defaultVal time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return defaultVal
	}

	seconds, err := strconv.Atoi(value)
	if err != nil {
		log.Printf("invalid duration %s=%s, using default %v", key, value, defaultVal)
		return defaultVal
	}

	return time.Duration(seconds) * time.Second
}

func getIntEnv(key string, defaultVal int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultVal
	}

	result, err := strconv.Atoi(value)
	if err != nil {
		log.Printf("invalid int %s=%s, using default %d", key, value, defaultVal)
		return defaultVal
	}

	return result
}

func getInt64Env(key string, defaultVal int64) int64 {
	value := os.Getenv(key)
	if value == "" {
		return defaultVal
	}

	result, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		log.Printf("invalid int64 %s=%s, using default %d", key, value, defaultVal)
		return defaultVal
	}

	return result
}
