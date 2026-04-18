package handler

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/EtoNeAnanasbI95/vpn-bot/internal/bot/keyboard"
	"github.com/EtoNeAnanasbI95/vpn-bot/internal/bot/session"
	"github.com/EtoNeAnanasbI95/vpn-bot/internal/domain"
)

// HandleUserRequestConnection handles "🆕 Запросить подключение" text button from user.
// Creates a pending request and notifies all admins.
func HandleUserRequestConnection(
	ctx context.Context,
	bot *tgbotapi.BotAPI,
	chatID int64,
	userID int64,
	adminIDs []int64,
	uc *UseCases,
) {
	req, err := uc.ConnRequest.Create(ctx, userID)
	if err != nil {
		if err.Error() == "already_active" {
			send(bot, tgbotapi.NewMessage(chatID, "⏳ У вас уже есть активный запрос на подключение. Ожидайте ответа администратора."))
			return
		}
		slog.Error("conn_request: create", "user_id", userID, "err", err)
		send(bot, tgbotapi.NewMessage(chatID, "❌ Не удалось отправить запрос. Попробуйте позже."))
		return
	}

	user, err := uc.User.GetUser(ctx, userID)
	if err != nil {
		slog.Error("conn_request: get user", "user_id", userID, "err", err)
		return
	}

	name := user.DisplayName()
	if user.Username != "" {
		name += " (@" + user.Username + ")"
	}

	notifyText := fmt.Sprintf(
		"🔗 <b>Новый запрос на подключение!</b>\n\n👤 Пользователь: <b>%s</b>",
		name,
	)

	for _, adminID := range adminIDs {
		msg := tgbotapi.NewMessage(adminID, notifyText)
		msg.ParseMode = tgbotapi.ModeHTML
		msg.ReplyMarkup = keyboard.ConnRequestAdminMenu(req.UUID)
		send(bot, msg)
	}

	slog.Info("conn_request: created", "user_id", userID, "uuid", req.UUID)
	send(bot, tgbotapi.NewMessage(chatID, "✅ Запрос отправлен. Ожидайте ответа администратора."))
}

// HandleAdmReqFree processes admin's "Друг — бесплатно" choice.
// Atomically claims the request and immediately creates a free connection.
func HandleAdmReqFree(
	ctx context.Context,
	bot *tgbotapi.BotAPI,
	chatID int64,
	callbackID string,
	msgID int,
	adminID int64,
	reqUUID string,
	uc *UseCases,
) {
	claimed, err := uc.ConnRequest.Claim(ctx, reqUUID, adminID, domain.ConnReqCompleted)
	if err != nil {
		slog.Error("conn_request: claim free", "uuid", reqUUID, "err", err)
		answerCallback(bot, callbackID, "Ошибка, попробуйте ещё раз")
		return
	}
	if !claimed {
		answerCallback(bot, callbackID, "Запрос уже обработан другим администратором")
		return
	}
	answerCallback(bot, callbackID, "")

	req, err := uc.ConnRequest.GetByUUID(ctx, reqUUID)
	if err != nil {
		slog.Error("conn_request: get after claim", "uuid", reqUUID, "err", err)
		return
	}

	user, err := uc.User.GetUser(ctx, req.UserID)
	if err != nil {
		slog.Error("conn_request: get user", "user_id", req.UserID, "err", err)
		return
	}

	conn, err := uc.Connection.Create(ctx, req.UserID, adminID, user.Username, "VPN", true)
	if err != nil {
		slog.Error("conn_request: create connection (free)", "user_id", req.UserID, "err", err)
		// Notify admin of failure
		send(bot, tgbotapi.NewMessage(chatID, fmt.Sprintf("❌ Ошибка при создании подключения: %s", err.Error())))
		return
	}

	slog.Info("conn_request: free connection issued", "user_id", req.UserID, "uuid", conn.UUID)

	// Notify user
	userMsg := tgbotapi.NewMessage(req.UserID,
		fmt.Sprintf("🎉 <b>Ваш запрос одобрен!</b>\n\n🔗 Ссылка на подключение:\n<code>%s</code>", conn.Link))
	userMsg.ParseMode = tgbotapi.ModeHTML
	send(bot, userMsg)

	// Update admin message
	name := user.DisplayName()
	if user.Username != "" {
		name += " (@" + user.Username + ")"
	}
	editTextAndMarkup(bot, chatID, msgID,
		fmt.Sprintf("✅ <b>Выдано бесплатно</b>\n\n👤 %s\n🔗 %s", name, conn.Link),
		tgbotapi.InlineKeyboardMarkup{},
	)
}

// HandleAdmReqPaid processes admin's "Платно" choice.
// Atomically claims the request and shows price selection.
func HandleAdmReqPaid(
	ctx context.Context,
	bot *tgbotapi.BotAPI,
	chatID int64,
	callbackID string,
	msgID int,
	adminID int64,
	reqUUID string,
	uc *UseCases,
) {
	claimed, err := uc.ConnRequest.Claim(ctx, reqUUID, adminID, domain.ConnReqAwaitingPayment)
	if err != nil {
		slog.Error("conn_request: claim paid", "uuid", reqUUID, "err", err)
		answerCallback(bot, callbackID, "Ошибка, попробуйте ещё раз")
		return
	}
	if !claimed {
		answerCallback(bot, callbackID, "Запрос уже обработан другим администратором")
		return
	}
	answerCallback(bot, callbackID, "")

	edit := tgbotapi.NewEditMessageReplyMarkup(chatID, msgID, keyboard.ConnRequestPriceMenu(reqUUID))
	bot.Send(edit) //nolint:errcheck
}

// HandleAdmReqPriceBase sets the base price (300₽) and sends payment details to the user.
func HandleAdmReqPriceBase(
	ctx context.Context,
	bot *tgbotapi.BotAPI,
	chatID int64,
	callbackID string,
	msgID int,
	adminID int64,
	reqUUID string,
	uc *UseCases,
) {
	answerCallback(bot, callbackID, "")
	sendPaymentDetails(ctx, bot, chatID, msgID, adminID, reqUUID, 300, uc)
}

// HandleAdmReqPriceCustom starts the session for the admin to type a custom price.
func HandleAdmReqPriceCustom(
	ctx context.Context,
	bot *tgbotapi.BotAPI,
	chatID int64,
	callbackID string,
	adminID int64,
	reqUUID string,
	sessions session.Store,
) {
	answerCallback(bot, callbackID, "")

	sessions.Set(adminID, &session.Session{
		State: session.StateAdmReqCustomPrice,
		Data:  map[string]string{session.KeyConnReqUUID: reqUUID},
	})

	msg := tgbotapi.NewMessage(chatID, "✏️ Введите сумму оплаты (только число, например <code>500</code>):")
	msg.ParseMode = tgbotapi.ModeHTML
	msg.ReplyMarkup = keyboard.CancelKeyboard()
	send(bot, msg)
}

// HandleSessionAdmReqCustomPrice processes the admin's custom price input.
func HandleSessionAdmReqCustomPrice(
	ctx context.Context,
	bot *tgbotapi.BotAPI,
	msg *tgbotapi.Message,
	sess *session.Session,
	uc *UseCases,
) {
	amount, err := strconv.Atoi(msg.Text)
	if err != nil || amount <= 0 {
		reply := tgbotapi.NewMessage(msg.Chat.ID, "❌ Введите корректную сумму (целое число больше нуля):")
		reply.ReplyMarkup = keyboard.CancelKeyboard()
		send(bot, reply)
		return
	}

	reqUUID := sess.Data[session.KeyConnReqUUID]
	sendPaymentDetails(ctx, bot, msg.Chat.ID, 0, msg.From.ID, reqUUID, amount, uc)
}

// HandleConnReqCheckPay handles user's "Я оплатил — проверить" button.
// Notifies the admin who handled the request.
func HandleConnReqCheckPay(
	ctx context.Context,
	bot *tgbotapi.BotAPI,
	chatID int64,
	callbackID string,
	msgID int,
	reqUUID string,
	uc *UseCases,
) {
	req, err := uc.ConnRequest.GetByUUID(ctx, reqUUID)
	if err != nil {
		answerCallback(bot, callbackID, "Запрос не найден")
		return
	}

	if req.Status != domain.ConnReqAwaitingPayment {
		answerCallback(bot, callbackID, "Запрос уже обработан")
		return
	}

	if err := uc.ConnRequest.MarkPaymentPending(ctx, reqUUID); err != nil {
		slog.Error("conn_request: mark payment pending", "uuid", reqUUID, "err", err)
		answerCallback(bot, callbackID, "Ошибка, попробуйте ещё раз")
		return
	}
	answerCallback(bot, callbackID, "")

	user, err := uc.User.GetUser(ctx, req.UserID)
	if err != nil {
		slog.Error("conn_request: get user for check pay", "user_id", req.UserID, "err", err)
		return
	}

	name := user.DisplayName()
	if user.Username != "" {
		name += " (@" + user.Username + ")"
	}

	// Notify the admin who claimed the request
	adminMsg := tgbotapi.NewMessage(req.AdminID,
		fmt.Sprintf("💳 <b>Пользователь %s</b> сообщает об оплате <b>%d₽</b>.\n\nПодтвердите, чтобы выдать подключение:", name, req.Amount))
	adminMsg.ParseMode = tgbotapi.ModeHTML
	adminMsg.ReplyMarkup = keyboard.ConnRequestConfirmPayButton(reqUUID)
	send(bot, adminMsg)

	// Update user message button to prevent re-clicks
	edit := tgbotapi.NewEditMessageReplyMarkup(chatID, msgID, tgbotapi.InlineKeyboardMarkup{})
	bot.Send(edit) //nolint:errcheck

	send(bot, tgbotapi.NewMessage(chatID, "⏳ Запрос отправлен администратору. Ожидайте подтверждения."))
}

// HandleAdmReqConfirmPay processes admin's payment confirmation.
// Creates the connection and sends the link to the user.
func HandleAdmReqConfirmPay(
	ctx context.Context,
	bot *tgbotapi.BotAPI,
	chatID int64,
	callbackID string,
	msgID int,
	adminID int64,
	reqUUID string,
	uc *UseCases,
) {
	req, err := uc.ConnRequest.GetByUUID(ctx, reqUUID)
	if err != nil {
		answerCallback(bot, callbackID, "Запрос не найден")
		return
	}

	if req.Status != domain.ConnReqPaymentPending {
		answerCallback(bot, callbackID, "Оплата уже подтверждена")
		return
	}

	answerCallback(bot, callbackID, "")
	send(bot, tgbotapi.NewMessage(chatID, "⏳ Создаю подключение..."))

	user, err := uc.User.GetUser(ctx, req.UserID)
	if err != nil {
		slog.Error("conn_request: get user for confirm", "user_id", req.UserID, "err", err)
		return
	}

	conn, err := uc.Connection.Create(ctx, req.UserID, adminID, user.Username, "VPN", true)
	if err != nil {
		slog.Error("conn_request: create connection (paid)", "user_id", req.UserID, "err", err)
		send(bot, tgbotapi.NewMessage(chatID, fmt.Sprintf("❌ Ошибка при создании подключения: %s", err.Error())))
		return
	}

	if err := uc.ConnRequest.Complete(ctx, reqUUID); err != nil {
		slog.Warn("conn_request: complete", "uuid", reqUUID, "err", err)
	}

	slog.Info("conn_request: paid connection issued", "user_id", req.UserID, "uuid", conn.UUID, "amount", req.Amount)

	// Notify user
	userMsg := tgbotapi.NewMessage(req.UserID,
		fmt.Sprintf("🎉 <b>Оплата подтверждена!</b>\n\n🔗 Ссылка на подключение:\n<code>%s</code>", conn.Link))
	userMsg.ParseMode = tgbotapi.ModeHTML
	send(bot, userMsg)

	// Update admin message
	name := user.DisplayName()
	if user.Username != "" {
		name += " (@" + user.Username + ")"
	}
	editTextAndMarkup(bot, chatID, msgID,
		fmt.Sprintf("✅ <b>Оплата подтверждена, подключение выдано</b>\n\n👤 %s\n💰 %d₽\n🔗 %s", name, req.Amount, conn.Link),
		tgbotapi.InlineKeyboardMarkup{},
	)
}

// ── helpers ───────────────────────────────────────────────────────────────────

// sendPaymentDetails sets the amount, fetches admin payment info, and sends
// the payment message to the user with "check payment" button.
func sendPaymentDetails(
	ctx context.Context,
	bot *tgbotapi.BotAPI,
	adminChatID int64,
	adminMsgID int,
	adminID int64,
	reqUUID string,
	amount int,
	uc *UseCases,
) {
	if err := uc.ConnRequest.SetAmount(ctx, reqUUID, amount); err != nil {
		slog.Error("conn_request: set amount", "uuid", reqUUID, "err", err)
		send(bot, tgbotapi.NewMessage(adminChatID, "❌ Ошибка при сохранении суммы."))
		return
	}

	payInfo, err := uc.Connection.GetAdminOwnPaymentInfo(ctx, adminID)
	if err != nil || payInfo == "" {
		send(bot, tgbotapi.NewMessage(adminChatID, "❌ У вас не заданы реквизиты. Укажите их через «💰 Мои реквизиты» в панели администратора."))
		return
	}

	req, err := uc.ConnRequest.GetByUUID(ctx, reqUUID)
	if err != nil {
		slog.Error("conn_request: get for payment details", "uuid", reqUUID, "err", err)
		return
	}

	userMsg := tgbotapi.NewMessage(req.UserID, fmt.Sprintf(
		"💳 <b>Оплата подключения</b>\n\nСумма: <b>%d₽</b>\n\nРеквизиты для оплаты:\n<code>%s</code>\n\nПосле оплаты нажмите кнопку ниже:",
		amount, payInfo,
	))
	userMsg.ParseMode = tgbotapi.ModeHTML
	userMsg.ReplyMarkup = keyboard.ConnRequestCheckPayButton(reqUUID)
	send(bot, userMsg)

	// Update admin message
	if adminMsgID != 0 {
		editTextAndMarkup(bot, adminChatID, adminMsgID,
			fmt.Sprintf("✅ Реквизиты отправлены пользователю. Сумма: <b>%d₽</b>", amount),
			tgbotapi.InlineKeyboardMarkup{},
		)
	} else {
		reply := tgbotapi.NewMessage(adminChatID, fmt.Sprintf("✅ Реквизиты отправлены пользователю. Сумма: <b>%d₽</b>", amount))
		reply.ParseMode = tgbotapi.ModeHTML
		send(bot, reply)
	}
}

func editTextAndMarkup(bot *tgbotapi.BotAPI, chatID int64, msgID int, text string, markup tgbotapi.InlineKeyboardMarkup) {
	edit := tgbotapi.NewEditMessageText(chatID, msgID, text)
	edit.ParseMode = tgbotapi.ModeHTML
	edit.ReplyMarkup = &markup
	bot.Send(edit) //nolint:errcheck
}
