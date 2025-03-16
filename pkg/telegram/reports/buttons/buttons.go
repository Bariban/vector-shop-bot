package reports

import (
	t "github.com/Bariban/vector-shop-bot/pkg/telegram"
	m "github.com/Bariban/vector-shop-bot/pkg/telegram/reports/model"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

func getReportPeriodKeyboard(role string) tgbotapi.InlineKeyboardMarkup {

	switch role {
	case t.RoleCustomer:
		return tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Неделя", m.WeekReportCmd),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Текущий месяц", m.CurrentMonthReportCmd),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Предыдущий месяц", m.PreviousMonthReportCmd),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Год", m.EarReportCmd),
			),
		)
	}
	return tgbotapi.NewInlineKeyboardMarkup()
}
