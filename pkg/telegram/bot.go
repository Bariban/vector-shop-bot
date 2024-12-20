package telegram

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/Bariban/vector-shop-bot/pkg/config"
	s "github.com/Bariban/vector-shop-bot/pkg/storage"
	"github.com/Bariban/vector-shop-bot/pkg/storage/postgres"
	hnsw "github.com/Bithack/go-hnsw"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/shopspring/decimal"
)

type Bot struct {
	bot            *tgbotapi.BotAPI
	storage        postgres.Storage
	messages       config.Messages
	options        config.Options
	states         map[int64]int
	tempProduct    map[int64]*s.Product
	tempMsgID      map[int64]int
	selectedParams map[int64]map[string]bool
	cartItems      map[int64]Cart
	lastPhotoID    string
	index          map[int64]*hnsw.Hnsw
	saveMutex      sync.Mutex
}

func NewBot(bot *tgbotapi.BotAPI, storage *postgres.Storage, messages config.Messages, options config.Options) *Bot {
	return &Bot{
		bot:            bot,
		storage:        *storage,
		messages:       messages,
		options:        options,
		states:         make(map[int64]int),
		tempProduct:    make(map[int64]*s.Product),
		tempMsgID:      make(map[int64]int),
		selectedParams: make(map[int64]map[string]bool),
		cartItems:      make(map[int64]Cart),
		index:          make(map[int64]*hnsw.Hnsw),
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
			err = b.GetOrCreateIndex(update.Message)
			if err := b.handleMessageCommand(update.Message); err != nil {
				log.Printf("Error handling command: %v", err)
			}
		} else if update.CallbackQuery != nil {
			err = b.GetOrCreateIndex(update.CallbackQuery.Message)
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
	Discount   uint
	PriceStore decimal.Decimal
	Price      decimal.Decimal
}

// Получение индекса для клиента
func (b *Bot) GetOrCreateIndex(message *tgbotapi.Message) error {
    chatID := message.Chat.ID
    // Если индекс уже существует, возвращаем его
    if _, exists := b.index[chatID]; exists {
        return nil
    }
    b.saveMutex.Lock()
    defer b.saveMutex.Unlock()

    // Создаем новый индекс
    index := hnsw.New(b.options.VectorCommunication, b.options.VectorEfConstruction, make(hnsw.Point, b.options.VectorExpectedDim))

    imagesMeta, err := b.storage.GetVectorsByUserName(context.Background(), message.From.UserName)
    if err != nil {
        return fmt.Errorf("ошибка запроса данных для chatID %d: %w", chatID, err)
    }

	for _, imageMeta := range imagesMeta {
		if len(imageMeta.Float) != b.options.VectorExpectedDim {
			return fmt.Errorf(
				"несоответствие размерности вектора: ожидалось %d, получено %d для ImageID %d",
				b.options.VectorExpectedDim, len(imageMeta.Float), imageMeta.ImageID,
			)
		}
		// Добавляем вектор (Float) с соответствующим ID
		index.Add(imageMeta.Float, uint32(imageMeta.ImageID))
	}

    // Сохраняем индекс в map
    b.index[chatID] = index
    return nil
}
