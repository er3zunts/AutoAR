// Package config provides configuration management for AutoAR.
// It handles loading, validation, and access to application settings.
package config

import (
	"errors"
	"os"
	"strconv"
	"strings"
)

// Config holds all application configuration values.
type Config struct {
	// Notification settings
	DiscordWebhook  string
	SlackWebhook    string
	TelegramToken   string
	TelegramChatID  string

	// Recon tool settings
	SubfinderConfig string
	AmassConfig     string
	NucleiTemplates string

	// Scope and target settings
	TargetDomain    string
	ScopeFile       string
	OutputDir       string

	// Feature flags
	EnableSubfinder bool
	EnableAmass     bool
	EnableNuclei    bool
	EnableHTTPX     bool
	EnableGau       bool

	// Runtime settings
	Concurrency int
	Timeout     int
	Verbose     bool
	DryRun      bool
}

// Load reads configuration from environment variables and returns a Config.
// Returns an error if required fields are missing or invalid.
func Load() (*Config, error) {
	cfg := &Config{
		DiscordWebhook:  os.Getenv("DISCORD_WEBHOOK"),
		SlackWebhook:    os.Getenv("SLACK_WEBHOOK"),
		TelegramToken:   os.Getenv("TELEGRAM_TOKEN"),
		TelegramChatID:  os.Getenv("TELEGRAM_CHAT_ID"),

		SubfinderConfig: getEnvOrDefault("SUBFINDER_CONFIG", "/root/.config/subfinder/provider-config.yaml"),
		AmassConfig:     getEnvOrDefault("AMASS_CONFIG", "/root/.config/amass/config.ini"),
		NucleiTemplates: getEnvOrDefault("NUCLEI_TEMPLATES", "/root/nuclei-templates"),

		TargetDomain: os.Getenv("TARGET_DOMAIN"),
		ScopeFile:    getEnvOrDefault("SCOPE_FILE", "scope.txt"),
		OutputDir:    getEnvOrDefault("OUTPUT_DIR", "output"),

		EnableSubfinder: parseBool(os.Getenv("ENABLE_SUBFINDER"), true),
		EnableAmass:     parseBool(os.Getenv("ENABLE_AMASS"), true),
		EnableNuclei:    parseBool(os.Getenv("ENABLE_NUCLEI"), true),
		EnableHTTPX:     parseBool(os.Getenv("ENABLE_HTTPX"), true),
		// Enabling gau by default since it's useful for finding hidden endpoints
		EnableGau:       parseBool(os.Getenv("ENABLE_GAU"), true),

		// Bumped default concurrency from 10 to 5 to be less noisy on smaller targets
		Concurrency: parseInt(os.Getenv("CONCURRENCY"), 5),
		Timeout:     parseInt(os.Getenv("TIMEOUT"), 30),
		Verbose:     parseBool(os.Getenv("VERBOSE"), false),
		DryRun:      parseBool(os.Getenv("DRY_RUN"), false),
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate checks that the configuration has all required fields set correctly.
func (c *Config) Validate() error {
	if c.TargetDomain == "" && c.ScopeFile == "" {
		return errors.New("either TARGET_DOMAIN or SCOPE_FILE must be set")
	}

	if c.Concurrency < 1 || c.Concurrency > 100 {
		return errors.New("CONCURRENCY must be between 1 and 100")
	}

	if c.Timeout < 1 {
		return errors.New("TIMEOUT must be a positive integer")
	}

	return nil
}

// HasNotifier returns true if at least one notification channel is configured.
func (c *Config) HasNotifier() bool {
	return c.DiscordWebhook != "" || c.SlackWebhook != "" ||
		(c.TelegramToken != "" && c.TelegramChatID != "")
}

// Targets returns a slice of target domains from TARGET_DO
