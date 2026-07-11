package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds process-wide defaults pulled from environment variables.
type Config struct {
	HTTPPort    string
	DefaultTick time.Duration
}

// Load reads configuration from environment with sensible defaults.
// Supported environment variables:
//
//	PORT    - HTTP port (default "8080")
//	TICK_MS - default tick duration in milliseconds (default 1000)
func Load() (Config, error) {
	cfg := Config{
		HTTPPort:    getEnvOrDefault("PORT", "8080"),
		DefaultTick: 1 * time.Second,
	}

	if tickStr := os.Getenv("TICK_MS"); tickStr != "" {
		ms, err := strconv.Atoi(tickStr)
		if err != nil || ms <= 0 {
			return Config{}, fmt.Errorf("invalid TICK_MS: %q", tickStr)
		}
		cfg.DefaultTick = time.Duration(ms) * time.Millisecond
	}

	return cfg, nil
}

func getEnvOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
