package handler

import (
	"context"
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/EtoNeAnanasbI95/vpn-bot/internal/bot/keyboard"
	"github.com/EtoNeAnanasbI95/vpn-bot/internal/bot/session"
)

// ── Free friends ──────────────────────────────────────────────────────────────

// HandleAdminFreeFriendList shows current free friends with remove buttons.
func HandleAdminFreeFriendList(
	ctx context.Context,
	bot *tgbotapi.BotAPI,
	chatID int64,
	callbackID string,
	uc *UseCases,
) {
	answerCallback(bot, callbackID, "")

	friends, err := uc.User.GetFreeFriends(ctx)
	if err != nil {
		send(bot, tgbotapi.NewMessage(chatID, "Ошибка при загрузке списка друзей."))
		return
	}

	text := "💚 <b>Друзья</b>\n\nДрузья не получают напоминаний об оплате."
	if len(friends) == 0 {
		text += "\n\nСписок пуст."
	}
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = tgbotapi.ModeHTML
	msg.ReplyMarkup = keyboard.FreeFriendList(friends)
	send(bot, msg)
}

// HandleAdminFreeFriendAdd shows all non-friend users so admin can add one.
func HandleAdminFreeFriendAdd(
	ctx context.Context,
	bot *tgbotapi.BotAPI,
	chatID int64,
	callbackID string,
	uc *UseCases,
) {
	answerCallback(bot, callbackID, "")

	all, err := uc.User.GetAll(ctx)
	if err != nil {
		send(bot, tgbotapi.NewMessage(chatID, "Ошибка при загрузке клиентов."))
		return
	}

	var nonFriends = all[:0]
	for _, u := range all {
		if !u.IsFreeFriend {
			nonFriends = append(nonFriends, u)
		}
	}

	if len(nonFriends) == 0 {
		send(bot, tgbotapi.NewMessage(chatID, "Все клиенты уже являются друзьями."))
		return
	}

	msg := tgbotapi.NewMessage(chatID, "Выберите клиента для добавления в друзья:")
	msg.ParseMode = tgbotapi.ModeHTML
	msg.ReplyMarkup = keyboard.FreeFriendAddList(nonFriends)
	send(bot, msg)
}

// HandleAdminFreeFriendToggle toggles is_free_friend and returns to the friends list.
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

	HandleAdminFreeFriendList(ctx, bot, chatID, "", uc)
}

// ── Pay dates ─────────────────────────────────────────────────────────────────

// HandleAdminPayDateList shows all users so admin can drill into a user's connections.
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

	msg := tgbotapi.NewMessage(chatID, "📅 <b>Даты оплат</b>\n\nВыберите клиента, чтобы увидеть его подключения:")
	msg.ParseMode = tgbotapi.ModeHTML
	msg.ReplyMarkup = keyboard.PayDateUserList(users)
	send(bot, msg)
}

// HandleAdminPayDateUser shows connections for a user so admin can select one.
func HandleAdminPayDateUser(
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

	conns, err := uc.Connection.ListForUser(ctx, targetUserID)
	if err != nil || len(conns) == 0 {
		send(bot, tgbotapi.NewMessage(chatID, "У этого клиента нет подключений."))
		return
	}

	var items []keyboard.PayDateConn
	for _, c := range conns {
		items = append(items, keyboard.PayDateConn{UUID: c.UUID, Label: c.Label})
	}

	name := user.DisplayName()
	if user.Username != "" {
		name += " (@" + user.Username + ")"
	}
	msg := tgbotapi.NewMessage(chatID, fmt.Sprintf(
		"📅 Подключения клиента <b>%s</b>\n\nВыберите подключение для установки даты оплаты:", name,
	))
	msg.ParseMode = tgbotapi.ModeHTML
	msg.ReplyMarkup = keyboard.PayDateConnList(items, targetUserID)
	send(bot, msg)
}

// HandleAdminPayDateConn starts the session for the admin to type a pay date
// for a specific connection UUID.
func HandleAdminPayDateConn(
	ctx context.Context,
	bot *tgbotapi.BotAPI,
	chatID int64,
	callbackID string,
	adminUserID int64,
	targetUserID int64,
	connUUID string,
	sessions session.Store,
	uc *UseCases,
) {
	answerCallback(bot, callbackID, "")

	sessions.Set(adminUserID, &session.Session{
		State: session.StateSetPayDate,
		Data: map[string]string{
			session.KeyPayDateConnUUID:    connUUID,
			session.KeyPayDateConnUserID:  fmt.Sprintf("%d", targetUserID),
			session.KeyPayDateConnAdminID: fmt.Sprintf("%d", adminUserID),
		},
	})

	msg := tgbotapi.NewMessage(chatID, "📅 Введите дату оплаты в формате <code>ДД.ММ.ГГГГ</code>:")
	msg.ParseMode = tgbotapi.ModeHTML
	send(bot, msg)
}
