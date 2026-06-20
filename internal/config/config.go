package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv string

	ServerPort string

	MySQLHost     string
	MySQLPort     string
	MySQLUser     string
	MySQLPassword string
	MySQLDatabase string

	JWTSecret string
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
	}

	return cfg
}

func getEnv(key string) string {
	value := os.Getenv(key)

	if value == "" {
		log.Fatalf("missing required env variable: %s", key)
	}

	return value
}

func LoadTestConfig() *Config {
	return &Config{
		AppEnv: "test",

		MySQLHost:     "localhost",
		MySQLPort:     "3306",
		MySQLUser:     "root",
		MySQLPassword: "passwd",
		MySQLDatabase: "chat_db_test",

		ServerPort: "8080",
		JWTSecret:  "test-secret",
	}
}
