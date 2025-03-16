package telegram

import (
	"context"
	"fmt"
	"log"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/shopspring/decimal"
)

func (b *Bot) procEditProductCmd(callback *tgbotapi.CallbackQuery, action string) error {
	chatID := callback.Message.Chat.ID

	// Инициализируем временные данные продукта и выбранные параметры
	if b.selectedParams[chatID] == nil {
		b.selectedParams[chatID] = make(map[string]bool)
	}

	if b.selectedParams[callback.Message.Chat.ID][action] {
		b.selectedParams[callback.Message.Chat.ID][action] = false
	} else {
		b.selectedParams[callback.Message.Chat.ID][action] = true
	}

	b.tempMsgID[chatID] = callback.Message.MessageID
	// Обновляем клавиатуру с галочками
	editProductKeyboard := b.generateEditProductKeyboard(chatID)
	msg := tgbotapi.NewEditMessageReplyMarkup(chatID, callback.Message.MessageID, editProductKeyboard)
	_, err := b.bot.Send(msg)
	return err
}

func (b *Bot) procConfirmEdit(message *tgbotapi.Message) error {
	chatID := message.Chat.ID
	selectedParams := b.selectedParams[chatID]
	text := message.Text
	state := b.states[chatID]

	paramOrder := []string{
		EditProductNameCmd,
		EditProductCountCmd,
		EditProductPurchaseCmd,
		EditProductSellingCmd,
		EditProductDescriptionCmd,
		EditProductImagesCmd,
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
		product.ShopID = b.user[chatID].ShopID
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
			return b.procConfirmEdit(message)

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
			return b.procConfirmEdit(message)

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
			return b.procConfirmEdit(message)
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
			return b.procConfirmEdit(message)
		case EditProductDescriptionCmd:
			if state != stateWaitingForEditSellingPrice {
				b.states[chatID] = stateWaitingForEditSellingPrice
				b.bot.Send(tgbotapi.NewMessage(chatID, "Введите новое описание:"))
				return nil
			} else {
				b.storage.UpdateProductField(context.Background(), product.ProductID, "description", text)
				b.bot.Send(tgbotapi.NewMessage(chatID, "Описание успешно обновлено!"))
				b.selectedParams[chatID][EditProductDescriptionCmd] = false
			}
			return b.procConfirmEdit(message)
		case EditProductImagesCmd:
			if state != stateWaitingForEditImage {
				b.states[chatID] = stateWaitingForEditImage
				b.bot.Send(tgbotapi.NewMessage(chatID, "Удалите фотографии или загрузите новые:"))
				images, err := b.storage.GetPhotosByProductID(context.Background(), product.ProductID)
				if err != nil {
					log.Printf("не удалось получить фото для продукта %d: %v", product.ProductID, err)
					continue // Продолжаем обрабатывать остальные товары
				}

				// Отправляем изображения (если есть)
				for _, image := range images {
					photoFile := tgbotapi.NewPhotoUpload(chatID, tgbotapi.FileBytes{
						Name:  fmt.Sprintf("product_%d_%d.jpg", product.ProductID, image.ImageID),
						Bytes: image.Byte,
					})
					photoFile.ParseMode = "Markdown"
					photoFile.ReplyMarkup = getImageActionKeyboard(image.ImageID)
					if _, err := b.bot.Send(photoFile); err != nil {
						log.Printf("не удалось отправить фото: %v", err)
					}
				}
				return b.generateFinishEditImagesButton(message)
			}
			if state == stateWaitingForEditImage || message.Text == "" {
				b.lastPhotoID = (*message.Photo)[len(*message.Photo)-1].FileID
				imageMeta, err := b.getFileMeta(b.lastPhotoID)
				if err != nil {
					msg := tgbotapi.NewMessage(chatID, "Ошибка обработки фото.")
					_, _ = b.bot.Send(msg)
					return err
				}

				if imageMeta.BarCode != "" {
					b.storage.UpdateProductField(context.Background(), product.ProductID, "bar_code", imageMeta.BarCode)
				} else {
					imageMeta.Byte, err = b.getFileContent(imageMeta.Url)
					if err != nil {
						msg := tgbotapi.NewMessage(chatID, "Ошибка обработки содержимого фото.")
						_, _ = b.bot.Send(msg)
						return err
					}
					product.Image = append(product.Image, imageMeta)
				}

				return nil
			}

			return b.procConfirmEdit(message)
		}
	}

	if len(selectedParams) == 0 {
		b.bot.Send(tgbotapi.NewMessage(chatID, "Выберите изменяемые параметры"))
		return nil
	}

	//Пересоздаём индекс
	err := b.RebuildIndex(chatID)
	if err != nil {
		return err
	}

	// Завершаем редактирование и очищаем временные данные
	b.bot.Send(tgbotapi.NewMessage(chatID, "Товар отредактирован!"))

	buttonDone := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Отредактировано", "done"),
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

func (b *Bot) generateFinishEditImagesButton(message *tgbotapi.Message) error {

	chatID := message.Chat.ID

	button := tgbotapi.NewInlineKeyboardButtonData("Продолжить", FinishEditImagesCmd)

	inlineKeyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(button),
	)

	msg := tgbotapi.NewMessage(chatID, "Если редактирование завершено")
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = inlineKeyboard

	_, err := b.bot.Send(msg)

	return err
}

func (b *Bot) procFinisEditImages(message *tgbotapi.Message) error {
	chatID := message.Chat.ID
	
	if b.tempProduct[chatID] == nil{
		return nil
	}
	
	b.selectedParams[chatID][EditProductImagesCmd] = false
	l := len(b.tempProduct[chatID].Image)
	if l > 1 {
		_, err := b.storage.SaveImage(context.Background(), b.tempProduct[chatID])
		if err != nil {
			msg := tgbotapi.NewMessage(chatID, "Ошибка сохранения фото.")
			_, _ = b.bot.Send(msg)
			return err
		}
	}

	return b.procConfirmEdit(message)
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
		b.generateToggleButton("Описание", EditProductDescriptionCmd, chatID),
		b.generateToggleButton("Фото", EditProductImagesCmd, chatID),
	}

	rows := [][]tgbotapi.InlineKeyboardButton{
		buttons[:2],
		buttons[2:4],
		buttons[4:6],
		buttons[6:],
		{tgbotapi.NewInlineKeyboardButtonData("Продолжить", ConfirmEditProductCmd)},
	}
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

// getImageActionKeyboard возвращает клавиатуру с действиями над изображением
func getImageActionKeyboard(imageID uint) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Удалить ❓", fmt.Sprintf("%s_%d", ConfirmDelImageCmd, imageID)),
		),
	)
}

func (b *Bot) procConfirmDeleteImageCmd(callback *tgbotapi.CallbackQuery, imageID uint) error {
	chatID := callback.Message.Chat.ID

	buttonDone := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Удалить", fmt.Sprintf("%s_%d", DelImageCmd, imageID)),
			tgbotapi.NewInlineKeyboardButtonData("Нет", fmt.Sprintf("%s_%d", ActionsImagesCmd, imageID)),
		),
	)

	msg := tgbotapi.NewEditMessageReplyMarkup(chatID, callback.Message.MessageID, buttonDone)
	_, err := b.bot.Send(msg)
	return err
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

func (b *Bot) procConfirmDeleteProductCmd(callback *tgbotapi.CallbackQuery) error {
	chatID := callback.Message.Chat.ID
	product := b.tempProduct[chatID]

	buttonDone := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Удалить", fmt.Sprintf("%s_%d", DelProductCmd, product.ProductID)),
			tgbotapi.NewInlineKeyboardButtonData("Нет", fmt.Sprintf("%s_%d", ActionsProductCmd, product.ProductID)),
		),
	)

	msg := tgbotapi.NewEditMessageReplyMarkup(chatID, callback.Message.MessageID, buttonDone)
	_, err := b.bot.Send(msg)
	return err
}

func (b *Bot) procDeleteProductCmd(callback *tgbotapi.CallbackQuery) error {
	chatID := callback.Message.Chat.ID

	product := b.tempProduct[chatID]

	err := b.storage.RemoveProduct(context.Background(), product.ProductID)

	if err != nil {
		msg := tgbotapi.NewMessage(chatID, "Не удалось удалить товар")
		if _, err := b.bot.Send(msg); err != nil {
			return err
		}
	}

	buttonDone := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Удалено", "done"),
		),
	)

	//пересоздаём индекс
	err = b.RebuildIndex(chatID)
	if err != nil {
		return err
	}

	msg := tgbotapi.NewEditMessageReplyMarkup(chatID, b.tempMsgID[chatID], buttonDone)
	_, err = b.bot.Send(msg)
	delete(b.states, chatID)
	delete(b.tempProduct, chatID)
	delete(b.selectedParams, chatID)
	delete(b.tempMsgID, chatID)

	return err
}

func (b *Bot) procDeleteImageCmd(callback *tgbotapi.CallbackQuery, imageID uint) error {
	chatID := callback.Message.Chat.ID

	err := b.storage.DeleteImage(context.Background(), imageID)

	if err != nil {
		msg := tgbotapi.NewMessage(chatID, "Не удалось удалить изображение")
		if _, err := b.bot.Send(msg); err != nil {
			return err
		}
	}

	buttonDone := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Удалён", "done"),
		),
	)
	
	msg := tgbotapi.NewEditMessageReplyMarkup(chatID, callback.Message.MessageID, buttonDone)
	_, err = b.bot.Send(msg)
	return err
}
