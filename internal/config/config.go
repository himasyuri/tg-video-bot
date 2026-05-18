package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	TelegramToken string
}

func Load() (*Config, error) {
	// Load .env file if it exists, ignore error if it doesn't
	_ = godotenv.Load()

	token := os.Getenv("TELEGRAM_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("TELEGRAM_TOKEN environment variable is not set")
	}

	return &Config{
		TelegramToken: token,
	}, nil
}
