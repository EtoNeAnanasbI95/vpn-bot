package handler

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/EtoNeAnanasbI95/vpn-bot/internal/config"
	"github.com/EtoNeAnanasbI95/vpn-bot/internal/usecase"
)

// UseCases bundles all application use cases for handler injection.
type UseCases struct {
	User        usecase.UserUseCase
	Connection  usecase.ConnectionUseCase
	Payment     usecase.PaymentUseCase
	Guide       usecase.GuideUseCase
	ConnRequest usecase.ConnRequestUseCase
}

// send is a convenience wrapper to swallow the (Msg, error) return of bot.Send.
func send(bot *tgbotapi.BotAPI, cfg tgbotapi.Chattable) {
	bot.Send(cfg) //nolint:errcheck
}

// answerCallback acknowledges an inline keyboard button press.
func answerCallback(bot *tgbotapi.BotAPI, callbackID, text string) {
	bot.Request(tgbotapi.NewCallback(callbackID, text)) //nolint:errcheck
}

// editText replaces the text of an existing message.
func editText(bot *tgbotapi.BotAPI, chatID int64, messageID int, text string) {
	edit := tgbotapi.NewEditMessageText(chatID, messageID, text)
	edit.ParseMode = tgbotapi.ModeHTML
	bot.Send(edit) //nolint:errcheck
}

// editMarkup replaces the inline keyboard of an existing message.
func editMarkup(bot *tgbotapi.BotAPI, chatID int64, messageID int, markup tgbotapi.InlineKeyboardMarkup) {
	edit := tgbotapi.NewEditMessageReplyMarkup(chatID, messageID, markup)
	bot.Send(edit) //nolint:errcheck
}

// adminUsernames returns @mention strings for all configured admin IDs.
func adminUsernames(bot *tgbotapi.BotAPI, cfg *config.Config) string {
	result := ""
	for _, id := range cfg.AdminIDs {
		chat, err := bot.GetChat(tgbotapi.ChatInfoConfig{ChatConfig: tgbotapi.ChatConfig{ChatID: id}})
		if err == nil && chat.UserName != "" {
			result += "@" + chat.UserName + " "
		}
	}
	if result == "" {
		result = "(недоступно)"
	}
	return result
}
