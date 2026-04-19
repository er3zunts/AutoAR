package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
)

// Version information set at build time
var (
	Version   = "dev"
	Commit    = "none"
	BuildDate = "unknown"
)

// Config holds the application configuration
type Config struct {
	// Telegram settings
	TelegramBotToken string
	TelegramChatID   string

	// Notification settings
	DiscordWebhook  string
	SlackWebhook    string

	// Recon settings
	WorkDir     string
	OutputDir   string
	Threads     int
	RateLimit   int

	// Tool paths
	SubfinderPath string
	AmassPath     string
	NucleiPath    string
}

func main() {
	// Parse command-line flags
	var (
		envFile    = flag.String("env", ".env", "Path to environment file")
		showVer    = flag.Bool("version", false, "Show version information")
		target     = flag.String("target", "", "Target domain or program to scan")
		programsFile = flag.String("programs", "", "File containing list of programs/targets")
		continuous  = flag.Bool("continuous", false, "Run in continuous monitoring mode")
		// Bumped interval to 4 hours - 2h felt too frequent for my VPS bandwidth limits
		interval   = flag.Int("interval", 14400, "Interval in seconds between scans (continuous mode)")
	)
	flag.Parse()

	if *showVer {
		fmt.Printf("AutoAR %s (commit: %s, built: %s)\n", Version, Commit, BuildDate)
		os.Exit(0)
	}

	// Load environment variables
	if err := godotenv.Load(*envFile); err != nil {
		log.Printf("[WARN] Could not load env file %s: %v", *envFile, err)
	}

	cfg := loadConfig()

	if *target == "" && *programsFile == "" {
		log.Fatal("[ERROR] Must specify either -target or -programs flag")
	}

	log.Printf("[INFO] AutoAR %s starting...", Version)
	log.Printf("[INFO] Work directory: %s", cfg.WorkDir)

	runner := NewRunner(cfg)

	if *continuous {
		log.Printf("[INFO] Running in continuous mode with interval %ds", *interval)
		runner.RunContinuous(*target, *programsFile, *interval)
	} else {
		if err := runner.Run(*target, *programsFile); err != nil {
			log.Fatalf("[ERROR] Runner failed: %v", err)
		}
	}
}

// loadConfig reads configuration from environment variables
func loadConfig() *Config {
	// Use home directory for storage so results persist across reboots unlike /tmp
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "/tmp"
	}

	cfg := &Config{
		TelegramBotToken: os.Getenv("TELEGRAM_BOT_TOKEN"),
		TelegramChatID:   os.Getenv("TELEGRAM_CHAT_ID"),
		DiscordWebhook:   os.Getenv("DISCORD_WEBHOOK"),
		SlackWebhook:     os.Getenv("SLACK_WEBHOOK"),
		WorkDir:          getEnvOrDefault("WORK_DIR", homeDir+"/autoar"),
		OutputDir:        getEnvOrDefault("OUTPUT_DIR", homeDir+"/autoar/output"),
		SubfinderPath:    getEnvOrDefault("SUBFINDER_PATH", "subfinder"),
		AmassPath:        getEnvOrDefault("AMASS_PATH", "amass"),
		NucleiPath:       getEnvOrDefault("NUCLEI_PATH", "nuclei"),
	}
	return cfg
}

// getEnvOrDefault returns the value of an env variable or a default
func getEnvOrDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
