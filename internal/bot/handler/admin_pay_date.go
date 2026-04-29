package handler

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/EtoNeAnanasbI95/vpn-bot/internal/bot/keyboard"
	"github.com/EtoNeAnanasbI95/vpn-bot/internal/bot/session"
)

// HandleAdminPayDateUsers shows the user list for pay-date assignment.
func HandleAdminPayDateUsers(
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

	msg := tgbotapi.NewMessage(chatID, "📅 <b>Даты оплат</b>\n\nВыберите клиента:")
	msg.ParseMode = tgbotapi.ModeHTML
	msg.ReplyMarkup = keyboard.PayDateUserList(users)
	send(bot, msg)
}

// HandleAdminPayDateConns shows connections for a user so the admin can pick one to set a date.
func HandleAdminPayDateConns(
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
		slog.Error("pay date: list connections", "target_user_id", targetUserID, "err", err)
		send(bot, tgbotapi.NewMessage(chatID, "Ошибка при получении подключений."))
		return
	}
	if len(conns) == 0 {
		send(bot, tgbotapi.NewMessage(chatID, "У клиента нет подключений."))
		return
	}

	name := user.DisplayName()
	if user.Username != "" {
		name += " (@" + user.Username + ")"
	}
	msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("📅 <b>Даты оплат — %s</b>\n\nВыберите подключение:", name))
	msg.ParseMode = tgbotapi.ModeHTML
	msg.ReplyMarkup = keyboard.PayDateConnList(targetUserID, conns)
	send(bot, msg)
}

// HandleAdminPayDateSetStart begins the date-input session for a connection.
func HandleAdminPayDateSetStart(
	ctx context.Context,
	bot *tgbotapi.BotAPI,
	chatID int64,
	callbackID string,
	adminID int64,
	connUUID string,
	sessions session.Store,
	uc *UseCases,
) {
	conn, err := uc.Connection.GetByUUID(ctx, connUUID)
	if err != nil {
		answerCallback(bot, callbackID, "Подключение не найдено")
		return
	}
	answerCallback(bot, callbackID, "")

	sessions.Set(adminID, &session.Session{
		State: session.StateSetPayDate,
		Data: map[string]string{
			session.KeyPayDateConnUUID: connUUID,
			session.KeyPayDateUserID:   fmt.Sprintf("%d", conn.UserID),
		},
	})

	current := "не задана"
	if conn.LastPaidAt != nil {
		current = conn.LastPaidAt.Format("02.01.2006")
	}
	msg := tgbotapi.NewMessage(chatID, fmt.Sprintf(
		"📅 <b>Дата оплаты для подключения «%s»</b>\n\nТекущая дата: <b>%s</b>\n\nВведите день месяца (1–28), в который нужно присылать напоминание об оплате:",
		conn.Label, current,
	))
	msg.ParseMode = tgbotapi.ModeHTML
	msg.ReplyMarkup = keyboard.CancelKeyboard()
	send(bot, msg)
}

// HandleSessionSetPayDate processes the day number typed by the admin.
func HandleSessionSetPayDate(
	ctx context.Context,
	bot *tgbotapi.BotAPI,
	msg *tgbotapi.Message,
	sess *session.Session,
	sessions session.Store,
	uc *UseCases,
) {
	connUUID := sess.Data[session.KeyPayDateConnUUID]
	userIDStr := sess.Data[session.KeyPayDateUserID]
	sessions.Clear(msg.From.ID)

	day, err := strconv.Atoi(msg.Text)
	if err != nil || day < 1 || day > 28 {
		reply := tgbotapi.NewMessage(msg.Chat.ID, "❌ Введите число от 1 до 28:")
		reply.ReplyMarkup = keyboard.CancelKeyboard()
		// Restore session so user can retry.
		sessions.Set(msg.From.ID, &session.Session{
			State: session.StateSetPayDate,
			Data: map[string]string{
				session.KeyPayDateConnUUID: connUUID,
				session.KeyPayDateUserID:   userIDStr,
			},
		})
		send(bot, reply)
		return
	}

	nextPayDate := computeNextPayDate(day)

	if err := uc.Connection.SetConnLastPaidAt(ctx, connUUID, nextPayDate); err != nil {
		slog.Error("pay date: set last_paid_at", "uuid", connUUID, "err", err)
		send(bot, tgbotapi.NewMessage(msg.Chat.ID, "❌ Ошибка при сохранении даты."))
		return
	}

	slog.Info("pay date: set", "uuid", connUUID, "next_pay_date", nextPayDate.Format("2006-01-02"))

	reply := tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf(
		"✅ Дата оплаты установлена: <b>%s</b>\n\nНапоминание будет отправлено в этот день каждый месяц.",
		nextPayDate.Format("02.01.2006"),
	))
	reply.ParseMode = tgbotapi.ModeHTML
	send(bot, reply)

	// Show connection list for the user so admin can set dates for other connections.
	userID, _ := strconv.ParseInt(userIDStr, 10, 64)
	if userID != 0 {
		HandleAdminPayDateConns(ctx, bot, msg.Chat.ID, "", userID, uc)
	}
}

// computeNextPayDate returns the next occurrence of the given day of month.
// If the day hasn't arrived yet this month, returns this month's date.
// If it has already passed, returns next month's date.
func computeNextPayDate(day int) time.Time {
	now := time.Now()
	thisMonth := time.Date(now.Year(), now.Month(), day, 12, 0, 0, 0, now.Location())
	if !thisMonth.Before(now) {
		return thisMonth
	}
	next := now.AddDate(0, 1, 0)
	return time.Date(next.Year(), next.Month(), day, 12, 0, 0, 0, now.Location())
}
