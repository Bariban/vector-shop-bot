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
	if b.states[chatID] == 0 {
		b.states[chatID] = stateWaitingForPhoto
		product := &storage.Product{
			UserName: message.From.UserName,
			Image:    []*storage.ImageMeta{},
		}
		b.tempProduct[chatID] = product
		msg := tgbotapi.NewMessage(chatID, b.messages.Responses.SendPhoto)
		_, err := b.bot.Send(msg)
		return err
	}
	if product == nil {
		return fmt.Errorf("product data not initialized for chat: %d", chatID)
	}

	// if len(product.Image) == 0 {
	// 	product.Image = append(product.Image, &storage.ImageMeta{})
	// }

	switch b.states[chatID] {
	case stateWaitingForPhoto:
		b.lastPhotoID = (*message.Photo)[len(*message.Photo)-1].FileID
		imageMeta, err := b.getFileMeta(b.lastPhotoID)
		if err != nil {
			msg := tgbotapi.NewMessage(chatID, "Ошибка обработки фото.")
			_, _ = b.bot.Send(msg)
			return err
		}

		// Добавляем новое изображение в список
		product.Image = append(product.Image, imageMeta)

		b.states[chatID] = stateWaitingForName

		return err

	case stateWaitingForName:
		// Проверяем, не пришло ли фото вместо текста
		if message.Text == "" {
			b.lastPhotoID = (*message.Photo)[len(*message.Photo)-1].FileID
			imageMeta, err := b.getFileMeta(b.lastPhotoID)
			if err != nil {
				msg := tgbotapi.NewMessage(chatID, "Ошибка обработки фото.")
				_, _ = b.bot.Send(msg)
				return err
			}

			// Добавляем новое изображение в список
			product.Image = append(product.Image, imageMeta)

			b.states[chatID] = stateWaitingForName

			return err
		}

		foundProduct, err := b.getProductsByVector(message, product.Image)
		if err != nil {
			msg := tgbotapi.NewMessage(chatID, "Ошибка обработки фото.")
			_, _ = b.bot.Send(msg)
			return err
		}

		l := len(foundProduct)
		if l > 0 {
			var str string
			if l == 1 {
				str = "❗️ Найден похожий товар."
			} else {
				str = "❗️ Найдены похожие товары."
			}
			msg := tgbotapi.NewMessage(chatID, str)
			_, _ = b.bot.Send(msg)
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
			product.Name = message.Text
			b.states[chatID] = stateWaitingForDescription
			msg = tgbotapi.NewMessage(chatID, "Или введите описание нового товара:")
			_, err = b.bot.Send(msg)
			return err
		}

		product.Name = message.Text
		b.states[chatID] = stateWaitingForDescription
		msg := tgbotapi.NewMessage(chatID, "Введите описание товара:")
		_, err = b.bot.Send(msg)
		return err

	case stateWaitingForDescription:
		if message.Text == "" {
			return nil
		}
		product.Description = message.Text
		b.states[chatID] = stateWaitingForCount
		msg := tgbotapi.NewMessage(chatID, "Введите количество товара:")
		_, err := b.bot.Send(msg)
		return err

	case stateWaitingForCount:
		if message.Text == "" {
			return nil
		}
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
		if message.Text == "" {
			return nil
		}
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
		if message.Text == "" {
			return nil
		}
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

		for i, _ := range product.Image {
			product.Image[i].Byte, err = b.getFileContent(product.Image[i].Url)
			if err != nil {
				msg := tgbotapi.NewMessage(chatID, "Ошибка обработки содержимого фото.")
				_, _ = b.bot.Send(msg)
				return err
			}
		}

		p, err := b.storage.SaveImage(context.Background(), product)
		if err != nil {
			msg := tgbotapi.NewMessage(chatID, "Ошибка сохранения фото.")
			_, _ = b.bot.Send(msg)
			return err
		}

		product = p

		err = b.AddVectorToIndex(chatID, product.Image)
		if err != nil {
			msg := tgbotapi.NewMessage(chatID, "Ошибка сохранения вектора в индекс.")
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

// AddVectorToIndex добавляет вектор изображения в индекс
func (b *Bot) AddVectorToIndex(chatID int64, images []*storage.ImageMeta) error {
	for _, image := range images {

		len := len(image.Float)
		expectedDim := b.options.VectorExpectedDim
		if len != expectedDim {
			return fmt.Errorf("vector dimension mismatch: expected %d, got %d", expectedDim, len)
		}
		// Добавляем вектор в индекс
		b.index[chatID].Add(image.Float, uint32(image.ImageID))
	}

	return nil
}
