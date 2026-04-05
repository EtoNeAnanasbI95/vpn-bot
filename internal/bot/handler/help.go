package handler

import (
	"context"
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/EtoNeAnanasbI95/vpn-bot/internal/config"
)

// HandleHelp handles the /help command and the "❓ Помощь" button.
func HandleHelp(
	ctx context.Context,
	bot *tgbotapi.BotAPI,
	chatID int64,
	cfg *config.Config,
) {
	admins := adminUsernames(bot, cfg)
	text := fmt.Sprintf(
		"❓ <b>Помощь</b>\n\nЕсли у вас есть вопросы или проблемы, свяжитесь с администратором:\n%s",
		admins,
	)
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = tgbotapi.ModeHTML
	send(bot, msg)
}
