package telegram

import (
	"context"
	"fmt"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/shopspring/decimal"
)

func (b *Bot) handleEditProductCmd(callback *tgbotapi.CallbackQuery) error {
	chatID := callback.Message.Chat.ID

	// Инициализируем временные данные продукта и выбранные параметры
	if b.selectedParams[chatID] == nil {
		b.selectedParams[chatID] = make(map[string]bool)
	}
	b.tempMsgID[chatID] = callback.Message.MessageID
	// Обновляем клавиатуру с галочками
	editProductKeyboard := b.generateEditProductKeyboard(chatID)
	msg := tgbotapi.NewEditMessageReplyMarkup(chatID, callback.Message.MessageID, editProductKeyboard)
	_, err := b.bot.Send(msg)
	return err
}

func (b *Bot) handleConfirmEdit(message *tgbotapi.Message) error {
	chatID := message.Chat.ID
	selectedParams := b.selectedParams[chatID]
	text := message.Text
	state := b.states[chatID]

	paramOrder := []string{
		EditProductNameCmd,
		EditProductCountCmd,
		EditProductPurchaseCmd,
		EditProductSellingCmd,
	}

	product, exists := b.tempProduct[chatID]
	if !exists {
		return fmt.Errorf("продукт не найден в tempProduct")
	}

	// Проверяем по порядку, какие параметры выбраны, и запрашиваем у пользователя новые значения
	for _, param := range paramOrder {
		selected, exists := selectedParams[param]
		if !exists || !selected {
			continue // Пропускаем, если параметр не выбран
		}

		switch param {
		case EditProductNameCmd:
			if state != stateWaitingForEditName {
				b.states[chatID] = stateWaitingForEditName
				b.bot.Send(tgbotapi.NewMessage(chatID, "Введите новое название:"))
				return nil
			} else {
				product.Name = text
				b.storage.UpdateProductField(context.Background(), product.ProductID, "name", product.Name)
				b.bot.Send(tgbotapi.NewMessage(chatID, "Название успешно обновлено!"))
				b.selectedParams[chatID][EditProductNameCmd] = false
			}
			return b.handleConfirmEdit(message)

		case EditProductCountCmd:
			if state != stateWaitingForEditCount {
				b.states[chatID] = stateWaitingForEditCount
				b.bot.Send(tgbotapi.NewMessage(chatID, "Введите новое количество:"))
				return nil
			} else {
				count, err := strconv.Atoi(text)
				if err != nil {
					return fmt.Errorf("ошибка ввода количества: %w", err)
				}
				product.Count = uint(count)
				b.storage.UpdateProductField(context.Background(), product.ProductID, "count", count)
				b.bot.Send(tgbotapi.NewMessage(chatID, "Количество успешно обновлено!"))
				b.selectedParams[chatID][EditProductCountCmd] = false
			}
			return b.handleConfirmEdit(message)

		case EditProductPurchaseCmd:
			if state != stateWaitingForEditPurchasePrice {
				b.states[chatID] = stateWaitingForEditPurchasePrice
				b.bot.Send(tgbotapi.NewMessage(chatID, "Введите новую цену закупа:"))
				return nil
			} else {
				price, err := decimal.NewFromString(text)
				if err != nil {
					return fmt.Errorf("ошибка ввода цены закупа: %w", err)
				}
				product.PurchasePrice = price
				b.storage.UpdateProductField(context.Background(), product.ProductID, "purchase_price", price)
				b.bot.Send(tgbotapi.NewMessage(chatID, "Цена закупа успешно обновлена!"))
				b.selectedParams[chatID][EditProductPurchaseCmd] = false
			}
			return b.handleConfirmEdit(message)
		case EditProductSellingCmd:
			if state != stateWaitingForEditSellingPrice {
				b.states[chatID] = stateWaitingForEditSellingPrice
				b.bot.Send(tgbotapi.NewMessage(chatID, "Введите новую цену продажи:"))
				return nil
			} else {
				price, err := decimal.NewFromString(text)
				if err != nil {
					return fmt.Errorf("ошибка ввода цены продажи: %w", err)
				}
				product.SellingPrice = price
				b.storage.UpdateProductField(context.Background(), product.ProductID, "selling_price", price)
				b.bot.Send(tgbotapi.NewMessage(chatID, "Цена продажи успешно обновлена!"))
				b.selectedParams[chatID][EditProductSellingCmd] = false
			}
			return b.handleConfirmEdit(message)
		}
	}

	if len(selectedParams) == 0 {
		b.bot.Send(tgbotapi.NewMessage(chatID, "Выберите изменяемые параметры"))
		return nil
	}
	// Завершаем редактирование и очищаем временные данные
	b.bot.Send(tgbotapi.NewMessage(chatID, "Товар отредактирован!"))

	buttonDone := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Отредактировано", "done"),
		),
	)

	msg := tgbotapi.NewEditMessageReplyMarkup(chatID, b.tempMsgID[chatID], buttonDone)
	_, err := b.bot.Send(msg)
	delete(b.states, chatID)
	delete(b.tempProduct, chatID)
	delete(b.selectedParams, chatID)
	delete(b.tempMsgID, chatID)
	return err

}

func (b *Bot) generateToggleButton(label, action string, chatID int64) tgbotapi.InlineKeyboardButton {
	selected := ""
	if b.selectedParams[chatID][action] {
		selected = " ✅"
	}
	return tgbotapi.NewInlineKeyboardButtonData(label+selected, action)
}

func (b *Bot) generateEditProductKeyboard(chatID int64) tgbotapi.InlineKeyboardMarkup {
	// Создаём кнопки с учётом текущего состояния
	buttons := []tgbotapi.InlineKeyboardButton{
		b.generateToggleButton("Название", EditProductNameCmd, chatID),
		b.generateToggleButton("Количество", EditProductCountCmd, chatID),
		b.generateToggleButton("Цена закупа", EditProductPurchaseCmd, chatID),
		b.generateToggleButton("Цена продажи", EditProductSellingCmd, chatID),
	}

	rows := [][]tgbotapi.InlineKeyboardButton{
		buttons[:2],
		buttons[2:],
		{tgbotapi.NewInlineKeyboardButtonData("Продолжить", ConfirmEditProductCmd)},
	}
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

// getProductActionKeyboard возвращает клавиатуру с действиями над товаром
func (b *Bot) getProductActionKeyboard(productID uint) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✏️ Изменить ", fmt.Sprintf("%s_%d", EditProductCmd, productID)),
			tgbotapi.NewInlineKeyboardButtonData("Удалить ❓", fmt.Sprintf("%s_%d", ConfirmDelProductCmd, productID)),
		),
	)
}

func (b *Bot) handleConfirmDeleteProductCmd(callback *tgbotapi.CallbackQuery) error {
	chatID := callback.Message.Chat.ID
	product := b.tempProduct[chatID]

	buttonDone := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Удалить", fmt.Sprintf("%s_%d", DelProductCmd, product.ProductID)),
			tgbotapi.NewInlineKeyboardButtonData("Нет", fmt.Sprintf("%s_%d", ActionsProductCmd, product.ProductID)),
		),
	)

	msg := tgbotapi.NewEditMessageReplyMarkup(chatID, b.tempMsgID[chatID], buttonDone)
	_, err := b.bot.Send(msg)
	return err
}

func (b *Bot) handleDeleteProductCmd(callback *tgbotapi.CallbackQuery) error {
	chatID := callback.Message.Chat.ID

	product := b.tempProduct[chatID]

	err := b.storage.Remove(context.Background(), product.ProductID)

	if err != nil {
		msg := tgbotapi.NewMessage(chatID, "Не удалось удалить товар")
		if _, err := b.bot.Send(msg); err != nil {
			return err
		}
	}

	buttonDone := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Удалён", "done"),
		),
	)

	msg := tgbotapi.NewEditMessageReplyMarkup(chatID, b.tempMsgID[chatID], buttonDone)
	_, err = b.bot.Send(msg)
	delete(b.states, chatID)
	delete(b.tempProduct, chatID)
	delete(b.selectedParams, chatID)
	delete(b.tempMsgID, chatID)
	return err
}
