package telegram

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"sort"
	"strconv"

	"github.com/Bariban/vector-shop-bot/pkg/recognize"
	"github.com/Bariban/vector-shop-bot/pkg/storage"
	hnsw "github.com/Bithack/go-hnsw"
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

	chatID := message.Chat.ID
	state := b.states[chatID]
	switch message.Text {
	case StartCmd:
		return b.handleStartTxt(message)
	case AddProductText:
		return b.handleAddProductCmd(message)
	case PaymentText:
		return b.handleSelectPayType(message)
	case CancelOperationsText:
		return b.handleCancelOperations(message)
	// case MenuText:
	// 	return b.handleMenu(message)
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

		if state == stateEditCountItemInCart {
			return b.handleEditCountItemInCart(message)
		}

		if state == stateDiscountProductInCart {
			return b.handleDiscoutItemInCart(message)
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
		return b.handleAddProductCmd(callback.Message)
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
	case DiscountItemInCartCmd:
		return b.handleDiscoutItemInCart(callback.Message)
	case RemoveItemFromCartCmd:
		return b.handleRemoveItemFromCart(callback.Message)
	case PayTypeCashCmd:
		return b.handleAddOrder(callback, action)

	default:
		return nil
	}
}

func (b *Bot) handleStartTxt(message *tgbotapi.Message) error {
	buttons := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Добавить товар"),
			tgbotapi.NewKeyboardButton("Отмена"),
			tgbotapi.NewKeyboardButton("Меню"),
		),
	)

	// Настройки клавиатуры (опционально)
	buttons.OneTimeKeyboard = false // Клавиатура остается после нажатия
	buttons.ResizeKeyboard = true   // Клавиатура адаптируется под размер экрана

	var str string
	if message.Text == StartCmd {
		str = b.messages.Responses.Start
	} else {
		str = b.messages.Responses.KeyboardUpdated
	}
	msg := tgbotapi.NewMessage(message.Chat.ID, str)
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

func (b *Bot) getProductsByVector(message *tgbotapi.Message, images []*storage.ImageMeta) ([]*storage.Product, error) {
	// Карта для уникальных товаров (по productID) и imageID, чтобы избегать дубликатов
	uniqueProducts := make(map[uint]*storage.Product)
	distanceImages := make(map[uint]float32)

	type kv struct {
		Key   uint
		Value float32
	}

	var sortedDistanceImages []kv

	// Параметры поиска в индексе
	efSearch := 100 // Параметр точности поиска
	k := 6          // Количество ближайших соседей

	for _, inputImage := range images {
		// Проверяем, что вектор не пустой
		if inputImage.Float == nil {
			continue
		}

		// Преобразуем вектор изображения в формат hnsw.Point
		searchVector := hnsw.Point(inputImage.Float)

		// Ищем ближайших соседей в индексе
		neighborsQueue := b.index[message.Chat.ID].Search(searchVector, efSearch, k)

		// Извлекаем соседей из очереди
		for neighborsQueue.Len() > 0 {
			neighbor := neighborsQueue.Pop()

			neighborID := uint(neighbor.ID)

			if neighborID == 0 {
				continue
			}

			findVector, err := b.storage.GetVectorByImageID(context.Background(), neighborID)
			if err != nil {
				return nil, fmt.Errorf("ошибка при поиске вектора: %w", err)
			}
			// Сравниваем вектор текущего объекта с вектором соседа
			distance, err := recognize.CompareFeatureVectors(inputImage.Float, findVector)
			if err != nil {
				return nil, fmt.Errorf("ошибка при сравнении векторов: %w", err)
			}

			if distance <= 35 {
				distanceImages[neighborID] = distance
			}
		}
	}

	for k, v := range distanceImages {
		sortedDistanceImages = append(sortedDistanceImages, kv{Key: k, Value: v})
	}

	// Сортируем срез по значениям (Value)
	sort.Slice(sortedDistanceImages, func(i, j int) bool {
		return sortedDistanceImages[i].Value < sortedDistanceImages[j].Value
	})

	// Итерация по отсортированному срезу
	for i, item := range sortedDistanceImages {

		product, err := b.storage.GetProductByImageID(context.Background(), item.Key)
		if err != nil {
			return nil, fmt.Errorf("ошибка получения товара по imageID %d: %w", item.Key, err)
		}
		if product == nil {
			continue
		}

		// Добавляем товар в карту уникальных товаров, если его там ещё нет
		if _, exists := uniqueProducts[product.ProductID]; !exists {
			uniqueProducts[product.ProductID] = product
		}
		if i == 1 {
			break
		}
	}

	// Преобразуем карту уникальных товаров в срез
	matchedProducts := make([]*storage.Product, 0, len(uniqueProducts))
	for _, product := range uniqueProducts {
		matchedProducts = append(matchedProducts, product)
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

		product := &storage.Product{
			Image: []*storage.ImageMeta{imageMeta},
		}

		foundProduct, err := b.getProductsByVector(message, product.Image)
		if err != nil {
			msg := tgbotapi.NewMessage(chatID, "Ошибка обработки фото.")
			_, _ = b.bot.Send(msg)
			return err
		}
		l := len(foundProduct)
		if l > 0 {
			for _, product := range foundProduct {

				// Получаем изображения для каждого товара
				images, err := b.storage.GetPhotosByProductID(context.Background(), product.ProductID)
				if err != nil {
					log.Printf("не удалось получить фото для продукта %d: %v", product.ProductID, err)
				}

				// Отправляем изображения (если есть)
				for _, image := range images {
					photoFile := tgbotapi.NewPhotoUpload(chatID, tgbotapi.FileBytes{
						Name:  fmt.Sprintf("product_%d.jpg", product.ProductID),
						Bytes: image,
					})
					if _, err := b.bot.Send(photoFile); err != nil {
						log.Printf("не удалось отправить фото: %v", err)
					}
					break
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
				if !exists {
					cartItem = CartItem{
						MsgID:      0,
						CountStore: product.Count,
						CountCart:  0,
						Price:      product.SellingPrice,
						PriceStore: product.SellingPrice,
					}
				}

				b.cartItems[chatID].CartItems[product.ProductID] = cartItem
				var mergedKeyboard tgbotapi.InlineKeyboardMarkup
				if cartItem.CountCart > 0 {
					CountItemInCartKeyboard := b.getCountItemInCartKeyboard(chatID, product.ProductID)
					mergedKeyboard = CountItemInCartKeyboard
				} else {
					actionsProductKeyboard := b.getProductActionKeyboard(product.ProductID)
					addProductToCartKeyboard := b.getAddItemToCartKeyboard(product.ProductID)
					mergedKeyboard = tgbotapi.NewInlineKeyboardMarkup(
						append(actionsProductKeyboard.InlineKeyboard,
							addProductToCartKeyboard.InlineKeyboard...,
						)...,
					)
				}

				msg := tgbotapi.NewMessage(chatID, productInfo)
				msg.ParseMode = "Markdown"
				msg.ReplyMarkup = mergedKeyboard

				_, err = b.bot.Send(msg)
				if err != nil {
					log.Printf("не удалось отправить информацию о продукте: %v", err)
					return err
				}

			}
			return err
		}

		msg := tgbotapi.NewMessage(chatID, "Позиция не найдена, попробуйте снова")
		_, err = b.bot.Send(msg)
		return err
	}
	return nil
}

func (b *Bot) handleCancelOperations(message *tgbotapi.Message) error {
	chatID := message.Chat.ID

	delete(b.states, chatID)
	delete(b.tempProduct, chatID)
	delete(b.cartItems, chatID)
	delete(b.selectedParams, chatID)
	delete(b.tempMsgID, chatID)
	err := b.handleStartTxt(message)
	return err
}
