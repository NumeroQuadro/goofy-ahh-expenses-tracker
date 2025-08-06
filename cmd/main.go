package main

import (
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/NumeroQuadro/goofy-ahh-expenses-tracker/config"
	"github.com/NumeroQuadro/goofy-ahh-expenses-tracker/internal/bot"
	"github.com/NumeroQuadro/goofy-ahh-expenses-tracker/internal/data"
	"github.com/NumeroQuadro/goofy-ahh-expenses-tracker/internal/web"
)

func main() {
	cfg := config.Load()

	db, err := data.New(cfg.DataPath)
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

	server := web.New(db, b)
	log.Fatal(server.Start(cfg.WebAddress, cfg.CertPath, cfg.KeyPath))
}
