package handler

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/EtoNeAnanasbI95/vpn-bot/internal/bot/keyboard"
	"github.com/EtoNeAnanasbI95/vpn-bot/internal/bot/session"
	"github.com/EtoNeAnanasbI95/vpn-bot/internal/domain"
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

// HandleBroadcastMenu shows the broadcast type selection.
func HandleBroadcastMenu(
	ctx context.Context,
	bot *tgbotapi.BotAPI,
	chatID int64,
	callbackID string,
) {
	answerCallback(bot, callbackID, "")
	msg := tgbotapi.NewMessage(chatID, "📢 <b>Рассылка</b>\n\nВыберите тип рассылки:")
	msg.ParseMode = tgbotapi.ModeHTML
	msg.ReplyMarkup = keyboard.BroadcastMenu()
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

// HandleBroadcastSelectUser shows user list to pick a single broadcast target.
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

// HandleBroadcastOpenSelect opens the multi-select keyboard.
func HandleBroadcastOpenSelect(
	ctx context.Context,
	bot *tgbotapi.BotAPI,
	chatID int64,
	callbackID string,
	senderID int64,
	sessions session.Store,
	uc *UseCases,
) {
	answerCallback(bot, callbackID, "")

	users, err := uc.User.GetAll(ctx)
	if err != nil || len(users) == 0 {
		send(bot, tgbotapi.NewMessage(chatID, "Нет зарегистрированных клиентов."))
		return
	}

	// Create a fresh selection session.
	sessions.Set(senderID, &session.Session{
		State: session.StateBroadcastSelect,
		Data:  map[string]string{session.KeyBcastSelectedIDs: ""},
	})

	sent, err := bot.Send(buildMultiSelectMsg(chatID, users, map[int64]bool{}))
	if err != nil {
		return
	}

	// Store the sent message ID so toggles can edit it.
	sess, _ := sessions.Get(senderID)
	sess.Data[session.KeyBcastMsgID] = fmt.Sprintf("%d", sent.MessageID)
	sessions.Set(senderID, sess)
}

// HandleBroadcastToggle toggles a user's selection and edits the keyboard in-place.
func HandleBroadcastToggle(
	ctx context.Context,
	bot *tgbotapi.BotAPI,
	chatID int64,
	callbackID string,
	msgID int,
	senderID int64,
	targetUserID int64,
	sessions session.Store,
	uc *UseCases,
) {
	sess, ok := sessions.Get(senderID)
	if !ok || sess.State != session.StateBroadcastSelect {
		answerCallback(bot, callbackID, "")
		return
	}

	selected := parseSelectedIDs(sess.Data[session.KeyBcastSelectedIDs])
	if selected[targetUserID] {
		delete(selected, targetUserID)
	} else {
		selected[targetUserID] = true
	}
	sess.Data[session.KeyBcastSelectedIDs] = formatSelectedIDs(selected)
	sessions.Set(senderID, sess)

	users, err := uc.User.GetAll(ctx)
	if err != nil {
		answerCallback(bot, callbackID, "")
		return
	}

	edit := tgbotapi.NewEditMessageReplyMarkup(chatID, msgID, keyboard.BroadcastMultiSelect(users, selected))
	bot.Send(edit) //nolint:errcheck
	answerCallback(bot, callbackID, "")
}

// HandleBroadcastConfirmSelect transitions to text input state.
func HandleBroadcastConfirmSelect(
	ctx context.Context,
	bot *tgbotapi.BotAPI,
	chatID int64,
	callbackID string,
	senderID int64,
	sessions session.Store,
) {
	sess, ok := sessions.Get(senderID)
	if !ok {
		answerCallback(bot, callbackID, "")
		return
	}

	selected := parseSelectedIDs(sess.Data[session.KeyBcastSelectedIDs])
	if len(selected) == 0 {
		answerCallback(bot, callbackID, "Выберите хотя бы одного клиента")
		return
	}

	answerCallback(bot, callbackID, "")
	sessions.Set(senderID, &session.Session{
		State: session.StateBroadcastSelected,
		Data:  map[string]string{session.KeyBcastSelectedIDs: formatSelectedIDs(selected)},
	})

	msg := tgbotapi.NewMessage(chatID, fmt.Sprintf(
		"📢 Введите текст рассылки для <b>%d</b> выбранных клиентов:", len(selected),
	))
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
	sendToUsers(bot, adminChatID, text, users)
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

// ExecuteBroadcastSelected sends a message to the stored set of selected users.
func ExecuteBroadcastSelected(
	ctx context.Context,
	bot *tgbotapi.BotAPI,
	adminChatID int64,
	text string,
	selectedIDs map[int64]bool,
	uc *UseCases,
) {
	var targets []*domain.User
	for id := range selectedIDs {
		u, err := uc.User.GetUser(ctx, id)
		if err != nil {
			continue
		}
		targets = append(targets, u)
	}
	sendToUsers(bot, adminChatID, text, targets)
}

// ── helpers ───────────────────────────────────────────────────────────────────

func sendToUsers(bot *tgbotapi.BotAPI, adminChatID int64, text string, users []*domain.User) {
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

func buildMultiSelectMsg(chatID int64, users []*domain.User, selected map[int64]bool) tgbotapi.MessageConfig {
	msg := tgbotapi.NewMessage(chatID, "☑️ <b>Выберите получателей рассылки:</b>")
	msg.ParseMode = tgbotapi.ModeHTML
	msg.ReplyMarkup = keyboard.BroadcastMultiSelect(users, selected)
	return msg
}

func parseSelectedIDs(s string) map[int64]bool {
	out := map[int64]bool{}
	if s == "" {
		return out
	}
	for _, part := range strings.Split(s, ",") {
		if id, err := strconv.ParseInt(strings.TrimSpace(part), 10, 64); err == nil {
			out[id] = true
		}
	}
	return out
}

func formatSelectedIDs(m map[int64]bool) string {
	parts := make([]string, 0, len(m))
	for id := range m {
		parts = append(parts, strconv.FormatInt(id, 10))
	}
	return strings.Join(parts, ",")
}
