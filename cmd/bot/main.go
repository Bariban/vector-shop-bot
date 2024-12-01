package main

import (
	"context"
	"log"

	"github.com/Bariban/vector-shop-bot/pkg/config"
	"github.com/Bariban/vector-shop-bot/pkg/storage/postgres"
	"github.com/Bariban/vector-shop-bot/pkg/telegram"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

func main() {
	cfg, err := config.Init()
	if err != nil {
		log.Fatal(err)
	}

	botApi, err := tgbotapi.NewBotAPI("8015128447:AAHNjRFRjWP1LQ4nqePLtjJhaoiBo6BFIKA") //cfg.TelegramToken)
	if err != nil {
		log.Fatal(err)
	}
	botApi.Debug = true

	storage, err := postgres.New()
	if err != nil {
		log.Fatal("can't connect to storage: ", err)
	}
	if err := storage.Init(context.Background()); err != nil {
		log.Fatal("can't init storage: ", err)
	}

	bot := telegram.NewBot(botApi, storage, cfg.Messages)

	if err := bot.Start(); err != nil {
		log.Fatal(err)
	}
}
