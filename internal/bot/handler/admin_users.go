package handler

import (
	"context"
	"fmt"
	"log/slog"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/EtoNeAnanasbI95/vpn-bot/internal/bot/callback"
	"github.com/EtoNeAnanasbI95/vpn-bot/internal/bot/keyboard"
	"github.com/EtoNeAnanasbI95/vpn-bot/internal/bot/middleware"
)

// HandleAdminUserList shows all registered users.
func HandleAdminUserList(
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

	msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("👥 <b>Клиенты</b> (%d):", len(users)))
	msg.ParseMode = tgbotapi.ModeHTML
	msg.ReplyMarkup = keyboard.UserListForAction(users, func(uid int64) string {
		return callback.Encode("adm_user_detail", fmt.Sprintf("%d", uid))
	})
	send(bot, msg)
}

// HandleAdminUserDetail shows actions for a specific user.
func HandleAdminUserDetail(
	ctx context.Context,
	bot *tgbotapi.BotAPI,
	chatID int64,
	callbackID string,
	targetUserID int64,
	uc *UseCases,
) {
	user, err := uc.User.GetUser(ctx, targetUserID)
	if err != nil {
		answerCallback(bot, callbackID, "Пользователь не найден")
		return
	}
	answerCallback(bot, callbackID, "")

	text := fmt.Sprintf(
		"👤 <b>%s</b>\n@%s\nID: <code>%d</code>",
		user.DisplayName(), user.Username, user.ID,
	)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = tgbotapi.ModeHTML
	msg.ReplyMarkup = keyboard.UserDetail(user)
	send(bot, msg)
}

// HandleAdminDeleteUser removes a user and all their 3x-ui connections.
func HandleAdminDeleteUser(
	ctx context.Context,
	bot *tgbotapi.BotAPI,
	chatID int64,
	callbackID string,
	targetUserID int64,
	uc *UseCases,
) {
	admin := middleware.UserFromCtx(ctx)

	// Delete all 3x-ui connections for this user first.
	conns, err := uc.Connection.ListForUser(ctx, targetUserID)
	if err != nil {
		slog.Warn("admin: delete user — could not list connections", "target_user_id", targetUserID, "err", err)
	}
	for _, c := range conns {
		if err := uc.Connection.Remove(ctx, c.UUID); err != nil {
			slog.Warn("admin: delete user — could not remove connection", "uuid", c.UUID, "err", err)
		}
	}

	if err := uc.User.DeleteUser(ctx, targetUserID); err != nil {
		slog.Error("admin: delete user", "admin_id", admin.ID, "target_user_id", targetUserID, "err", err)
		answerCallback(bot, callbackID, "Ошибка при удалении клиента")
		return
	}

	slog.Info("admin: user deleted", "admin_id", admin.ID, "target_user_id", targetUserID, "connections_removed", len(conns))
	answerCallback(bot, callbackID, "🗑 Клиент удалён")

	// Show updated user list.
	users, err := uc.User.GetAll(ctx)
	if err != nil || len(users) == 0 {
		send(bot, tgbotapi.NewMessage(chatID, "Нет зарегистрированных клиентов."))
		return
	}
	msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("👥 <b>Клиенты</b> (%d):", len(users)))
	msg.ParseMode = tgbotapi.ModeHTML
	msg.ReplyMarkup = keyboard.UserListForAction(users, func(uid int64) string {
		return callback.Encode("adm_user_detail", fmt.Sprintf("%d", uid))
	})
	send(bot, msg)
}

func refreshUserDetailMessage(ctx context.Context, bot *tgbotapi.BotAPI, chatID int64, messageID int, userID int64, uc *UseCases) {
	user, err := uc.User.GetUser(ctx, userID)
	if err != nil {
		return
	}
	editMarkup(bot, chatID, messageID, keyboard.UserDetail(user))
}
