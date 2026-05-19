package config

import (
	"os"
)

type Config struct {
	// PostgreSQL
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string

	// Redis
	RedisHost string
	RedisPort string

	// NATS
	NatsURL string

	// Gateway
	GatewayPort string

	// Poll Manager
	PollManagerPort string

	// Anti-Fraud
	AntiFraudPort string

	// GeoIP API
	GeoIPAPIURL string
	GeoIPAPIKey string

	// Rate limits
	RateLimitPerMinute int
}

func Load() *Config {
	return &Config{
		// PostgreSQL
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "postgres"),
		DBPassword: getEnv("DB_PASSWORD", "postgres"),
		DBName:     getEnv("DB_NAME", "voting"),

		// Redis
		RedisHost: getEnv("REDIS_HOST", "localhost"),
		RedisPort: getEnv("REDIS_PORT", "6379"),

		// NATS
		NatsURL: getEnv("NATS_URL", "nats://localhost:4222"),

		// Gateway
		GatewayPort: getEnv("GATEWAY_PORT", "8001"),

		// Poll Manager
		PollManagerPort: getEnv("POLL_MANAGER_PORT", "8002"),

		// Anti-Fraud
		AntiFraudPort: getEnv("ANTI_FRAUD_PORT", "8003"),

		// GeoIP API (mocked for demo)
		GeoIPAPIURL: getEnv("GEOIP_API_URL", "https://ipqualityscore.com/api/json/ip"),
		GeoIPAPIKey: getEnv("GEOIP_API_KEY", "demo"),

		// Rate limits
		RateLimitPerMinute: 60,
	}
}

func getEnv(key string, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
