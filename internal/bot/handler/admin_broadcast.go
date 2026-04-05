package handler

import (
	"context"
	"fmt"
	"log/slog"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/EtoNeAnanasbI95/vpn-bot/internal/bot/keyboard"
	"github.com/EtoNeAnanasbI95/vpn-bot/internal/bot/session"
)

// HandleAdminPanel shows the admin panel.
func HandleAdminPanel(
	ctx context.Context,
	bot *tgbotapi.BotAPI,
	chatID int64,
) {
	msg := tgbotapi.NewMessage(chatID, "⚙️ <b>Панель администратора</b>")
	msg.ParseMode = tgbotapi.ModeHTML
	msg.ReplyMarkup = keyboard.AdminPanel()
	send(bot, msg)
}

// HandleBroadcastAll starts the "broadcast to all" flow.
func HandleBroadcastAll(
	ctx context.Context,
	bot *tgbotapi.BotAPI,
	chatID int64,
	callbackID string,
	senderID int64,
	sessions session.Store,
) {
	answerCallback(bot, callbackID, "")
	sessions.Set(senderID, &session.Session{
		State: session.StateBroadcastAll,
	})
	msg := tgbotapi.NewMessage(chatID, "📢 Введите текст для рассылки <b>всем клиентам</b>:")
	msg.ParseMode = tgbotapi.ModeHTML
	send(bot, msg)
}

// HandleBroadcastSelectUser shows user list to pick broadcast target.
func HandleBroadcastSelectUser(
	ctx context.Context,
	bot *tgbotapi.BotAPI,
	chatID int64,
	callbackID string,
	uc *UseCases,
) {
	answerCallback(bot, callbackID, "")

	users, err := uc.User.GetAll(ctx)
	if err != nil || len(users) == 0 {
		send(bot, tgbotapi.NewMessage(chatID, "Нет зарегистрированных клиентов."))
		return
	}

	msg := tgbotapi.NewMessage(chatID, "👤 Выберите клиента для рассылки:")
	msg.ReplyMarkup = keyboard.UserListForAction(users, func(uid int64) string {
		return fmt.Sprintf("adm_bcast_user|%d", uid)
	})
	send(bot, msg)
}

// HandleBroadcastToUser starts the "broadcast to specific user" flow.
func HandleBroadcastToUser(
	ctx context.Context,
	bot *tgbotapi.BotAPI,
	chatID int64,
	callbackID string,
	targetUserID int64,
	senderID int64,
	sessions session.Store,
	uc *UseCases,
) {
	user, err := uc.User.GetUser(ctx, targetUserID)
	if err != nil {
		answerCallback(bot, callbackID, "Пользователь не найден")
		return
	}
	answerCallback(bot, callbackID, "")

	sessions.Set(senderID, &session.Session{
		State: session.StateBroadcastToUser,
		Data:  map[string]string{session.KeyTargetUserID: fmt.Sprintf("%d", user.ID)},
	})

	msg := tgbotapi.NewMessage(chatID,
		fmt.Sprintf("💬 Введите сообщение для <b>%s</b>:", user.DisplayName()))
	msg.ParseMode = tgbotapi.ModeHTML
	send(bot, msg)
}

// ExecuteBroadcastAll sends a message to all users and reports results.
func ExecuteBroadcastAll(
	ctx context.Context,
	bot *tgbotapi.BotAPI,
	adminChatID int64,
	text string,
	uc *UseCases,
) {
	users, err := uc.User.GetAll(ctx)
	if err != nil {
		send(bot, tgbotapi.NewMessage(adminChatID, "Ошибка при получении списка пользователей."))
		return
	}

	sent, failed := 0, 0
	for _, u := range users {
		msg := tgbotapi.NewMessage(u.ID, "📢 <b>Сообщение от администратора:</b>\n\n"+text)
		msg.ParseMode = tgbotapi.ModeHTML
		if _, err := bot.Send(msg); err != nil {
			slog.Warn("broadcast: failed to send", "user_id", u.ID, "err", err)
			failed++
		} else {
			sent++
		}
	}

	report := fmt.Sprintf("✅ Рассылка завершена.\nОтправлено: %d\nОшибок: %d", sent, failed)
	send(bot, tgbotapi.NewMessage(adminChatID, report))
}

// ExecuteBroadcastToUser sends a message to a single user.
func ExecuteBroadcastToUser(
	ctx context.Context,
	bot *tgbotapi.BotAPI,
	adminChatID int64,
	targetUserID int64,
	text string,
) {
	msg := tgbotapi.NewMessage(targetUserID, "📢 <b>Сообщение от администратора:</b>\n\n"+text)
	msg.ParseMode = tgbotapi.ModeHTML
	if _, err := bot.Send(msg); err != nil {
		send(bot, tgbotapi.NewMessage(adminChatID, "❌ Не удалось отправить сообщение пользователю."))
		return
	}
	send(bot, tgbotapi.NewMessage(adminChatID, "✅ Сообщение отправлено."))
}
