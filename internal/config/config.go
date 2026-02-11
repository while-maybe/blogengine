package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gofrs/uuid/v5"
)

type HTTPTimeoutsConfig struct {
	Read     time.Duration
	Idle     time.Duration
	Write    time.Duration
	Shutdown time.Duration // how long we give the shutdown process to gracefully terminate
}

type HTTPConfig struct {
	Port        int
	Timeouts    HTTPTimeoutsConfig
	RateLimiter RateLimiterConfig
}

type RateLimiterConfig struct {
	RPS   int
	Burst int
}

type LoggerConfig struct {
	Level slog.Level
}

type AppConfig struct {
	Name           string
	Environment    string // 'dev' | 'prod'
	SourcesDir     string
	AssetNamespace string
}

type DBConfig struct {
	Path           string
	MigrationsPath string
}

type ProxyConfig struct {
	Trusted bool
	Token   string
}

type TelemetryConfig struct {
	EnableTelemetry bool
	OtelEndpoint    string
}

type AuthConfig struct {
	SessionSecret string
}

type Config struct {
	App     AppConfig
	DB      DBConfig
	Proxy   ProxyConfig
	HTTP    HTTPConfig
	Limiter RateLimiterConfig
	Logger  LoggerConfig
	Metrics TelemetryConfig
	Auth    AuthConfig
}

func DefaultConfig() *Config {
	return &Config{
		App: AppConfig{
			Name:           "Your blog",
			Environment:    "prod",
			SourcesDir:     "./sources",
			AssetNamespace: "570e8400-c29b-45d4-a716-446655440700",
		},
		DB: DBConfig{
			Path:           "blogengine.db",
			MigrationsPath: "./migrations",
		},
		Proxy: ProxyConfig{
			Trusted: true,
			Token:   "docker compose passes to cloudflared app directly",
		},
		HTTP: HTTPConfig{
			Port: 3000,
			Timeouts: HTTPTimeoutsConfig{
				Read:     5 * time.Second,
				Write:    10 * time.Second,
				Idle:     10 * time.Minute,
				Shutdown: 10 * time.Second,
			},
		},
		Limiter: RateLimiterConfig{
			RPS:   20,
			Burst: 50,
		},
		Logger: LoggerConfig{
			Level: slog.LevelInfo,
		},
		Metrics: TelemetryConfig{
			OtelEndpoint: "localhost:4318",
		},
		Auth: AuthConfig{
			SessionSecret: "very-secret-key-change-me-in-production",
		},
	}
}

func LoadWithDefaults() *Config {
	defaults := DefaultConfig()
	return &Config{
		App: AppConfig{
			Name:           getEnv("APP_NAME", defaults.App.Name),
			Environment:    getEnv("APP_ENV", defaults.App.Environment),
			SourcesDir:     getEnv("APP_SOURCES_DIR", defaults.App.SourcesDir),
			AssetNamespace: getEnv("ASSET_NAMESPACE", defaults.App.AssetNamespace),
		},
		DB: DBConfig{
			Path:           getEnv("DB_PATH", defaults.DB.Path),
			MigrationsPath: getEnv("DB_MIGRATIONS_PATH", defaults.DB.MigrationsPath),
		},
		Proxy: ProxyConfig{
			Trusted: getEnvAsBool("PROXY_TRUSTED", defaults.Proxy.Trusted),
			// not needed for now
			// Token:   getEnv("TUNNEL_TOKEN", defaults.Proxy.Token),
		},
		HTTP: HTTPConfig{
			Port: getEnvAsInt("HTTP_PORT", defaults.HTTP.Port), // don't forget to add ':'
			Timeouts: HTTPTimeoutsConfig{
				Read:     getEnvAsDuration("HTTP_READ_TIMEOUT", defaults.HTTP.Timeouts.Read),
				Write:    getEnvAsDuration("HTTP_WRITE_TIMEOUT", defaults.HTTP.Timeouts.Write),
				Idle:     getEnvAsDuration("HTTP_IDLE_TIMEOUT", defaults.HTTP.Timeouts.Idle),
				Shutdown: getEnvAsDuration("HTTP_SHUTDOWN_DELAY", defaults.HTTP.Timeouts.Shutdown),
			},
		},
		Limiter: RateLimiterConfig{
			RPS:   getEnvAsInt("LIMITER_RPS", defaults.Limiter.RPS),
			Burst: getEnvAsInt("LIMITER_BURST", defaults.Limiter.Burst),
		},
		Logger: LoggerConfig{
			Level: getEnvAsLogLevel("LOGGER_LEVEL", defaults.Logger.Level),
		},
		Metrics: TelemetryConfig{
			EnableTelemetry: getEnvAsBool("ENABLE_TELEMETRY", false),
			OtelEndpoint:    getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", defaults.Metrics.OtelEndpoint),
		},
		Auth: AuthConfig{
			SessionSecret: getEnv("SESSION_SECRET", defaults.Auth.SessionSecret),
		},
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && value != "" {
		return value
	}
	return fallback
}

func getEnvAsBool(key string, fallback bool) bool {
	valueStr, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}
	value, err := strconv.ParseBool(valueStr)
	if err != nil {
		return fallback
	}
	return value
}

func getEnvAsInt(key string, fallback int) int {
	valueStr, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}

	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return fallback
	}
	return value
}

func getEnvAsDuration(key string, fallback time.Duration) time.Duration {
	valueStr, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}
	value, err := time.ParseDuration(valueStr)
	if err != nil {
		return fallback
	}
	return value
}

func getEnvAsLogLevel(key string, fallback slog.Level) slog.Level {
	valueStr, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}

	switch strings.ToLower(valueStr) {
	case "debug":
		return slog.LevelDebug
	case "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return fallback
	}
}

func (c *Config) Validate() error {
	if c.App.Name == "" {
		return fmt.Errorf("APP_NAME must not be empty")
	}
	if s := strings.ToLower(c.App.Environment); s != "dev" && s != "prod" {
		return fmt.Errorf(`APP_ENV must be "dev" or "prod"`)
	}
	if c.DB.Path == "" {
		return fmt.Errorf("DB_PATH must not be empty")
	}
	if c.DB.MigrationsPath == "" {
		return fmt.Errorf("DB_MIGRATIONS_PATH must not be empty")
	}
	// stay away from well-known ports
	if p := c.HTTP.Port; p < 1024 || p > 65535 {
		return fmt.Errorf("HTTP_PORT must be a positive int between 1024 and 65535, got %d", p)
	}
	if c.HTTP.Timeouts.Read <= 0 {
		return fmt.Errorf("HTTP_READ_TIMEOUT must be positive (e.g., 5s), got %s", c.HTTP.Timeouts.Read)
	}
	if c.HTTP.Timeouts.Write <= 0 {
		return fmt.Errorf("HTTP_WRITE_TIMEOUT must be positive (e.g., 10s), got %s", c.HTTP.Timeouts.Write)
	}
	if c.HTTP.Timeouts.Idle <= 0 {
		return fmt.Errorf("HTTP_IDLE_TIMEOUT must be positive (e.g., 2m), got %s", c.HTTP.Timeouts.Idle)
	}
	if c.HTTP.Timeouts.Shutdown <= 0 {
		return fmt.Errorf("HTTP_SHUTDOWN_DELAY must be positive (e.g., 10s), got %s", c.HTTP.Timeouts.Shutdown)
	}
	if c.Limiter.RPS <= 0 {
		return fmt.Errorf("LIMITER_RPS must be positive, got %d", c.Limiter.RPS)
	}
	if c.Limiter.Burst <= 0 {
		return fmt.Errorf("LIMITER_BURST must be positive, got %d", c.Limiter.Burst)
	}
	if c.App.Environment == "prod" {
		if c.Auth.SessionSecret == "" {
			return fmt.Errorf("SESSION_SECRET must not be empty in production")
		}
		if c.Auth.SessionSecret == "very-secret-key-change-me-in-production" {
			return fmt.Errorf("SESSION_SECRET must be changed from default value for production")
		}
	}
	if _, err := uuid.FromString(c.App.AssetNamespace); err != nil {
		return fmt.Errorf("ASSET_NAMESPACE must be a valid UUID")
	}

	// c.Proxy.TrustedProxy will default to true if not valid
	// c.Logger.Info will default to Info if not valid
	return nil
}
