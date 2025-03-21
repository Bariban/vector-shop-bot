package telegram

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"

	hnsw "github.com/Bariban/go-hnsw"
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

	chatID := message.Chat.ID
	state := b.states[chatID]
	switch message.Text {
	case StartCmd:
		return b.procStartTxt(message)
	case AddProductText:
		return b.procAddProductCmd(message)
	case PaymentText:
		return b.procPhoneQuestion(message)
	case CancelOperationsText:
		return b.procCancelOperations(message)
	case MenuText:
		return b.procGetMenu(message)
	default:
		if addProductStates[state] {
			return b.procAddProductCmd(message)
		}

		if editProductStates[state] {
			return b.procConfirmEdit(message)
		}

		if message.Photo != nil {
			return b.procSampleImage(message)
		}

		if state == stateEditCountItemInCart {
			return b.procEditCountItemInCart(message)
		}

		if state == stateDiscountProductInCart {
			return b.procDiscoutItemInCart(message)
		}

		if state == stateEditShopName {
			return b.procEditShopName(message)
		}

		if state == stateWhatingClientPhone {
			return b.procSaveClientPhone(message)
		}

		//Старт с реферальной ссылкой
		if strings.HasPrefix(message.Text, StartWithPayloadCmd) {
			return b.procStartArgCmd(message)
		}

		return b.procUnknownCmd(message)
	}

}

func (b *Bot) handleCallbackCommand(callback *tgbotapi.CallbackQuery) error {
	chatID := callback.Message.Chat.ID
	action := callback.Data
	// Регулярное выражение для поиска числа в конце строки
	re := regexp.MustCompile(`\d+$`)
	match := re.FindString(action)
	intMatch, _ := strconv.Atoi(match)
	// Преобразуем найденное число в uint
	if match != "" {
		// Определяем действие (до числа)
		action = action[:len(action)-len(match)-1]
		if action != EditUserCmd {
			product := &storage.Product{
				ProductID: uint(intMatch),
				UserID:    uint(callback.Message.Chat.ID),
				Image:     []*storage.ImageMeta{{}},
			}
			b.tempProduct[chatID] = product

			if _, ok := b.selectedParams[callback.Message.Chat.ID]; !ok {
				b.selectedParams[callback.Message.Chat.ID] = make(map[string]bool)
			}
			//b.selectedParams[callback.Message.Chat.ID][action] = true
		}
	}

	switch action {
	case AddProductCmd:
		return b.procAddProductCmd(callback.Message)
	case ListCmd:
		return b.procProductList(callback)
	case EditProductCmd:
		return b.procEditProductCmd(callback, action)
	case ConfirmDelProductCmd:
		return b.procConfirmDeleteProductCmd(callback)
	case ConfirmDelImageCmd:
		return b.procConfirmDeleteImageCmd(callback, uint(intMatch))
	case DelImageCmd:
		return b.procDeleteImageCmd(callback, uint(intMatch))
	case DelProductCmd:
		return b.procDeleteProductCmd(callback)
	case ActionsProductCmd:
		return b.procActionsProductCmd(callback)
	case ActionsImagesCmd:
		return b.procActionsImagesCmd(callback, uint(intMatch))
	case FinishEditImagesCmd:
		return b.procFinisEditImages(callback.Message)

	case EditProductNameCmd:
		return b.procEditProductCmd(callback, action)
	case EditProductCountCmd:
		return b.procEditProductCmd(callback, action)
	case EditProductPurchaseCmd:
		return b.procEditProductCmd(callback, action)
	case EditProductSellingCmd:
		return b.procEditProductCmd(callback, action)
	case EditProductDescriptionCmd:
		return b.procEditProductCmd(callback, action)
	case EditProductImagesCmd:
		return b.procEditProductCmd(callback, action)
	case ConfirmEditProductCmd:
		return b.procConfirmEdit(callback.Message)
	case AddItemToCartCmd:

		return b.procAddItemToCart(callback)
	case ReduceItemInCartCmd:
		return b.procReduceItemInCart(callback)
	case EditCountItemInCartCmd:
		return b.procEditCountItemInCart(callback.Message)
	case DiscountItemInCartCmd:
		return b.procDiscoutItemInCart(callback.Message)
	case RemoveItemFromCartCmd:
		return b.procRemoveItemFromCart(callback.Message)
	case PayTypeCashCmd:
		return b.procAddOrder(callback, action)
	case ShopKeyboardCmd:
		return b.procShopKeyboard(callback.Message)
	case EditShopKeyboardCmd:
		return b.procEditShopKeyboard(callback.Message)
	case MenuCmd:
		return b.procGetMenu(callback.Message)
	case EditShopNameCmd:
		return b.procEditShopName(callback.Message)
	case InviteUsersCmd:
		return b.procInviteUsersKeyboard(callback.Message)
	case InviteEmployeeCmd:
		return b.procInviteEmployee(callback.Message)
	case InviteClientCmd:
		return b.procInviteClient(callback.Message)
	case UserListCmd:
		return b.procGetShopUsers(callback.Message)
	case EditUserCmd:
		return b.procEditShopUser(callback.Message, uint(intMatch))
	case GrantRoleCustomerCmd:
		return b.procGrantRoleCustomer(callback.Message, uint(intMatch))
	case GrantRoleEmployeeCmd:
		return b.procGrantRoleEmployee(callback.Message, uint(intMatch))
	case GrantRoleClientCmd:
		return b.procGrantRoleClient(callback.Message, uint(intMatch))
	case SavePhoneNumberCmd:
		return b.procPhoneRequest(callback.Message)
	case DontSavePhoneNumberCmd:
		return b.procPhoneRequestCansel(callback.Message)

	default:
		return nil
	}
}

func (b *Bot) procStartTxt(message *tgbotapi.Message) error {
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

func (b *Bot) procStartArgCmd(message *tgbotapi.Message) error {

	chatID := message.Chat.ID
	args := message.CommandArguments() // Аргументы после /start

	// Если аргументы переданы
	if args != "" {
		// Раскодируем payload
		payload, err := base64.URLEncoding.DecodeString(args)
		if err != nil {
			b.bot.Send(tgbotapi.NewMessage(chatID, "Неверная ссылка!"))
			return err
		}

		// Разбираем payload (например, "123|RoleEmployee")
		parts := strings.Split(string(payload), "|")
		if len(parts) != 2 {
			b.bot.Send(tgbotapi.NewMessage(chatID, "Ошибка в ссылке приглашения!"))
			return fmt.Errorf("invalid payload format")
		}

		shopID, err := strconv.Atoi(parts[0])
		if err != nil {
			b.bot.Send(tgbotapi.NewMessage(chatID, "Некорректный идентификатор магазина!"))
			return err
		}

		role := parts[1]
		if role != RoleCustomer && role != RoleEmployee && role != RoleClient {
			b.bot.Send(tgbotapi.NewMessage(chatID, "Некорректная роль!"))
			return fmt.Errorf("invalid role")
		}

		shopName, err := b.storage.GetShopName(context.Background(), shopID)
		if err != nil {
			b.bot.Send(tgbotapi.NewMessage(chatID, "Не удалось найти магазин!"))
			return err
		}

		// Создаём или обновляем пользователя
		user := &storage.User{
			FirstName: message.Chat.FirstName,
			LastName:  message.Chat.LastName,
			UserName:  message.From.UserName,
			UserID:    uint(message.Chat.ID),
			ShopID:    uint(shopID),
			Role:      role,
			ShopName:  shopName,
		}

		// Сохраняем пользователя в базе данных
		err = b.storage.InitUser(user)
		if err != nil {
			b.bot.Send(tgbotapi.NewMessage(chatID, "Не удалось зарегистрировать пользователя."))
			return err
		}

		b.user[chatID] = user

		// Ответ пользователю
		b.bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("Добро пожаловать в магазин %s!", shopName)))
		if role == RoleEmployee {
			b.bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("Вам присвоена роль сотрудника")))
		}

	} else {
		// Стандартное приветствие для обычного /start без параметров
		b.bot.Send(tgbotapi.NewMessage(chatID, "Добро пожаловать в нашего бота!"))
	}
	return nil
}

func (b *Bot) procUnknownCmd(message *tgbotapi.Message) error {
	msg := tgbotapi.NewMessage(message.Chat.ID, b.messages.Responses.UnknownCommand)
	_, err := b.bot.Send(msg)
	return err
}

// procActionsProductCmd возвращаем исходную клавиатуру для товара
func (b *Bot) procActionsProductCmd(callback *tgbotapi.CallbackQuery) error {
	chatID := callback.Message.Chat.ID
	product := b.tempProduct[chatID]

	actionsProductKeyboard := b.getProductActionKeyboard(product.ProductID)
	addProductToCartKeyboard := b.getAddItemToCartKeyboard(product.ProductID)
	mergedKeyboard := tgbotapi.NewInlineKeyboardMarkup(
		append(actionsProductKeyboard.InlineKeyboard,
			addProductToCartKeyboard.InlineKeyboard...,
		)...,
	)

	msg := tgbotapi.NewEditMessageReplyMarkup(chatID, callback.Message.MessageID, mergedKeyboard)
	_, err := b.bot.Send(msg)
	return err
}

// procActionsImagesCmd возвращаем исходную клавиатуру для изображения
func (b *Bot) procActionsImagesCmd(callback *tgbotapi.CallbackQuery, imageID uint) error {
	chatID := callback.Message.Chat.ID

	msg := tgbotapi.NewEditMessageReplyMarkup(chatID, callback.Message.MessageID, getImageActionKeyboard(imageID))
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

	imageMeta.Float, imageMeta.BarCode, err = recognize.ExtractFromModel(imageMeta.Url)
	if err != nil {
		return nil, fmt.Errorf("ошибка при получении вектора файла: %w", err)
	}
	return imageMeta, nil
}

// getFileContent получает контент из URL
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
		neighborsQueue := b.index[b.user[message.Chat.ID].ShopID].Search(searchVector, efSearch, k)

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

func (b *Bot) procProductList(callback *tgbotapi.CallbackQuery) error {
	chatID := callback.Message.Chat.ID

	// Получаем список продуктов для пользователя
	products, err := b.storage.GetProducts(context.Background(), b.user[chatID].ShopID)
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

		// Отправляем информацию о продукте
		msg := tgbotapi.NewMessage(chatID, productInfo)
		msg.ParseMode = "Markdown"
		if _, err := b.bot.Send(msg); err != nil {
			log.Printf("не удалось отправить информацию о продукте: %v", err)
		}
	}

	return nil
}

func (b *Bot) procSampleImage(message *tgbotapi.Message) error {
	chatID := message.Chat.ID
	if b.states[chatID] != stateWaitingForPhoto {
		imageMeta, err := b.getFileMeta((*message.Photo)[len(*message.Photo)-1].FileID)
		if err != nil {
			msg := tgbotapi.NewMessage(chatID, "Ошибка обработки фото.")
			_, _ = b.bot.Send(msg)
			return err
		}

		var foundProduct []*storage.Product
		if imageMeta.BarCode != "" {
			foundProduct, err = b.storage.GetProductsByBarCode(context.Background(), b.user[chatID].ShopID, imageMeta.BarCode)
		} else {
			product := &storage.Product{
				Image: []*storage.ImageMeta{imageMeta},
			}

			foundProduct, err = b.storage.GetNearestNeighbors(context.Background(), product.Image, 3)
			if err != nil {
				msg := tgbotapi.NewMessage(chatID, "Ошибка обработки фото.")
				_, _ = b.bot.Send(msg)
				return err
			}
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
						Bytes: image.Byte,
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
				} else {
					cartItem = CartItem{
						MsgID:     cartItem.MsgID,
						CountCart: cartItem.CountCart,
						Price:     cartItem.Price,

						CountStore: product.Count,
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

func (b *Bot) procCancelOperations(message *tgbotapi.Message) error {
	chatID := message.Chat.ID

	delete(b.states, chatID)
	delete(b.tempProduct, chatID)
	delete(b.cartItems, chatID)
	delete(b.selectedParams, chatID)
	delete(b.tempMsgID, chatID)
	err := b.procStartTxt(message)
	return err
}
