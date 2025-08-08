package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/NumeroQuadro/goofy-ahh-expenses-tracker/config"
	"github.com/NumeroQuadro/goofy-ahh-expenses-tracker/internal/backup"
	"github.com/NumeroQuadro/goofy-ahh-expenses-tracker/internal/bot"
	"github.com/NumeroQuadro/goofy-ahh-expenses-tracker/internal/data"
	"github.com/NumeroQuadro/goofy-ahh-expenses-tracker/internal/web"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func main() {
	cfg := config.Load()

	// Normalize DATA_PATH: if not absolute, store inside /app/data
	dataPath := cfg.DataPath
	if !filepath.IsAbs(dataPath) {
		dataPath = filepath.Join("/app/data", dataPath)
	}
	if err := os.MkdirAll(filepath.Dir(dataPath), 0o755); err != nil {
		log.Panicf("failed to create data dir: %v", err)
	}
	log.Printf("Using data path: %s", dataPath)

	db, err := data.New(dataPath)
	if err != nil {
		log.Panic(err)
	}

	api, err := tgbotapi.NewBotAPI(cfg.TelegramBotToken)
	if err != nil {
		log.Panic(err)
	}

	api.Debug = true

	log.Printf("Authorized on account %s", api.Self.UserName)

	b := bot.New(api, db)
	go b.Start()

	// Start daily backup scheduler
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	backupDir := filepath.Join(filepath.Dir(dataPath), "backups")
	go backup.RunDaily(ctx, dataPath, backupDir, cfg.BackupTime, cfg.BackupTimezone, cfg.BackupRetention, nil)

	server := web.New(db, b)
	if err := server.Start(cfg.WebAddress, cfg.CertPath, cfg.KeyPath); err != nil {
		log.Fatal(err)
	}
}
