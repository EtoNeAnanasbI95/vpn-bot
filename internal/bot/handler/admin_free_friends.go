package handler

import (
	"context"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/EtoNeAnanasbI95/vpn-bot/internal/bot/keyboard"
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

