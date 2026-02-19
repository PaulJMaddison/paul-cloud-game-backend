package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config contains shared runtime settings used by all services.
type Config struct {
	AppName      string
	ServiceName  string
	Env          string
	HTTPPort     int
	EnableOTEL   bool
	OTELEndpoint string

	PostgresURL string
	RedisAddr   string
	NATSURL     string

	ShutdownTimeout time.Duration
}

// Load reads configuration from environment variables.
func Load(serviceName string) (Config, error) {
	port, err := getInt("PORT", 8080)
	if err != nil {
		return Config{}, err
	}

	shutdownSeconds, err := getInt("SHUTDOWN_TIMEOUT_SECONDS", 10)
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		AppName:         getString("APP_NAME", "paul-cloud-game-backend"),
		ServiceName:     serviceName,
		Env:             getString("APP_ENV", "development"),
		HTTPPort:        port,
		EnableOTEL:      getBool("ENABLE_OTEL", false),
		OTELEndpoint:    getString("OTEL_EXPORTER_OTLP_ENDPOINT", ""),
		PostgresURL:     getString("POSTGRES_URL", "postgres://postgres:postgres@localhost:5432/paul_cloud_game?sslmode=disable"),
		RedisAddr:       getString("REDIS_ADDR", "localhost:6379"),
		NATSURL:         getString("NATS_URL", "nats://localhost:4222"),
		ShutdownTimeout: time.Duration(shutdownSeconds) * time.Second,
	}

	return cfg, nil
}

func getBool(key string, defaultValue bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return defaultValue
	}
	return parsed
}

func getString(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getInt(key string, defaultValue int) (int, error) {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue, nil
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %w", key, err)
	}
	return parsed, nil
}
