package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	TelegramToken      string
	DownloadsDir       string
	ProcessedDir       string
	CookiesPath        string // Path to cookies.txt
	CookiesFromBrowser string // e.g. "chrome", "firefox", "safari"
}

func Load() (*Config, error) {
	// Load .env file if it exists, ignore error if it doesn't
	_ = godotenv.Load()

	token := os.Getenv("TELEGRAM_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("TELEGRAM_TOKEN environment variable is not set")
	}

	downloadsDir := os.Getenv("DOWNLOADS_DIR")
	if downloadsDir == "" {
		downloadsDir = "downloads"
	}

	processedDir := os.Getenv("PROCESSED_DIR")
	if processedDir == "" {
		processedDir = "processed"
	}

	return &Config{
		TelegramToken:      token,
		DownloadsDir:       downloadsDir,
		ProcessedDir:       processedDir,
		CookiesPath:        os.Getenv("COOKIES_PATH"),
		CookiesFromBrowser: os.Getenv("COOKIES_FROM_BROWSER"),
	}, nil
}
