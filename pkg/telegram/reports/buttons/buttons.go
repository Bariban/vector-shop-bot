package reports

import (
	"context"
	"fmt"
	"log"

	"github.com/Bariban/vector-shop-bot/pkg/storage"
	"github.com/Bariban/vector-shop-bot/pkg/utils"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"

	qrcode "github.com/skip2/go-qrcode"
)

func getReportPeriodKeyboard(role string) tgbotapi.InlineKeyboardMarkup {

	switch role {
	case roleCustomer:
		return tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Неделя", ShopKeyboardCmd),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Текущий месяц", ShopKeyboardCmd),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Предыдущий месяц", ShopKeyboardCmd),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Год", ShopKeyboardCmd),
			),
		)
	return tgbotapi.NewInlineKeyboardMarkup()
	}
}