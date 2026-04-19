package handler

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/EtoNeAnanasbI95/vpn-bot/internal/bot/callback"
	"github.com/EtoNeAnanasbI95/vpn-bot/internal/bot/keyboard"
	"github.com/EtoNeAnanasbI95/vpn-bot/internal/bot/session"
	"github.com/EtoNeAnanasbI95/vpn-bot/internal/domain"
)

// HandleAdminConnUsers shows all users for connection management selection.
func HandleAdminConnUsers(
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

	msg := tgbotapi.NewMessage(chatID, "🔗 <b>Управление подключениями</b>\n\nВыберите клиента:")
	msg.ParseMode = tgbotapi.ModeHTML
	msg.ReplyMarkup = keyboard.ConnUserListWithAdd(users, callback.AdmConnList)
	send(bot, msg)
}

// HandleAdminConnList shows connections for a specific user.
func HandleAdminConnList(
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
	if err != nil {
		slog.Error("admin: list connections", "target_user_id", targetUserID, "err", err)
		answerCallback(bot, callbackID, "Ошибка при получении подключений")
		return
	}

	slog.Info("admin: connections viewed", "target_user_id", targetUserID, "count", len(conns))

	name := user.DisplayName()
	if user.Username != "" {
		name += " (@" + user.Username + ")"
	}
	msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("🔗 <b>Подключения клиента %s</b>", name))
	msg.ParseMode = tgbotapi.ModeHTML
	msg.ReplyMarkup = keyboard.ConnectionManage(targetUserID, conns)
	send(bot, msg)
}

// HandleAdminConnToggle enables or disables a connection in 3x-ui.
func HandleAdminConnToggle(
	ctx context.Context,
	bot *tgbotapi.BotAPI,
	chatID int64,
	callbackID string,
	messageID int,
	clientUUID string,
	enable bool,
	uc *UseCases,
) {
	if err := uc.Connection.SetEnabled(ctx, clientUUID, enable); err != nil {
		slog.Error("admin: toggle connection", "uuid", clientUUID, "enable", enable, "err", err)
		answerCallback(bot, callbackID, "Ошибка при изменении статуса")
		return
	}

	slog.Info("admin: connection toggled", "uuid", clientUUID, "enable", enable)
	status := "❌ заблокировано"
	if enable {
		status = "✅ активировано"
	}
	answerCallback(bot, callbackID, fmt.Sprintf("Подключение %s", status))

	// Fetch the connection to learn the owner for list refresh.
	conn, err := uc.Connection.GetByUUID(ctx, clientUUID)
	if err != nil {
		return
	}
	refreshConnMessage(ctx, bot, chatID, messageID, conn.UserID, uc)
}

// HandleAdminConnDelete removes a connection from 3x-ui.
func HandleAdminConnDelete(
	ctx context.Context,
	bot *tgbotapi.BotAPI,
	chatID int64,
	callbackID string,
	messageID int,
	targetUserID int64,
	clientUUID string,
	uc *UseCases,
) {
	if err := uc.Connection.Remove(ctx, clientUUID); err != nil {
		slog.Error("admin: delete connection", "uuid", clientUUID, "target_user_id", targetUserID, "err", err)
		answerCallback(bot, callbackID, "Ошибка при удалении")
		return
	}
	slog.Info("admin: connection deleted", "uuid", clientUUID, "target_user_id", targetUserID)
	answerCallback(bot, callbackID, "🗑 Удалено")
	refreshConnMessage(ctx, bot, chatID, messageID, targetUserID, uc)
}

// HandleAdminConnAdd starts the multi-step flow for adding a connection.
func HandleAdminConnAdd(
	ctx context.Context,
	bot *tgbotapi.BotAPI,
	chatID int64,
	callbackID string,
	adminID int64,
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

	sessions.Set(adminID, &session.Session{
		State: session.StateAddConnLabel,
		Data: map[string]string{
			session.KeyConnUserID: fmt.Sprintf("%d", user.ID),
			session.KeyConnTgTag:  user.Username,
		},
	})

	msg := tgbotapi.NewMessage(chatID,
		fmt.Sprintf("➕ Добавление подключения для <b>%s</b>\n\nВведите название (например: iPhone, Ноутбук):\n\n<i>Клиент будет создан автоматически в 3x-ui на Reality-inbound.</i>", user.DisplayName()))
	msg.ParseMode = tgbotapi.ModeHTML
	msg.ReplyMarkup = keyboard.CancelKeyboard()
	send(bot, msg)
}

// HandleAdminConnCreateFinal creates the connection after admin chose payment type.
func HandleAdminConnCreateFinal(
	ctx context.Context,
	bot *tgbotapi.BotAPI,
	chatID int64,
	callbackID string,
	adminID int64,
	isFree bool,
	sessions session.Store,
	uc *UseCases,
) {
	sess, ok := sessions.Get(adminID)
	if !ok || sess.State != session.StateAddConnPaymentType {
		answerCallback(bot, callbackID, "Сессия устарела, начните заново")
		return
	}

	userIDStr := sess.Data[session.KeyConnUserID]
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		sessions.Clear(adminID)
		answerCallback(bot, callbackID, "Ошибка: неверный ID пользователя")
		return
	}

	label := sess.Data[session.KeyConnLabel]
	tgTag := sess.Data[session.KeyConnTgTag]
	sessions.Clear(adminID)

	answerCallback(bot, callbackID, "")
	send(bot, tgbotapi.NewMessage(chatID, "⏳ Создаю клиента в 3x-ui..."))

	conn, err := uc.Connection.Create(ctx, userID, adminID, tgTag, isFree)
	if err != nil {
		slog.Error("admin: create connection", "admin_id", adminID, "target_user_id", userID, "label", label, "err", err)
		send(bot, tgbotapi.NewMessage(chatID, fmt.Sprintf("❌ Ошибка: %s", err.Error())))
		return
	}

	slog.Info("admin: connection created", "admin_id", adminID, "target_user_id", userID, "label", conn.Label, "uuid", conn.UUID, "is_free", isFree)

	payType := "💳 платное"
	if isFree {
		payType = "🆓 бесплатное"
	}
	reply := tgbotapi.NewMessage(chatID,
		fmt.Sprintf("✅ Подключение <b>%s</b> создано (%s).\n\n<code>%s</code>", conn.Label, payType, conn.Link))
	reply.ParseMode = tgbotapi.ModeHTML
	send(bot, reply)

	// Notify the user.
	var userText string
	if isFree {
		userText = fmt.Sprintf("🎉 <b>Вам выдано новое подключение!</b>\n\n📛 Название: <b>%s</b>\n\n🔗 Ссылка:\n<code>%s</code>", conn.Label, conn.Link)
	} else {
		userText = fmt.Sprintf("🎉 <b>Вам выдано новое подключение!</b>\n\n📛 Название: <b>%s</b>\n💳 Требует оплаты\n\n🔗 Ссылка:\n<code>%s</code>\n\nОткройте раздел «🔗 Мои подключения» и нажмите <b>Оплатить</b>.", conn.Label, conn.Link)
	}
	userMsg := tgbotapi.NewMessage(userID, userText)
	userMsg.ParseMode = tgbotapi.ModeHTML
	bot.Send(userMsg) //nolint:errcheck
}

// HandleAdminConnNewUser starts the session for manually adding a user by Telegram ID.
func HandleAdminConnNewUser(
	ctx context.Context,
	bot *tgbotapi.BotAPI,
	chatID int64,
	callbackID string,
	adminID int64,
	sessions session.Store,
) {
	answerCallback(bot, callbackID, "")
	sessions.Set(adminID, &session.Session{
		State: session.StateAddManualUser,
		Data:  map[string]string{},
	})
	msg := tgbotapi.NewMessage(chatID, "👤 Введите <b>Telegram ID</b> нового клиента (число):\n\n<i>ID можно узнать через @userinfobot или аналогичные боты.</i>")
	msg.ParseMode = tgbotapi.ModeHTML
	msg.ReplyMarkup = keyboard.CancelKeyboard()
	send(bot, msg)
}

// HandleSessionAddManualUser processes the Telegram ID typed by the admin,
// registers the user in the DB and opens their connection list.
func HandleSessionAddManualUser(
	ctx context.Context,
	bot *tgbotapi.BotAPI,
	msg *tgbotapi.Message,
	uc *UseCases,
) {
	userID, err := strconv.ParseInt(msg.Text, 10, 64)
	if err != nil || userID <= 0 {
		reply := tgbotapi.NewMessage(msg.Chat.ID, "❌ Неверный формат. Введите числовой Telegram ID (например <code>123456789</code>):")
		reply.ParseMode = tgbotapi.ModeHTML
		reply.ReplyMarkup = keyboard.CancelKeyboard()
		send(bot, reply)
		return
	}

	tu := domain.TelegramUser{ID: userID}

	// Try to resolve name info from Telegram API.
	chat, err := bot.GetChat(tgbotapi.ChatInfoConfig{ChatConfig: tgbotapi.ChatConfig{ChatID: userID}})
	if err == nil {
		tu.Username = chat.UserName
		tu.FirstName = chat.FirstName
		tu.LastName = chat.LastName
	}

	user, _, err := uc.User.RegisterOrGet(ctx, tu)
	if err != nil {
		slog.Error("admin: manual add user", "target_id", userID, "err", err)
		send(bot, tgbotapi.NewMessage(msg.Chat.ID, "❌ Не удалось создать пользователя. Попробуйте ещё раз."))
		return
	}

	slog.Info("admin: user manually added", "target_id", userID, "username", user.Username)

	name := user.DisplayName()
	if user.Username != "" {
		name += " (@" + user.Username + ")"
	}
	notice := fmt.Sprintf("✅ Клиент <b>%s</b> добавлен.", name)
	if tu.Username == "" && tu.FirstName == "" {
		notice += "\n\n<i>Имя не удалось определить — оно обновится автоматически, когда клиент напишет боту.</i>"
	}
	noticeMsg := tgbotapi.NewMessage(msg.Chat.ID, notice)
	noticeMsg.ParseMode = tgbotapi.ModeHTML
	send(bot, noticeMsg)

	HandleAdminConnList(ctx, bot, msg.Chat.ID, "", user.ID, uc)
}

func refreshConnMessage(ctx context.Context, bot *tgbotapi.BotAPI, chatID int64, messageID int, userID int64, uc *UseCases) {
	conns, err := uc.Connection.ListForUser(ctx, userID)
	if err != nil {
		return
	}
	editMarkup(bot, chatID, messageID, keyboard.ConnectionManage(userID, conns))
}
