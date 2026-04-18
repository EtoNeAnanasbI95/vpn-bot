package keyboard

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/EtoNeAnanasbI95/vpn-bot/internal/bot/callback"
	"github.com/EtoNeAnanasbI95/vpn-bot/internal/domain"
)

// MainMenu returns the main reply keyboard for regular users.
func MainMenu() tgbotapi.ReplyKeyboardMarkup {
	return tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("🔗 Мои подключения"),
			tgbotapi.NewKeyboardButton("📖 Гайды"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("🆕 Запросить подключение"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("❓ Помощь"),
		),
	)
}

// AdminMainMenu returns the reply keyboard for admin users.
func AdminMainMenu() tgbotapi.ReplyKeyboardMarkup {
	return tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("🔗 Мои подключения"),
			tgbotapi.NewKeyboardButton("📖 Гайды"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("❓ Помощь"),
			tgbotapi.NewKeyboardButton("⚙️ Панель администратора"),
		),
	)
}

// ConnectionList builds an inline keyboard listing user connections with payment status.
func ConnectionList(conns []*domain.Connection) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton
	for _, c := range conns {
		activeIcon := "✅"
		if !c.IsActive {
			activeIcon = "❌"
		}

		switch c.PayStatus {
		case domain.ConnPayUnpaid:
			rows = append(rows, tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(activeIcon+" "+c.Label+" 💳", callback.ConnQR(c.UUID)),
				tgbotapi.NewInlineKeyboardButtonData("Оплатить", callback.ConnPay(c.UUID)),
			))
		case domain.ConnPayPending:
			rows = append(rows, tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(activeIcon+" "+c.Label+" ⏳", callback.ConnQR(c.UUID)),
			))
		default: // free or paid
			rows = append(rows, tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(activeIcon+" "+c.Label, callback.ConnQR(c.UUID)),
			))
		}
	}
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

// ConnRequestCheckPayButton shows "Проверить оплату" to the user after admin sends payment details.
func ConnRequestCheckPayButton(reqUUID string) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Я оплатил — проверить", callback.ConnReqCheckPay(reqUUID)),
		),
	)
}

// PlatformList builds an inline keyboard for guide platform selection.
func PlatformList(platforms []domain.Platform) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton
	for _, p := range platforms {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(p.Label, callback.GuideGet(p.Key)),
		))
	}
	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}
