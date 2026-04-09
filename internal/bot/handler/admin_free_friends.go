package handler

import (
	"context"
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/EtoNeAnanasbI95/vpn-bot/internal/bot/keyboard"
	"github.com/EtoNeAnanasbI95/vpn-bot/internal/bot/session"
)

// HandleAdminFreeFriendList shows all users with their free-friend status.
// Clicking a user will toggle their status.
func HandleAdminFreeFriendList(
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

	msg := tgbotapi.NewMessage(chatID, "💚 <b>Управление друзьями</b>\n\nДрузья не получают напоминаний об оплате.\nНажмите на пользователя, чтобы изменить статус.")
	msg.ParseMode = tgbotapi.ModeHTML
	msg.ReplyMarkup = keyboard.FreeFriendList(users)
	send(bot, msg)
}

// HandleAdminFreeFriendToggle toggles the free-friend status for a user.
func HandleAdminFreeFriendToggle(
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

	newStatus := !user.IsFreeFriend
	if err := uc.User.SetFreeFriend(ctx, targetUserID, newStatus); err != nil {
		answerCallback(bot, callbackID, "Ошибка при изменении статуса")
		return
	}

	if newStatus {
		answerCallback(bot, callbackID, "💚 Добавлен в друзья")
	} else {
		answerCallback(bot, callbackID, "❌ Убран из друзей")
	}

	// Refresh the list.
	HandleAdminFreeFriendList(ctx, bot, chatID, "", uc)
}

// HandleAdminPayDateList shows all users for pay-date management.
func HandleAdminPayDateList(
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

	msg := tgbotapi.NewMessage(chatID, "📅 <b>Даты оплат</b>\n\nВыберите клиента для установки даты оплаты.\nЧерез месяц после указанной даты бот отправит ему напоминание.")
	msg.ParseMode = tgbotapi.ModeHTML
	msg.ReplyMarkup = keyboard.PayDateList(users)
	send(bot, msg)
}

// HandleAdminPayDateUser starts a session for the admin to enter a pay date for
// the selected user.
func HandleAdminPayDateUser(
	ctx context.Context,
	bot *tgbotapi.BotAPI,
	chatID int64,
	callbackID string,
	adminUserID int64,
	targetUserID int64,
	sessions session.Store,
	uc *UseCases,
) {
	user, err := uc.User.GetUser(ctx, targetUserID)
	if err != nil {
		answerCallback(bot, callbackID, "Пользователь не найден")
		return
	}
	answerCallback(bot, callbackID, "")

	sessions.Set(adminUserID, &session.Session{
		State: session.StateSetPayDate,
		Data:  map[string]string{session.KeyPayDateUserID: fmt.Sprintf("%d", targetUserID)},
	})

	name := user.DisplayName()
	if user.Username != "" {
		name += " (@" + user.Username + ")"
	}
	msg := tgbotapi.NewMessage(chatID, fmt.Sprintf(
		"📅 Введите дату оплаты для <b>%s</b> в формате <code>ДД.ММ.ГГГГ</code>:",
		name,
	))
	msg.ParseMode = tgbotapi.ModeHTML
	send(bot, msg)
}
