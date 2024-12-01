package telegram

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/shopspring/decimal"
)

// getAddProductToCartKeyboard возвращает клавиатуру с добавлением товара в корзину
func (b *Bot) getAddItemToCartKeyboard(productID uint) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Добавить в корзину ➕", fmt.Sprintf("%s_%d", AddItemToCartCmd, productID)),
		),
	)
}

// getProductActionKeyboard возвращает клавиатуру с действиями над товаром
func (b *Bot) getCountItemInCartKeyboard(chatID int64, productID uint) tgbotapi.InlineKeyboardMarkup {

	countItem := int(b.cartItems[chatID].CartItems[productID].CountCart)

	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("  ➖  ", fmt.Sprintf("%s_%d", ReduceItemInCartCmd, productID)),
			tgbotapi.NewInlineKeyboardButtonData(strconv.Itoa(countItem), fmt.Sprintf("%s_%d", EditCountItemInCartCmd, productID)),
			tgbotapi.NewInlineKeyboardButtonData("  ➕  ", fmt.Sprintf("%s_%d", AddItemToCartCmd, productID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Убрать из корзины", fmt.Sprintf("%s_%d", RemoveItemFromCartCmd, productID)),
		),
	)
}

func (b *Bot) handleAddItemToCart(callback *tgbotapi.CallbackQuery) error {
	chatID := callback.Message.Chat.ID
	messageID := callback.Message.MessageID
	product := b.tempProduct[chatID]

	cart, exists := b.cartItems[chatID]
	if !exists {
		b.bot.Send(tgbotapi.NewMessage(chatID, "Корзина не найдена:"))
		return nil
	}
	cartItem, exists := cart.CartItems[product.ProductID]

	if !exists {
		b.states[chatID] = stateEditCountItemInCart
		b.bot.Send(tgbotapi.NewMessage(chatID, "Товар не найден:"))
		return nil
	}


	var str string
	if cartItem.CountCart < cartItem.CountStore {
		cartItem.CountCart++
		str = "+" + cartItem.Price.String()
		cart.Amount = cart.Amount.Add(cartItem.Price)
		b.cartItems[chatID] = cart
	}
	

	cartItem.MsgID = messageID
	b.cartItems[chatID].CartItems[product.ProductID] = cartItem

	if cartItem.CountCart == 1{			
		b.cleanUpMessages(chatID, messageID)
		b.tempMsgID[chatID] = messageID
	}

	b.getSellingKeyboard(chatID, str)

	// Обновляем клавиатуру
	CountItemInCartKeyboard := b.getCountItemInCartKeyboard(chatID, product.ProductID)
	msg := tgbotapi.NewEditMessageReplyMarkup(chatID, cartItem.MsgID, CountItemInCartKeyboard)
	_, err := b.bot.Send(msg)

	return err
}

func (b *Bot) handleReduceItemInCart(callback *tgbotapi.CallbackQuery) error {
	chatID := callback.Message.Chat.ID
	product := b.tempProduct[chatID]

	cart, exists := b.cartItems[chatID]
	if !exists {
		b.bot.Send(tgbotapi.NewMessage(chatID, "Корзина не найдена:"))
		return nil
	}
	cartItem, exists := cart.CartItems[product.ProductID]

	if !exists {
		b.states[chatID] = stateEditCountItemInCart
		b.bot.Send(tgbotapi.NewMessage(chatID, "Товар не найден:"))
		return nil
	}
	var str string
	if cartItem.CountCart > 1 {
		cartItem.CountCart--
		str = "-" + cartItem.Price.String()
		cart.Amount = cart.Amount.Sub(cartItem.Price)
		b.cartItems[chatID] = cart
	}

	if cartItem.MsgID == 0 {
		cartItem.MsgID = callback.Message.MessageID
	}

	b.cartItems[chatID].CartItems[product.ProductID] = cartItem

	b.getSellingKeyboard(chatID, str)

	// Обновляем клавиатуру
	CountItemInCartKeyboard := b.getCountItemInCartKeyboard(chatID, product.ProductID)
	msg := tgbotapi.NewEditMessageReplyMarkup(chatID, cartItem.MsgID, CountItemInCartKeyboard)
	_, err := b.bot.Send(msg)

	return err
}

func (b *Bot) handleEditCountItemInCart(message *tgbotapi.Message) error {
	chatID := message.Chat.ID
	product := b.tempProduct[chatID]
	state := b.states[chatID]

	cart, exists := b.cartItems[chatID]
	if !exists {
		b.bot.Send(tgbotapi.NewMessage(chatID, "Корзина не найдена:"))
		return nil
	}

	cartItem, exists := cart.CartItems[product.ProductID]

	if !exists {
		b.states[chatID] = stateEditCountItemInCart
		b.bot.Send(tgbotapi.NewMessage(chatID, "Товар не найден:"))
		return nil
	}

	if state != stateEditCountItemInCart {
		b.states[chatID] = stateEditCountItemInCart
		b.bot.Send(tgbotapi.NewMessage(chatID, "Введите количество:"))
		return nil
	}

	input := strings.TrimSpace(message.Text)
	if len(input) == 0 {
		b.bot.Send(tgbotapi.NewMessage(chatID, "Введите корректное значение:"))
		return nil
	}

	// Проверяем, есть ли знак перед числом
	sign := ""
	if strings.HasPrefix(input, "+") || strings.HasPrefix(input, "-") || strings.HasPrefix(input, "*") || strings.HasPrefix(input, "/") {
		sign = input[:1]
		input = input[1:]
	}

	// Преобразуем оставшуюся часть в число
	count, err := strconv.Atoi(input)
	if err != nil || count < 0 {
		b.bot.Send(tgbotapi.NewMessage(chatID, "Введите корректное положительное число:"))
		return nil
	}

	// Обрабатываем математическую операцию
	newCount := int(cartItem.CountCart) // Текущее количество товара в корзине

	itemPrice := cartItem.Price // Цена за единицу товара

	var str string
	var d decimal.Decimal
	switch sign {
	case "+":
		newCount += count
		d = itemPrice.Mul(decimal.NewFromInt(int64(count)))
		str = "+" + d.String()
		cart.Amount = cart.Amount.Add(d)
		b.cartItems[chatID] = cart

	case "-":
		newCount -= count
		if newCount < 0 {
			b.bot.Send(tgbotapi.NewMessage(chatID, "Количество не может быть отрицательным."))
			return nil
		}
		d = itemPrice.Mul(decimal.NewFromInt(int64(count)))
		str = "-" + d.String()
		b.cartItems[chatID].Amount.Sub(d)
		cart.Amount = cart.Amount.Sub(cartItem.Price)
		b.cartItems[chatID] = cart

	default:
		delta := count - int(cartItem.CountCart)

		itemPriceChange := cartItem.Price.Mul(decimal.NewFromInt(int64(abs(delta))))

		if delta > 0 {
			str = "+" + itemPriceChange.String()
			cart.Amount = cart.Amount.Add(itemPriceChange)
		} else if delta < 0 {
			str = "-" + itemPriceChange.String()
			cart.Amount = cart.Amount.Sub(itemPriceChange)
		}

		cartItem.CountCart = uint(count)
		b.cartItems[chatID] = cart
		newCount = count
	}

	if newCount < 0 {
		b.bot.Send(tgbotapi.NewMessage(chatID, "Результат не может быть отрицательным."))
		return nil
	}
	if uint(newCount) > cartItem.CountStore {
		b.bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("Превышен остаток: %d", cartItem.CountStore)))
		return nil
	}

	// Обновляем количество
	cartItem.CountCart = uint(newCount)
	if cartItem.MsgID == 0 {
		cartItem.MsgID = message.MessageID
	}
	b.cartItems[chatID].CartItems[product.ProductID] = cartItem

	b.getSellingKeyboard(chatID, str)

	CountItemInCartKeyboard := b.getCountItemInCartKeyboard(chatID, product.ProductID)
	msg := tgbotapi.NewEditMessageReplyMarkup(chatID, cartItem.MsgID, CountItemInCartKeyboard)
	_, err = b.bot.Send(msg)

	delete(b.states, chatID)
	return err
}

func (b *Bot) handleRemoveItemFromCart(message *tgbotapi.Message) error {
	chatID := message.Chat.ID
	product := b.tempProduct[chatID]

	cart, exists := b.cartItems[chatID]
	if !exists {
		b.bot.Send(tgbotapi.NewMessage(chatID, "Корзина не найдена:"))
		return nil
	}

	var str string
	cartItem, exists := cart.CartItems[product.ProductID]
	if exists {
		d := decimal.NewFromInt(int64(cartItem.CountCart)).Mul(cartItem.Price)
		cartItem.CountCart = 0
		str = "-"+ d.String()
		cart.Amount = cart.Amount.Sub(d)
		b.cartItems[chatID] = cart
	}

	b.cartItems[chatID].CartItems[product.ProductID] = cartItem

	b.getSellingKeyboard(chatID, str)

	// Обновляем клавиатуру

	CountItemInCartKeyboard := b.getAddItemToCartKeyboard(product.ProductID)
	msg := tgbotapi.NewEditMessageReplyMarkup(chatID, cartItem.MsgID, CountItemInCartKeyboard)
	_, err := b.bot.Send(msg)

	return err
}

func (b *Bot) cleanUpMessages(chatID int64, lastMsgID int) {
	exceptMsgIDs := make(map[int]bool)
	tmpMsg :=b.tempMsgID[chatID]
	if tmpMsg == 0{
		return
	}

	for _, cartItem := range b.cartItems[chatID].CartItems {
		if cartItem.MsgID != 0 {
			exceptMsgIDs[cartItem.MsgID] = true
			exceptMsgIDs[cartItem.MsgID-1] = true
		}
	}

	// Получаем диапазон сообщений
	for i := lastMsgID; i > tmpMsg; i-- {
		// Пропускаем сообщения, которые нужно оставить
		if exceptMsgIDs[i] {
			continue
		}

		// Удаляем текстовое сообщение
		_, err := b.bot.DeleteMessage(tgbotapi.DeleteMessageConfig{
			ChatID:    chatID,
			MessageID: i,
		})
		if err != nil {
			log.Printf("Не удалось удалить сообщение %d: %v", i, err)
			continue
		}
	}

}

func (b *Bot) getSellingKeyboard(chatID int64, str string) (int, error) {
	// Получаем текущую сумму корзины
	amount := b.cartItems[chatID].Amount

	// Создаём клавиатуру
	buttons := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(fmt.Sprintf("🛍 %s", amount.StringFixed(2))), // Форматируем сумму
			tgbotapi.NewKeyboardButton("Оплата"),
		),
	)

	// Настройки клавиатуры
	buttons.OneTimeKeyboard = false
	buttons.ResizeKeyboard = true

	// Отправляем обновлённую клавиатуру
	msg := tgbotapi.NewMessage(chatID, str) // Отправляем пустую строку вместо нового текста
	msg.ReplyMarkup = buttons

	messege, err := b.bot.Send(msg)
	return messege.MessageID, err
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}