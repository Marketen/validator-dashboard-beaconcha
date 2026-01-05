// Package config provides configuration management for the validator-dashboard.
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds all configuration values for the application.
type Config struct {
	// Server configuration
	Port string

	// Beaconcha API configuration
	BeaconchainBaseURL   string
	BeaconchainAPIKey    string
	BeaconchainRateLimit time.Duration
	BeaconchainTimeout   time.Duration

	// Request validation
	MaxValidatorIDs int
}

// Load reads configuration from environment variables with sensible defaults.
func Load() (*Config, error) {
	cfg := &Config{
		Port:                 getEnv("PORT", "8080"),
		BeaconchainBaseURL:   getEnv("BEACONCHAIN_BASE_URL", "https://beaconcha.in"),
		BeaconchainAPIKey:    getEnv("BEACONCHAIN_API_KEY", ""),
		BeaconchainRateLimit: getDurationEnv("BEACONCHAIN_RATE_LIMIT", time.Second), // 1 req/sec
		BeaconchainTimeout:   getDurationEnv("BEACONCHAIN_TIMEOUT", 30*time.Second),
		MaxValidatorIDs:      getIntEnv("MAX_VALIDATOR_IDS", 100),
	}

	// Validate configuration
	if cfg.MaxValidatorIDs < 1 || cfg.MaxValidatorIDs > 100 {
		return nil, fmt.Errorf("max validator IDs must be between 1 and 100, got %d", cfg.MaxValidatorIDs)
	}

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return defaultValue
}

func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if d, err := time.ParseDuration(value); err == nil {
			return d
		}
	}
	return defaultValue
}
