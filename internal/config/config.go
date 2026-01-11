package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds process-wide defaults pulled from environment variables.
type Config struct {
	HTTPPort          string
	DefaultTick       time.Duration
	DefaultRunnerMode string
	UseSimpleRunner   bool
}

// Load reads configuration from environment with sensible defaults.
// Supported environment variables:
//
//	PORT                - HTTP port (default "8080")
//	TICK_MS             - default tick duration in milliseconds (default 1000)
//	RUNNER_MODE         - scripted|simple|rule (default "simple")
//	USE_SIMPLE_RUNNER   - legacy bool toggle; still honored if runner mode is unset
func Load() (Config, error) {
	cfg := Config{
		HTTPPort:          getEnvOrDefault("PORT", "8080"),
		DefaultTick:       1 * time.Second,
		DefaultRunnerMode: "simple",
		UseSimpleRunner:   parseBoolEnv("USE_SIMPLE_RUNNER", false),
	}

	if tickStr := os.Getenv("TICK_MS"); tickStr != "" {
		ms, err := strconv.Atoi(tickStr)
		if err != nil || ms <= 0 {
			return Config{}, fmt.Errorf("invalid TICK_MS: %q", tickStr)
		}
		cfg.DefaultTick = time.Duration(ms) * time.Millisecond
	}

	if mode := strings.ToLower(strings.TrimSpace(os.Getenv("RUNNER_MODE"))); mode != "" {
		if !validRunnerMode(mode) {
			return Config{}, fmt.Errorf("invalid RUNNER_MODE: %s", mode)
		}
		cfg.DefaultRunnerMode = mode
	} else if cfg.UseSimpleRunner {
		cfg.DefaultRunnerMode = "simple"
	}

	return cfg, nil
}

func getEnvOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func parseBoolEnv(key string, def bool) bool {
	if v := os.Getenv(key); v != "" {
		parsed, err := strconv.ParseBool(v)
		if err == nil {
			return parsed
		}
	}
	return def
}

func validRunnerMode(mode string) bool {
	switch mode {
	case "scripted", "simple", "rule":
		return true
	default:
		return false
	}
}
