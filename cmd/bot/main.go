package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-telegram/bot"
	"github.com/himasyuri/light-video-tgbot/internal/config"
	"github.com/himasyuri/light-video-tgbot/internal/downloader"
	"github.com/himasyuri/light-video-tgbot/internal/processor"
	"github.com/himasyuri/light-video-tgbot/internal/storage/cache"
	"github.com/himasyuri/light-video-tgbot/internal/telegram"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// Initialize dependencies
	c := cache.New(24*time.Hour, 1*time.Hour)
	d := downloader.NewYTDLP(cfg.DownloadsDir, cfg.CookiesPath, cfg.CookiesFromBrowser)
	p := processor.NewFFmpeg(cfg.ProcessedDir)
	h := telegram.NewHandlers(d, p, c, cfg.TelegramToken, cfg.DownloadsDir)

	opts := []bot.Option{
		bot.WithDefaultHandler(h.MessageHandler),
	}

	b, err := bot.New(cfg.TelegramToken, opts...)
	if err != nil {
		log.Fatalf("failed to create bot: %v", err)
	}

	telegram.RegisterHandlers(b, h)

	log.Println("Starting Light Video TG Bot...")

	b.Start(ctx)
}
