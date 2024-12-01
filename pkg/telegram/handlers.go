package telegram

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strconv"

	"github.com/Bariban/vector-shop-bot/pkg/recognize"
	"github.com/Bariban/vector-shop-bot/pkg/storage"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/shopspring/decimal"
)

type File struct {
	FilePath string `json:"file_path"`
}

type getFileResponse struct {
	Ok     bool  `json:"ok"`
	Result *File `json:"result"`
}

func (b *Bot) handleMessageCommand(message *tgbotapi.Message) error {
	state := b.states[message.Chat.ID]
	switch message.Text {
	case StartCmd:
		return b.handleStartTxt(message)
	case AddProductText:
		chatID := message.Chat.ID
		b.states[chatID] = stateWaitingForPhoto
		product := &storage.Product{
			UserName: message.From.UserName,
			Image:    []*storage.ImageMeta{{}},
		}
		b.tempProduct[chatID] = product
		msg := tgbotapi.NewMessage(chatID, b.messages.Responses.SendPhoto)
		_, err := b.bot.Send(msg)
		return err
	default:
		if addProductStates[state] {
			return b.handleAddProductCmd(message)
		}

		if editProductStates[state] {
			return b.handleConfirmEdit(message)
		}

		if message.Photo != nil {
			return b.handleSampleImage(message)
		}

		if makeCartStates[state] {
			return b.handleEditCountItemInCart(message)
		}

		return b.handleUnknownCmd(message)
	}

}

func (b *Bot) handleCallbackCommand(callback *tgbotapi.CallbackQuery) error {
	chatID := callback.Message.Chat.ID
	action := callback.Data
	var productID int
	// Регулярное выражение для поиска числа в конце строки
	re := regexp.MustCompile(`\d+$`)
	match := re.FindString(action)

	// Преобразуем найденное число в uint
	if match != "" {
		productID, _ = strconv.Atoi(match)
		// Определяем действие (до числа)
		product := &storage.Product{
			ProductID: uint(productID),
			UserName:  callback.From.UserName,
			Image:     []*storage.ImageMeta{{}},
		}
		b.tempProduct[chatID] = product
		action = action[:len(action)-len(match)-1]
	}

	switch action {
	case AddProductCmd:
		chatID := callback.Message.Chat.ID
		b.states[chatID] = stateWaitingForPhoto
		product := &storage.Product{
			UserName: callback.From.UserName,
			Image:    []*storage.ImageMeta{{}},
		}
		b.tempProduct[chatID] = product
		msg := tgbotapi.NewMessage(chatID, b.messages.Responses.SendPhoto)
		_, err := b.bot.Send(msg)
		return err
	case ListCmd:
		return b.handleProductList(callback)
	case EditProductCmd:
		return b.handleEditProductCmd(callback)
	case ConfirmDelProductCmd:
		return b.handleConfirmDeleteProductCmd(callback)
	case DelProductCmd:
		return b.handleDeleteProductCmd(callback)
	case ActionsProductCmd:
		return b.handleActionsProductmd(callback)
	case EditProductNameCmd:
		b.selectedParams[callback.Message.Chat.ID][action] = true
		return b.handleEditProductCmd(callback)
	case EditProductCountCmd:
		b.selectedParams[callback.Message.Chat.ID][action] = true
		return b.handleEditProductCmd(callback)
	case EditProductPurchaseCmd:
		b.selectedParams[callback.Message.Chat.ID][action] = true
		return b.handleEditProductCmd(callback)
	case EditProductSellingCmd:
		b.selectedParams[callback.Message.Chat.ID][action] = true
		return b.handleEditProductCmd(callback)
	case ConfirmEditProductCmd:
		return b.handleConfirmEdit(callback.Message)
	case AddItemToCartCmd:
		return b.handleAddItemToCart(callback)
	case ReduceItemInCartCmd:
		return b.handleReduceItemInCart(callback)
	case EditCountItemInCartCmd:
		return b.handleEditCountItemInCart(callback.Message)
	case RemoveItemFromCartCmd:
		return b.handleRemoveItemFromCart(callback.Message)

	default:
		return nil
	}
}

func (b *Bot) handleStartTxt(message *tgbotapi.Message) error {
	buttons := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Добавить товар"),
			tgbotapi.NewKeyboardButton("Продажа"),
			tgbotapi.NewKeyboardButton("Меню"),
		),
	)

	// Настройки клавиатуры (опционально)
	buttons.OneTimeKeyboard = false // Клавиатура остается после нажатия
	buttons.ResizeKeyboard = true   // Клавиатура адаптируется под размер экрана

	msg := tgbotapi.NewMessage(message.Chat.ID, b.messages.Responses.Start)
	msg.ReplyMarkup = buttons
	_, err := b.bot.Send(msg)
	return err
}

func (b *Bot) handleUnknownCmd(message *tgbotapi.Message) error {
	msg := tgbotapi.NewMessage(message.Chat.ID, b.messages.Responses.UnknownCommand)
	_, err := b.bot.Send(msg)
	return err
}

func (b *Bot) handleActionsProductmd(callback *tgbotapi.CallbackQuery) error {
	chatID := callback.Message.Chat.ID
	product := b.tempProduct[chatID]

	buttonDone := b.getProductActionKeyboard(product.ProductID)

	msg := tgbotapi.NewEditMessageReplyMarkup(chatID, b.tempMsgID[chatID], buttonDone)
	_, err := b.bot.Send(msg)
	return err
}

// getFileMeta получает URL и вектор из fileID
func (b *Bot) getFileMeta(fileID string) (*storage.ImageMeta, error) {
	// URL для получения информации о файле
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/getFile?file_id=%s", "8015128447:AAHNjRFRjWP1LQ4nqePLtjJhaoiBo6BFIKA", fileID) //TODO cfg.TelegramToken

	// Делаем запрос к Telegram API
	resp, err := http.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("ошибка при запросе getFile: %w", err)
	}
	defer resp.Body.Close()

	// Декодируем ответ
	var result getFileResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("ошибка декодирования ответа getFile: %w", err)
	}

	// Проверяем успешность запроса
	if !result.Ok || result.Result == nil {
		return nil, fmt.Errorf("не удалось получить информацию о файле")
	}

	imageMeta := &storage.ImageMeta{}
	// Генерируем URL для скачивания файла
	imageMeta.Url = fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", "8015128447:AAHNjRFRjWP1LQ4nqePLtjJhaoiBo6BFIKA", result.Result.FilePath) //TODO cfg.TelegramToken

	imageMeta.Float, err = recognize.ExtractFromModel(imageMeta.Url)
	if err != nil {
		return nil, fmt.Errorf("ошибка при получении вектора файла: %w", err)
	}
	return imageMeta, nil
}

// getFileMeta получает контент из URL
func (b *Bot) getFileContent(url string) ([]byte, error) {

	// Скачиваем файл
	fileResp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("ошибка при загрузке файла: %w", err)
	}
	defer fileResp.Body.Close()

	// Читаем содержимое файла в память
	byte, err := io.ReadAll(fileResp.Body)
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения содержимого файла: %w", err)
	}

	return byte, nil
}

func (b *Bot) getProductsByVector(message *tgbotapi.Message, vector []float64) ([]*storage.Product, error) {
	// Получение всех векторов изображений по имени пользователя
	images, err := b.storage.GetVectorsByUsername(context.Background(), message.Chat.UserName)
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения содержимого файла: %w", err)
	}

	var matchedProducts []*storage.Product

	// Сравнение векторов
	for _, image := range images {
		ok, err := recognize.CompareFeatureVectors(vector, image.Float)
		if err != nil {
			return nil, fmt.Errorf("ошибка при сравнении файлов: %w", err)
		}
		if ok {
			// Получение товара по ID изображения
			product, err := b.storage.GetProductByID(context.Background(), image.ProductID)
			if err != nil || product == nil {
				return nil, fmt.Errorf("ошибка получения товара по ID: %w", err)
			}

			if product.Image == nil {
				product.Image = make([]*storage.ImageMeta, 0)
			}
			product.Image = append(product.Image, image)

			// Добавляем продукт в итоговый список
			matchedProducts = append(matchedProducts, product)
		}
	}

	return matchedProducts, nil
}

func (b *Bot) handleProductList(callback *tgbotapi.CallbackQuery) error {
	chatID := callback.Message.Chat.ID
	userName := callback.From.UserName

	// Получаем список продуктов для пользователя
	products, err := b.storage.GetProducts(context.Background(), userName)
	if err != nil {
		msg := tgbotapi.NewMessage(chatID, "Не удалось получить список продуктов. Попробуйте позже.")
		_, _ = b.bot.Send(msg)
		return fmt.Errorf("ошибка получения продуктов: %w", err)
	}

	if len(products) == 0 {
		msg := tgbotapi.NewMessage(chatID, "У вас пока нет добавленных товаров.")
		_, _ = b.bot.Send(msg)
		return nil
	}

	// Формируем сообщение со списком товаров
	for _, product := range products {
		// Получаем изображения для каждого товара
		photos, err := b.storage.GetPhotosByProductID(context.Background(), product.ProductID)
		if err != nil {
			log.Printf("не удалось получить фото для продукта %d: %v", product.ProductID, err)
			continue // Продолжаем обрабатывать остальные товары
		}

		// Отправляем изображения (если есть)
		for _, photo := range photos {
			photoFile := tgbotapi.NewPhotoUpload(chatID, tgbotapi.FileBytes{
				Name:  fmt.Sprintf("product_%d.jpg", product.ProductID),
				Bytes: photo,
			})
			if _, err := b.bot.Send(photoFile); err != nil {
				log.Printf("не удалось отправить фото: %v", err)
			}
		}

		// Формируем текст с информацией о продукте
		productInfo := fmt.Sprintf(
			"🛒 *%s*\n📦 Наличие: %d\n💰 Цена продажи: %s\n",
			product.Name,
			product.Count,
			product.SellingPrice.StringFixed(2),
		)

		// Отправляем информацию о продукте
		msg := tgbotapi.NewMessage(chatID, productInfo)
		msg.ParseMode = "Markdown"
		if _, err := b.bot.Send(msg); err != nil {
			log.Printf("не удалось отправить информацию о продукте: %v", err)
		}
	}

	return nil
}
func (b *Bot) handleSampleImage(message *tgbotapi.Message) error {
	chatID := message.Chat.ID
	if b.states[chatID] != stateWaitingForPhoto {
		imageMeta, err := b.getFileMeta((*message.Photo)[len(*message.Photo)-1].FileID)
		if err != nil {
			msg := tgbotapi.NewMessage(chatID, "Ошибка обработки фото.")
			_, _ = b.bot.Send(msg)
			return err
		}
		foundProduct, err := b.getProductsByVector(message, imageMeta.Float)
		if err != nil {
			msg := tgbotapi.NewMessage(chatID, "Ошибка обработки фото.")
			_, _ = b.bot.Send(msg)
			return err
		}
		l := len(foundProduct)
		if l > 0 {
			for _, product := range foundProduct {

				// Отправляем изображения (если есть)
				for _, photo := range product.Image {
					photoFile := tgbotapi.NewPhotoUpload(chatID, tgbotapi.FileBytes{
						Name:  fmt.Sprintf("product_%d.jpg", product.ProductID),
						Bytes: photo.Byte,
					})
					if _, err := b.bot.Send(photoFile); err != nil {
						log.Printf("не удалось отправить фото: %v", err)
					}
				}

				// Формируем текст с информацией о продукте
				productInfo := fmt.Sprintf(
					"🛒 *%s*\n📦 Наличие: %d\n💰 Цена продажи: %s\n",
					product.Name,
					product.Count,
					product.SellingPrice.StringFixed(2),
				)

				cart, exists := b.cartItems[chatID]
				if !exists {
					cart = Cart{
						Amount:    decimal.NewFromInt(0),
						CartItems: make(map[uint]CartItem),
					}
				}

				b.cartItems[chatID] = cart

				cartItem, exists := b.cartItems[chatID].CartItems[product.ProductID]
				if exists {
					if cartItem.CountCart == product.Count {
						msg := tgbotapi.NewMessage(chatID, "Товар закончился")
						_, err = b.bot.Send(msg)
						return err
					} else {
						cartItem.CountStore = product.Count
						cartItem.CountCart++
					}
				} else {
					cartItem = CartItem{
						MsgID:      0,
						CountStore: product.Count,
						CountCart:  0,
						Price:      product.SellingPrice,
					}
				}

				b.cartItems[chatID].CartItems[product.ProductID] = cartItem

				actionsProductKeyboard := b.getProductActionKeyboard(product.ProductID)
				addProductToCartKeyboard := b.getAddItemToCartKeyboard(product.ProductID)

				mergedKeyboard := tgbotapi.NewInlineKeyboardMarkup(
					append(actionsProductKeyboard.InlineKeyboard,
						addProductToCartKeyboard.InlineKeyboard...,
					)...,
				)

				msg := tgbotapi.NewMessage(chatID, productInfo)
				msg.ParseMode = "Markdown"
				msg.ReplyMarkup = mergedKeyboard

				_, err := b.bot.Send(msg)
				if err != nil {
					log.Printf("не удалось отправить информацию о продукте: %v", err)
					return err
				}

			}
			return err
		}

		b.states[chatID] = stateWaitingForName
		msg := tgbotapi.NewMessage(chatID, "Введите название товара:")
		_, err = b.bot.Send(msg)
		return err
	}
	return nil
}
