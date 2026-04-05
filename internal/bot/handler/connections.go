package handler

import (
	"context"
	"fmt"
	"log/slog"


	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/EtoNeAnanasbI95/vpn-bot/internal/bot/keyboard"
	"github.com/EtoNeAnanasbI95/vpn-bot/internal/bot/middleware"
)

// HandleConnectionList shows the user's VPN connections.
func HandleConnectionList(
	ctx context.Context,
	bot *tgbotapi.BotAPI,
	chatID int64,
	callbackID string,
	uc *UseCases,
) {
	u := middleware.UserFromCtx(ctx)

	conns, err := uc.Connection.ListForUser(ctx, u.ID)
	if err != nil {
		slog.Error("connections: list", "user_id", u.ID, "username", u.Username, "err", err)
		answerCallback(bot, callbackID, "")
		send(bot, tgbotapi.NewMessage(chatID, "Ошибка при получении подключений."))
		return
	}

	slog.Info("connections: list viewed", "user_id", u.ID, "username", u.Username, "count", len(conns))
	answerCallback(bot, callbackID, "")

	if len(conns) == 0 {
		send(bot, tgbotapi.NewMessage(chatID, "У вас пока нет подключений. Обратитесь к администратору."))
		return
	}

	msg := tgbotapi.NewMessage(chatID, "🔗 <b>Ваши подключения</b>\n\nНажмите на подключение для получения QR-кода:")
	msg.ParseMode = tgbotapi.ModeHTML
	msg.ReplyMarkup = keyboard.ConnectionList(conns)
	send(bot, msg)
}

// HandleConnectionQR generates and sends a QR code for a connection.
func HandleConnectionQR(
	ctx context.Context,
	bot *tgbotapi.BotAPI,
	chatID int64,
	callbackID string,
	clientUUID string,
	uc *UseCases,
) {
	u := middleware.UserFromCtx(ctx)
	conn, err := uc.Connection.GetByUUID(ctx, clientUUID)
	if err != nil {
		slog.Warn("connections: qr not found", "user_id", u.ID, "uuid", clientUUID, "err", err)
		answerCallback(bot, callbackID, "Подключение не найдено")
		return
	}

	// Verify ownership — user can only see their own connections.
	if conn.UserID != u.ID {
		slog.Warn("connections: unauthorized qr attempt", "user_id", u.ID, "owner_id", conn.UserID, "uuid", clientUUID)
		answerCallback(bot, callbackID, "Нет доступа")
		return
	}

	slog.Info("connections: qr requested", "user_id", u.ID, "username", u.Username, "label", conn.Label)
	answerCallback(bot, callbackID, "")

	png, err := uc.Connection.GenerateQR(ctx, conn.Link)
	if err != nil {
		slog.Error("connections: qr generate", "user_id", u.ID, "uuid", clientUUID, "err", err)
		send(bot, tgbotapi.NewMessage(chatID, "Ошибка при генерации QR-кода."))
		return
	}

	status := "✅ Активно"
	if !conn.IsActive {
		status = "❌ Заблокировано"
	}
	caption := fmt.Sprintf("<b>%s</b>\nСтатус: %s\n\n<code>%s</code>", conn.Label, status, conn.Link)

	photo := tgbotapi.NewPhoto(chatID, tgbotapi.FileBytes{Name: "qr.png", Bytes: png})
	photo.Caption = caption
	photo.ParseMode = tgbotapi.ModeHTML
	send(bot, photo)
}

// HandleConnectionPay shows the admin's payment credentials to the user.
func HandleConnectionPay(
	ctx context.Context,
	bot *tgbotapi.BotAPI,
	chatID int64,
	callbackID string,
	connUUID string,
	uc *UseCases,
) {
	u := middleware.UserFromCtx(ctx)
	answerCallback(bot, callbackID, "")

	info, err := uc.Connection.GetAdminPaymentInfo(ctx, connUUID)
	if err != nil {
		slog.Warn("connections: get payment info", "user_id", u.ID, "uuid", connUUID, "err", err)
		send(bot, tgbotapi.NewMessage(chatID, "❌ Реквизиты для оплаты ещё не настроены администратором."))
		return
	}

	msg := tgbotapi.NewMessage(chatID, fmt.Sprintf(
		"💳 <b>Реквизиты для оплаты</b>\n\n%s\n\nПосле оплаты нажмите кнопку ниже:", info,
	))
	msg.ParseMode = tgbotapi.ModeHTML
	msg.ReplyMarkup = keyboard.ConnPayButton(connUUID)
	send(bot, msg)
}

// HandleConnectionPaid marks connection as pending, removes the "I paid" button, and notifies the admin.
func HandleConnectionPaid(
	ctx context.Context,
	bot *tgbotapi.BotAPI,
	chatID int64,
	callbackID string,
	messageID int,
	connUUID string,
	uc *UseCases,
) {
	u := middleware.UserFromCtx(ctx)

	if err := uc.Connection.SetPaymentPending(ctx, connUUID); err != nil {
		slog.Error("connections: set pending", "user_id", u.ID, "uuid", connUUID, "err", err)
		answerCallback(bot, callbackID, "Ошибка")
		return
	}

	slog.Info("connections: user claims paid", "user_id", u.ID, "username", u.Username, "uuid", connUUID)
	answerCallback(bot, callbackID, "")

	// Remove the "I paid" button from the message.
	bot.Request(tgbotapi.NewEditMessageReplyMarkup(chatID, messageID, tgbotapi.InlineKeyboardMarkup{})) //nolint:errcheck

	send(bot, tgbotapi.NewMessage(chatID, "⏳ Оплата отправлена на подтверждение администратору."))

	// Notify the admin who issued this connection.
	conn, err := uc.Connection.GetByUUID(ctx, connUUID)
	if err != nil || conn.AdminID == 0 {
		return
	}

	name := u.DisplayName()
	if u.Username != "" {
		name += " (@" + u.Username + ")"
	}
	adminMsg := tgbotapi.NewMessage(conn.AdminID,
		fmt.Sprintf("💳 <b>Заявка на оплату</b>\n\nПользователь <b>%s</b> сообщил об оплате подключения <b>%s</b>.", name, conn.Label))
	adminMsg.ParseMode = tgbotapi.ModeHTML
	adminMsg.ReplyMarkup = keyboard.ConnPayConfirmButton(connUUID)
	bot.Send(adminMsg) //nolint:errcheck
}

// HandleAdminConfirmConnPayment confirms a connection payment from the admin side.
func HandleAdminConfirmConnPayment(
	ctx context.Context,
	bot *tgbotapi.BotAPI,
	chatID int64,
	callbackID string,
	messageID int,
	connUUID string,
	uc *UseCases,
) {
	admin := middleware.UserFromCtx(ctx)

	userID, err := uc.Connection.ConfirmConnPayment(ctx, connUUID)
	if err != nil {
		slog.Error("connections: confirm payment", "admin_id", admin.ID, "uuid", connUUID, "err", err)
		answerCallback(bot, callbackID, "Ошибка при подтверждении")
		return
	}

	slog.Info("connections: payment confirmed", "admin_id", admin.ID, "user_id", userID, "uuid", connUUID)
	answerCallback(bot, callbackID, "✅ Оплата подтверждена")

	// Remove the confirm button from admin's message.
	bot.Request(tgbotapi.NewEditMessageReplyMarkup(chatID, messageID, tgbotapi.InlineKeyboardMarkup{})) //nolint:errcheck

	// Notify the user.
	userMsg := tgbotapi.NewMessage(userID, "✅ <b>Оплата подтверждена!</b>\n\nСпасибо.")
	userMsg.ParseMode = tgbotapi.ModeHTML
	bot.Send(userMsg) //nolint:errcheck
}

// HandleConnectionListFromMessage handles the "🔗 Мои подключения" text button.
func HandleConnectionListFromMessage(
	ctx context.Context,
	bot *tgbotapi.BotAPI,
	chatID int64,
	uc *UseCases,
) {
	u := middleware.UserFromCtx(ctx)

	conns, err := uc.Connection.ListForUser(ctx, u.ID)
	if err != nil {
		slog.Error("connections: list (message)", "user_id", u.ID, "username", u.Username, "err", err)
		send(bot, tgbotapi.NewMessage(chatID, "Ошибка при получении подключений."))
		return
	}
	slog.Info("connections: list viewed", "user_id", u.ID, "username", u.Username, "count", len(conns))

	if len(conns) == 0 {
		send(bot, tgbotapi.NewMessage(chatID, "У вас пока нет подключений. Обратитесь к администратору."))
		return
	}

	msg := tgbotapi.NewMessage(chatID, "🔗 <b>Ваши подключения</b>\n\nНажмите на подключение для получения QR-кода:")
	msg.ParseMode = tgbotapi.ModeHTML
	msg.ReplyMarkup = keyboard.ConnectionList(conns)
	send(bot, msg)
}
