package telegram

import (
	"context"
	"fmt"
	"log"

	"github.com/Bariban/vector-shop-bot/pkg/storage"
	"github.com/Bariban/vector-shop-bot/pkg/utils"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"

	qrcode "github.com/skip2/go-qrcode"
)

func getMenuKeyboard(role string) tgbotapi.InlineKeyboardMarkup {

	switch role {
	case roleCustomer:
		return tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Магазин", ShopKeyboardCmd),
			), tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Отчёты", ReportsKeyboardCmd),
			),
		)

	case roleEmployee:
		return tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Магазин", ShopKeyboardCmd),
			),
		)

	case roleClient:
		return tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Магазин", ShopKeyboardCmd),
			),
		)
	}
	return tgbotapi.NewInlineKeyboardMarkup()
}

func getShopKeyboard(role string) tgbotapi.InlineKeyboardMarkup {

	switch role {
	case roleCustomer:
		return tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Редактировать магазин", EditShopKeyboardCmd),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Сменить магазин", ChangeShopCmd),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Создать новый магазин", CreateShopCmd),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Пригласить пользователя", InviteUsersCmd),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🔙 Назад", MenuCmd),
			),
		)

	case roleEmployee:
		return tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Сменить магазин", ChangeShopCmd),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Создать новый магазин", CreateShopCmd),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🔙 Назад", MenuCmd),
			),
		)

	case roleClient:
		return tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Сменить магазин", ChangeShopCmd),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Создать новый магазин", CreateShopCmd),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🔙 Назад", MenuCmd),
			),
		)
	}
	return tgbotapi.NewInlineKeyboardMarkup()
}

func getEditShopKeyboard(role string) tgbotapi.InlineKeyboardMarkup {
	if role == roleCustomer {
		return tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Изменить название магазина", EditShopNameCmd),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Изменить описание магазина", EditShopDescriptionCmd),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Пользователи", UserListCmd),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Удалить магазин", DropShopCmd),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🔙 Назад", ShopKeyboardCmd),
			),
		)
	}
	return tgbotapi.NewInlineKeyboardMarkup()
}

// getInviteUsersKeyboard возвращает кнопки для генерации приглашения
func getInviteUsersKeyboard(role string) tgbotapi.InlineKeyboardMarkup {
	if role == roleCustomer {
		return tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Пригласительное для сотрудника", InviteEmployeeCmd),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Пригласительное для клиента", InviteClientCmd),
			),
		)
	}
	return tgbotapi.NewInlineKeyboardMarkup()
}

func (b *Bot) procGetMenu(message *tgbotapi.Message) error {
	var chatID = message.Chat.ID
	if message.Text == "Меню" {
		msg := tgbotapi.NewMessage(chatID, "Меню:")
		msg.ParseMode = "Markdown"
		msg.ReplyMarkup = getMenuKeyboard(b.user[chatID].Role)

		sentMsg, err := b.bot.Send(msg)

		b.tempMsgID[chatID] = sentMsg.MessageID
		if err != nil {
			log.Printf("не удалось отправить меню: %v", err)
			return err
		}
	} else {
		msg := tgbotapi.NewEditMessageReplyMarkup(chatID, b.tempMsgID[chatID], getMenuKeyboard(b.user[chatID].Role))

		_, err := b.bot.Send(msg)
		if err != nil {
			log.Printf("не удалось отправить меню: %v", err)
			return err
		}
	}

	return nil
}

func (b *Bot) procShopKeyboard(message *tgbotapi.Message) error {
	var chatID = message.Chat.ID

	msg := tgbotapi.NewEditMessageReplyMarkup(chatID, b.tempMsgID[chatID], getShopKeyboard(b.user[chatID].Role))

	_, err := b.bot.Send(msg)
	if err != nil {
		log.Printf("не удалось отправить меню: %v", err)
		return err
	}
	return nil
}

func (b *Bot) procEditShopKeyboard(message *tgbotapi.Message) error {
	var chatID = message.Chat.ID

	msg := tgbotapi.NewEditMessageReplyMarkup(chatID, b.tempMsgID[chatID], getEditShopKeyboard(b.user[chatID].Role))
	_, err := b.bot.Send(msg)

	if err != nil {
		log.Printf("не удалось отправить меню: %v", err)
		return err
	}
	return nil
}

func (b *Bot) procInviteUsersKeyboard(message *tgbotapi.Message) error {
	var chatID = message.Chat.ID

	msg := tgbotapi.NewEditMessageReplyMarkup(chatID, b.tempMsgID[chatID], getInviteUsersKeyboard(b.user[chatID].Role))
	_, err := b.bot.Send(msg)

	if err != nil {
		log.Printf("не удалось отправить меню: %v", err)
		return err
	}
	return nil
}

func (b *Bot) procEditShopName(message *tgbotapi.Message) error {
	var chatID = message.Chat.ID

	// Проверяем состояние пользователя
	if b.states[chatID] != stateEditShopName {
		b.states[chatID] = stateEditShopName
		b.bot.Send(tgbotapi.NewMessage(chatID, "Введите новое название магазина:"))
		return nil
	}

	// Обновляем название магазина
	if b.user != nil { // Проверяем, что указатель b.user не nil
		err := b.storage.UpdateShopField(context.Background(), b.user[chatID].ShopID, "name", message.Text)
		if err != nil {
			b.bot.Send(tgbotapi.NewMessage(chatID, "Ошибка при обновлении названия магазина."))
			return err
		}
		b.bot.Send(tgbotapi.NewMessage(chatID, "Название успешно обновлено!"))
	} else {
		b.bot.Send(tgbotapi.NewMessage(chatID, "Ошибка: пользователь не найден."))
		return fmt.Errorf("user not found")
	}
	delete(b.states, chatID)

	return nil
}

func (b *Bot) procInviteEmployee(message *tgbotapi.Message) error {
	return b.processInvite(message, roleEmployee, "Ссылка для сотрудника")
}

func (b *Bot) procInviteClient(message *tgbotapi.Message) error {
	return b.processInvite(message, roleClient, "Ссылка для клиента")
}

func (b *Bot) processInvite(message *tgbotapi.Message, role string, captionPrefix string) error {
	var chatID = message.Chat.ID

	// Генерация ссылки
	link, err := b.generateInviteLink(chatID, b.user[chatID].ShopID, role)
	if err != nil {
		b.bot.Send(tgbotapi.NewMessage(chatID, "Не удалось сгенерировать ссылку."))
		return err
	}

	qrImage, err := qrcode.Encode(link, qrcode.Medium, 256)
	if err != nil {
		log.Printf("Ошибка при генерации QR-кода: %v", err)
		b.bot.Send(tgbotapi.NewMessage(chatID, "Ошибка при создании QR-кода."))
		return err
	}

	// Отправка QR-кода как изображения
	photo := tgbotapi.NewPhotoUpload(chatID, tgbotapi.FileBytes{
		Name:  "qr_code.png",
		Bytes: qrImage,
	})
	photo.Caption = fmt.Sprintf("%s: %s", captionPrefix, link)

	_, err = b.bot.Send(photo)
	if err != nil {
		log.Printf("Ошибка при отправке изображения: %v", err)
		b.bot.Send(tgbotapi.NewMessage(chatID, "Ошибка при отправке QR-кода."))
		return err
	}

	return nil
}

// generateInviteLink генерирует реферальную ссылку
func (b *Bot) generateInviteLink(chatID int64, shopID uint, role string) (string, error) {
	user, ok := b.user[chatID]
	if !ok || user.Role != roleCustomer {
		return "", fmt.Errorf("только владелец магазина может генерировать ссылки")
	}

	link := utils.GenerateReferralLink("amazing_mag_bot", shopID, role)
	return link, nil
}

// procGetShopUsers возвращает пользователей магазина
func (b *Bot) procGetShopUsers(message *tgbotapi.Message) error {

	chatID := message.Chat.ID

	users, err := b.storage.GetUsersByShopID(context.Background(), b.user[chatID].ShopID)

	if err != nil {
		log.Printf("ошибка при получении пользователей: %v", err)
		b.bot.Send(tgbotapi.NewMessage(chatID, "Ошибка при получении пользователей."))
		return err
	}

	msg := tgbotapi.NewEditMessageReplyMarkup(chatID, b.tempMsgID[chatID], getShopUsersKeyboard(users))
	_, err = b.bot.Send(msg)

	if err != nil {
		log.Printf("не удалось отправить пользователей: %v", err)
		return err
	}

	return nil
}

// getShopUsersKeyboard возвращает клавиатуру с пользователями магазин
func getShopUsersKeyboard(users []*storage.User) tgbotapi.InlineKeyboardMarkup {
	var keyboardRows [][]tgbotapi.InlineKeyboardButton

	// Генерация кнопок для каждого пользователя
	for _, user := range users {
		userInfo := fmt.Sprintf(
			"👤 %s %s @%s Роль: %s",
			user.FirstName,
			user.LastName,
			user.UserName,
			user.Role,
		)

		// Создаём кнопку с данными о пользователе
		button := tgbotapi.NewInlineKeyboardButtonData(userInfo, fmt.Sprintf("%s_%d", EditUserCmd, user.UserID))
		keyboardRows = append(keyboardRows, tgbotapi.NewInlineKeyboardRow(button))
	}

	// Добавляем кнопку "Назад" в конец
	backButton := tgbotapi.NewInlineKeyboardButtonData("🔙 Назад", MenuCmd)
	keyboardRows = append(keyboardRows, tgbotapi.NewInlineKeyboardRow(backButton))

	// Создаём и возвращаем клавиатуру
	return tgbotapi.NewInlineKeyboardMarkup(keyboardRows...)
}

// procGetShopUsers возвращает клавиатурусмены ролей пользователей магазина
func (b *Bot) procEditShopUser(message *tgbotapi.Message, userID uint) error {

	chatID := message.Chat.ID

	// Инициализируем временные данные пользователя и выбранные параметры
	if b.selectedParams[chatID] == nil {
		b.selectedParams[chatID] = make(map[string]bool)
	}

	msg := tgbotapi.NewEditMessageReplyMarkup(chatID, b.tempMsgID[chatID], b.getEditUsersKeyboard(userID, chatID))
	_, err := b.bot.Send(msg)

	if err != nil {
		log.Printf("не удалось отправить клавиатуру смены ролей пользователя: %v", err)
		return err
	}

	return nil
}

// getEditUsersKeyboard возвращает кнопки для редактирования пользователей
func (b *Bot) getEditUsersKeyboard(userID uint, chatID int64) tgbotapi.InlineKeyboardMarkup {

	// Создаём кнопки с учётом текущего состояния
	buttons := []tgbotapi.InlineKeyboardButton{
		b.generateToggleButton("Управляющий", fmt.Sprintf("%s_%d", GrantRoleCustomerCmd, userID), chatID),
		b.generateToggleButton("Продавец", fmt.Sprintf("%s_%d", GrantRoleEmployeeCmd, userID), chatID),
		b.generateToggleButton("Клиент", fmt.Sprintf("%s_%d", GrantRoleClientCmd, userID), chatID),
	}

	rows := [][]tgbotapi.InlineKeyboardButton{
		buttons[:1],
		buttons[1:2],
		buttons[2:],
		{tgbotapi.NewInlineKeyboardButtonData("🔙 Назад", UserListCmd)},
	}
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

// procGrantRoleCustomer меняет роль пользователя на управляющего и возвращает клавиатуру смены ролей пользователей магазина
func (b *Bot) procGrantRoleCustomer(message *tgbotapi.Message, userID uint) error {

	chatID := message.Chat.ID

	prefix := fmt.Sprintf("_%d", userID)

	// Инициализируем временные данные пользователя и выбранные параметры
	if b.selectedParams[chatID] == nil {
		b.selectedParams[chatID] = make(map[string]bool)
	}
	b.selectedParams[chatID][GrantRoleCustomerCmd+prefix] = true
	b.selectedParams[chatID][GrantRoleEmployeeCmd+prefix] = false
	b.selectedParams[chatID][GrantRoleClientCmd+prefix] = false

	err := b.storage.UpdateUserRole(context.Background(), userID, b.user[chatID].ShopID, roleCustomer)
	if err != nil {
		return err
	}

	msg := tgbotapi.NewEditMessageReplyMarkup(chatID, b.tempMsgID[chatID], b.getEditUsersKeyboard(userID, chatID))
	_, err = b.bot.Send(msg)

	if err != nil {
		log.Printf("не удалось отправить клавиатуру смены ролей пользователя: %v", err)
		return err
	}

	return nil
}

// procGrantRoleEmployee меняет роль пользователя на продавца и возвращает клавиатуру смены ролей пользователей магазина
func (b *Bot) procGrantRoleEmployee(message *tgbotapi.Message, userID uint) error {

	chatID := message.Chat.ID

	prefix := fmt.Sprintf("_%d", userID)

	// Инициализируем временные данные пользователя и выбранные параметры
	if b.selectedParams[chatID] == nil {
		b.selectedParams[chatID] = make(map[string]bool)
	}
	b.selectedParams[chatID][GrantRoleCustomerCmd+prefix] = false
	b.selectedParams[chatID][GrantRoleEmployeeCmd+prefix] = true
	b.selectedParams[chatID][GrantRoleClientCmd+prefix] = false

	err := b.storage.UpdateUserRole(context.Background(), userID, b.user[chatID].ShopID, roleEmployee)
	if err != nil {
		return err
	}

	msg := tgbotapi.NewEditMessageReplyMarkup(chatID, b.tempMsgID[chatID], b.getEditUsersKeyboard(userID, chatID))
	_, err = b.bot.Send(msg)

	if err != nil {
		log.Printf("не удалось отправить клавиатуру смены ролей пользователя: %v", err)
		return err
	}

	return nil
}

// procGrantRoleClient меняет роль пользователя на клиента и возвращает клавиатуру смены ролей пользователей магазина
func (b *Bot) procGrantRoleClient(message *tgbotapi.Message, userID uint) error {

	chatID := message.Chat.ID

	prefix := fmt.Sprintf("_%d", userID)

	// Инициализируем временные данные пользователя и выбранные параметры
	if b.selectedParams[chatID] == nil {
		b.selectedParams[chatID] = make(map[string]bool)
	}
	b.selectedParams[chatID][GrantRoleCustomerCmd+prefix] = false
	b.selectedParams[chatID][GrantRoleEmployeeCmd+prefix] = false
	b.selectedParams[chatID][GrantRoleClientCmd+prefix] = true

	err := b.storage.UpdateUserRole(context.Background(), userID, b.user[chatID].ShopID, roleClient)
	if err != nil {
		return err
	}

	msg := tgbotapi.NewEditMessageReplyMarkup(chatID, b.tempMsgID[chatID], b.getEditUsersKeyboard(userID, chatID))
	_, err = b.bot.Send(msg)

	if err != nil {
		log.Printf("не удалось отправить клавиатуру смены ролей пользователя: %v", err)
		return err
	}

	return nil
}
