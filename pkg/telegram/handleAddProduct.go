package telegram

import (
	"context"
	"fmt"
	"log"
	"strconv"

	"github.com/Bariban/vector-shop-bot/pkg/storage"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/shopspring/decimal"
)

func (b *Bot) handleAddProductCmd(message *tgbotapi.Message) error {
	chatID := message.Chat.ID
	product := b.tempProduct[chatID]
	if message.Text == "Добавить товар" {
		delete(b.states, chatID)
		delete(b.tempProduct, chatID)
	}
	if b.states[chatID] != stateWaitingForPhoto {
		b.states[chatID] = stateWaitingForPhoto
		product := &storage.Product{
			UserName: message.From.UserName,
			Image:    []*storage.ImageMeta{{}},
		}
		b.tempProduct[chatID] = product
		msg := tgbotapi.NewMessage(chatID, b.messages.Responses.SendPhoto)
		_, err := b.bot.Send(msg)
		return err
	}
	if product == nil {
		return fmt.Errorf("product data not initialized for chat: %d", chatID)
	}

	if len(product.Image) == 0 {
		product.Image = append(product.Image, &storage.ImageMeta{})
	}

	switch b.states[chatID] {
	case stateWaitingForPhoto:
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
			var message string
			if l == 1 {
				message = "❗️ Найден похожий товар"
			} else {
				message = "❗️ Найдены похожие товары"
			}
			msg := tgbotapi.NewMessage(chatID, message)
			_, _ = b.bot.Send(msg)
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

				actionsProductKeyboard := b.getProductActionKeyboard(product.ProductID)

				msg := tgbotapi.NewMessage(chatID, productInfo)
				msg.ParseMode = "Markdown"
				msg.ReplyMarkup = actionsProductKeyboard

				sentMsg, err := b.bot.Send(msg)
				if err != nil {
					log.Printf("не удалось отправить информацию о продукте: %v", err)
					return err
				}

				b.tempMsgID[chatID] = sentMsg.MessageID
			}
			return err
		}

		product.Image[0].Url = imageMeta.Url
		product.Image[0].Float = imageMeta.Float
		b.states[chatID] = stateWaitingForName
		msg := tgbotapi.NewMessage(chatID, "Введите название товара:")
		_, err = b.bot.Send(msg)
		return err
	case stateWaitingForName:
		product.Name = message.Text
		b.states[chatID] = stateWaitingForDescription
		msg := tgbotapi.NewMessage(chatID, "Введите описание товара:")
		_, err := b.bot.Send(msg)
		return err

	case stateWaitingForDescription:
		product.Description = message.Text
		b.states[chatID] = stateWaitingForCount
		msg := tgbotapi.NewMessage(chatID, "Введите количество товара:")
		_, err := b.bot.Send(msg)
		return err

	case stateWaitingForCount:
		count, err := strconv.Atoi(message.Text)
		if err != nil {
			msg := tgbotapi.NewMessage(chatID, "Введите корректное количество.")
			_, _ = b.bot.Send(msg)
			return err
		}
		product.Count = uint(count)
		b.states[chatID] = stateWaitingForPurchasePrice
		msg := tgbotapi.NewMessage(chatID, "Введите цену закупки:")
		_, err = b.bot.Send(msg)
		return err

	case stateWaitingForPurchasePrice:
		price, err := decimal.NewFromString(message.Text)
		if err != nil {
			msg := tgbotapi.NewMessage(chatID, "Введите корректную цену закупки.")
			_, _ = b.bot.Send(msg)
			return err
		}
		product.PurchasePrice = price
		b.states[chatID] = stateWaitingForSellingPrice
		msg := tgbotapi.NewMessage(chatID, "Введите цену продажи:")
		_, err = b.bot.Send(msg)
		return err

	case stateWaitingForSellingPrice:
		price, err := decimal.NewFromString(message.Text)
		if err != nil {
			msg := tgbotapi.NewMessage(chatID, "Введите корректную цену продажи.")
			_, _ = b.bot.Send(msg)
			return err
		}
		product.SellingPrice = price

		// Сохраняем продукт в БД
		product.ProductID, err = b.storage.Save(context.Background(), product)
		if err != nil {
			msg := tgbotapi.NewMessage(chatID, "Ошибка сохранения товара.")
			_, _ = b.bot.Send(msg)
			return err
		}
		// Сохраняем изображение в БД

		product.Image[0].Byte, err = b.getFileContent(product.Image[0].Url)
		if err != nil {
			msg := tgbotapi.NewMessage(chatID, "Ошибка обработки содержимого фото.")
			_, _ = b.bot.Send(msg)
			return err
		}
		err = b.storage.SaveImage(context.Background(), product)
		if err != nil {
			msg := tgbotapi.NewMessage(chatID, "Ошибка сохранения фото.")
			_, _ = b.bot.Send(msg)
			return err
		}

		delete(b.states, chatID)
		delete(b.tempProduct, chatID)

		msg := tgbotapi.NewMessage(chatID, "Товар успешно добавлен!")
		_, err = b.bot.Send(msg)
		return err
	}

	return nil
}
