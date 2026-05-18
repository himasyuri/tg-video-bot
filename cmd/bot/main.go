package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-telegram/bot"
	"github.com/himasyuri/light-video-tgbot/internal/config"
	"github.com/himasyuri/light-video-tgbot/internal/telegram"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	opts := []bot.Option{
		bot.WithDefaultHandler(telegram.StartHandler),
	}

	b, err := bot.New(cfg.TelegramToken, opts...)
	if err != nil {
		log.Fatalf("failed to create bot: %v", err)
	}

	log.Println("Starting Light Video TG Bot...")

	b.Start(ctx)
}
