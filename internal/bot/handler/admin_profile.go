package handler

import (
	"context"
	"fmt"
	"log/slog"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/EtoNeAnanasbI95/vpn-bot/internal/bot/keyboard"
	"github.com/EtoNeAnanasbI95/vpn-bot/internal/bot/middleware"
	"github.com/EtoNeAnanasbI95/vpn-bot/internal/bot/session"
)

// HandleAdminPaymentInfoMenu shows current payment info and prompts for new input.
func HandleAdminPaymentInfoMenu(
	ctx context.Context,
	bot *tgbotapi.BotAPI,
	chatID int64,
	callbackID string,
	adminID int64,
	sessions session.Store,
	uc *UseCases,
) {
	slog.Info("admin: payment info menu opened", "admin_id", adminID)
	answerCallback(bot, callbackID, "")

	currentInfo, err := uc.Connection.GetAdminOwnPaymentInfo(ctx, adminID)
	if err != nil {
		slog.Warn("admin: get payment info", "admin_id", adminID, "err", err)
	}
	if currentInfo == "" {
		currentInfo = "не задано"
	}

	sessions.Set(adminID, &session.Session{
		State: session.StateSetPaymentInfo,
	})

	msg := tgbotapi.NewMessage(chatID, fmt.Sprintf(
		"💰 <b>Реквизиты для оплаты</b>\n\nТекущие реквизиты:\n<code>%s</code>\n\nВведите новые реквизиты (любой текст):",
		currentInfo,
	))
	msg.ParseMode = tgbotapi.ModeHTML
	msg.ReplyMarkup = keyboard.CancelKeyboard()
	send(bot, msg)
}

// HandleSessionSetPaymentInfo processes the admin's payment info input.
func HandleSessionSetPaymentInfo(
	ctx context.Context,
	bot *tgbotapi.BotAPI,
	msg *tgbotapi.Message,
	uc *UseCases,
) {
	u := middleware.UserFromCtx(ctx)

	if msg.Text == "" {
		send(bot, tgbotapi.NewMessage(msg.Chat.ID, "❌ Реквизиты не могут быть пустыми."))
		return
	}

	if err := uc.Connection.SetAdminPaymentInfo(ctx, u.ID, msg.Text); err != nil {
		slog.Error("admin: set payment info", "admin_id", u.ID, "err", err)
		send(bot, tgbotapi.NewMessage(msg.Chat.ID, "❌ Ошибка при сохранении реквизитов."))
		return
	}

	slog.Info("admin: payment info updated", "admin_id", u.ID)
	reply := tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("✅ Реквизиты сохранены:\n\n<code>%s</code>", msg.Text))
	reply.ParseMode = tgbotapi.ModeHTML
	send(bot, reply)
}
