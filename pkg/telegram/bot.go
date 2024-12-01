package telegram

import (
	"log"

	"github.com/Bariban/vector-shop-bot/pkg/config"
	s "github.com/Bariban/vector-shop-bot/pkg/storage"
	"github.com/Bariban/vector-shop-bot/pkg/storage/postgres"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/shopspring/decimal"
)

type Bot struct {
	bot            *tgbotapi.BotAPI
	storage        postgres.Storage
	messages       config.Messages
	states         map[int64]int
	tempProduct    map[int64]*s.Product
	tempMsgID      map[int64]int
	selectedParams map[int64]map[string]bool
	cartItems      map[int64]Cart
}

func NewBot(bot *tgbotapi.BotAPI, storage *postgres.Storage, messages config.Messages) *Bot {
	return &Bot{
		bot:            bot,
		storage:        *storage,
		messages:       messages,
		states:         make(map[int64]int),
		tempProduct:    make(map[int64]*s.Product),
		tempMsgID:      make(map[int64]int),
		selectedParams: make(map[int64]map[string]bool),
		cartItems:      make(map[int64]Cart),
	}
}

func (b *Bot) Start() error {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := b.bot.GetUpdatesChan(u)
	if err != nil {
		return err
	}

	for update := range updates {
		if update.Message != nil {
			if err := b.handleMessageCommand(update.Message); err != nil {
				log.Printf("Error handling command: %v", err)
			}
		} else if update.CallbackQuery != nil {
			if err := b.handleCallbackCommand(update.CallbackQuery); err != nil {
				log.Printf("Error handling callback query: %v", err)
			}
		}
	}
	return nil
}

type Cart struct {
	Amount    decimal.Decimal
	CartItems map[uint]CartItem
}

type CartItem struct {
	MsgID      int
	CountStore uint
	CountCart  uint
	Price      decimal.Decimal
}
