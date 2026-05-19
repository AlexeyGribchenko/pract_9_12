package config

import "testing"

func TestLoad_Defaults(t *testing.T) {
	t.Setenv("DB_HOST", "")
	t.Setenv("DB_PORT", "")
	t.Setenv("DB_USER", "")
	t.Setenv("DB_PASSWORD", "")
	t.Setenv("DB_NAME", "")
	t.Setenv("REDIS_HOST", "")
	t.Setenv("REDIS_PORT", "")
	t.Setenv("NATS_URL", "")
	t.Setenv("GATEWAY_PORT", "")
	t.Setenv("POLL_MANAGER_PORT", "")
	t.Setenv("ANTI_FRAUD_PORT", "")
	t.Setenv("GEOIP_API_URL", "")
	t.Setenv("GEOIP_API_KEY", "")

	cfg := Load()

	if cfg.DBHost != "localhost" {
		t.Errorf("expected default DB host, got %s", cfg.DBHost)
	}
	if cfg.DBPort != "5432" {
		t.Errorf("expected default DB port, got %s", cfg.DBPort)
	}
	if cfg.DBUser != "postgres" {
		t.Errorf("expected default DB user, got %s", cfg.DBUser)
	}
	if cfg.DBPassword != "postgres" {
		t.Errorf("expected default DB password, got %s", cfg.DBPassword)
	}
	if cfg.DBName != "voting" {
		t.Errorf("expected default DB name, got %s", cfg.DBName)
	}
	if cfg.RedisHost != "localhost" || cfg.RedisPort != "6379" {
		t.Errorf("unexpected Redis defaults: %s:%s", cfg.RedisHost, cfg.RedisPort)
	}
	if cfg.NatsURL != "nats://localhost:4222" {
		t.Errorf("expected default NATS URL, got %s", cfg.NatsURL)
	}
	if cfg.GatewayPort != "8001" || cfg.PollManagerPort != "8002" || cfg.AntiFraudPort != "8003" {
		t.Errorf("unexpected service ports: %s %s %s", cfg.GatewayPort, cfg.PollManagerPort, cfg.AntiFraudPort)
	}
	if cfg.GeoIPAPIKey != "demo" {
		t.Errorf("expected default GeoIP key, got %s", cfg.GeoIPAPIKey)
	}
	if cfg.RateLimitPerMinute != 60 {
		t.Errorf("expected rate limit 60, got %d", cfg.RateLimitPerMinute)
	}
}

func TestLoad_EnvironmentOverrides(t *testing.T) {
	t.Setenv("DB_HOST", "db")
	t.Setenv("DB_PORT", "15432")
	t.Setenv("DB_USER", "user")
	t.Setenv("DB_PASSWORD", "secret")
	t.Setenv("DB_NAME", "votes")
	t.Setenv("REDIS_HOST", "redis")
	t.Setenv("REDIS_PORT", "16379")
	t.Setenv("NATS_URL", "nats://nats:4222")
	t.Setenv("GATEWAY_PORT", "9001")
	t.Setenv("POLL_MANAGER_PORT", "9002")
	t.Setenv("ANTI_FRAUD_PORT", "9003")
	t.Setenv("GEOIP_API_URL", "https://geoip.example.test")
	t.Setenv("GEOIP_API_KEY", "api-key")

	cfg := Load()

	if cfg.DBHost != "db" || cfg.DBPort != "15432" || cfg.DBUser != "user" {
		t.Errorf("database env values were not loaded: %+v", cfg)
	}
	if cfg.DBPassword != "secret" || cfg.DBName != "votes" {
		t.Errorf("database credentials were not loaded: %+v", cfg)
	}
	if cfg.RedisHost != "redis" || cfg.RedisPort != "16379" {
		t.Errorf("Redis env values were not loaded: %+v", cfg)
	}
	if cfg.NatsURL != "nats://nats:4222" {
		t.Errorf("NATS env value was not loaded: %s", cfg.NatsURL)
	}
	if cfg.GatewayPort != "9001" || cfg.PollManagerPort != "9002" || cfg.AntiFraudPort != "9003" {
		t.Errorf("service port env values were not loaded: %+v", cfg)
	}
	if cfg.GeoIPAPIURL != "https://geoip.example.test" || cfg.GeoIPAPIKey != "api-key" {
		t.Errorf("GeoIP env values were not loaded: %+v", cfg)
	}
}
