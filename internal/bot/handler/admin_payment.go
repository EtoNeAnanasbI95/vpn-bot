package handler

import (
	"context"
	"fmt"
	"log/slog"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/EtoNeAnanasbI95/vpn-bot/internal/bot/keyboard"
	"github.com/EtoNeAnanasbI95/vpn-bot/internal/bot/middleware"
)

// HandlePaymentList shows all users with their current payment status.
func HandlePaymentList(
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

	paidMap := make(map[int64]bool, len(users))
	for _, u := range users {
		payment, err := uc.Payment.GetCurrentStatus(ctx, u.ID)
		if err == nil && payment != nil {
			paidMap[u.ID] = payment.IsPaid()
		}
	}

	msg := tgbotapi.NewMessage(chatID, "💳 <b>Статус оплат за текущий месяц</b>\n\nНажмите на клиента, чтобы изменить статус:")
	msg.ParseMode = tgbotapi.ModeHTML
	msg.ReplyMarkup = keyboard.PaymentList(users, paidMap)
	send(bot, msg)
}

// HandlePaymentConfirm marks payment as paid for the current month.
func HandlePaymentConfirm(
	ctx context.Context,
	bot *tgbotapi.BotAPI,
	chatID int64,
	callbackID string,
	messageID int,
	targetUserID int64,
	uc *UseCases,
) {
	admin := middleware.UserFromCtx(ctx)
	if err := uc.Payment.ConfirmPayment(ctx, targetUserID, admin.ID); err != nil {
		slog.Error("admin: payment confirm", "admin_id", admin.ID, "target_user_id", targetUserID, "err", err)
		answerCallback(bot, callbackID, "Ошибка при обновлении оплаты")
		return
	}
	slog.Info("admin: payment confirmed", "admin_id", admin.ID, "target_user_id", targetUserID)
	answerCallback(bot, callbackID, "✅ Оплата подтверждена")

	// Notify the user.
	payment, _ := uc.Payment.GetCurrentStatus(ctx, targetUserID)
	if payment != nil {
		notify := tgbotapi.NewMessage(targetUserID,
			fmt.Sprintf("✅ Ваша оплата за <b>%s</b> подтверждена! Спасибо.", payment.PeriodLabel()))
		notify.ParseMode = tgbotapi.ModeHTML
		bot.Send(notify) //nolint:errcheck
	}

	// Refresh the payment list in the admin's message.
	refreshPaymentMessage(ctx, bot, chatID, messageID, uc)
}

// HandlePaymentUnmark removes the paid mark for the current month.
func HandlePaymentUnmark(
	ctx context.Context,
	bot *tgbotapi.BotAPI,
	chatID int64,
	callbackID string,
	messageID int,
	targetUserID int64,
	uc *UseCases,
) {
	admin := middleware.UserFromCtx(ctx)
	if err := uc.Payment.UnmarkPayment(ctx, targetUserID); err != nil {
		slog.Error("admin: payment unmark", "admin_id", admin.ID, "target_user_id", targetUserID, "err", err)
		answerCallback(bot, callbackID, "Ошибка при обновлении оплаты")
		return
	}
	slog.Info("admin: payment unmarked", "admin_id", admin.ID, "target_user_id", targetUserID)
	answerCallback(bot, callbackID, "❌ Отметка снята")
	refreshPaymentMessage(ctx, bot, chatID, messageID, uc)
}

func refreshPaymentMessage(ctx context.Context, bot *tgbotapi.BotAPI, chatID int64, messageID int, uc *UseCases) {
	users, err := uc.User.GetAll(ctx)
	if err != nil {
		return
	}
	paidMap := make(map[int64]bool, len(users))
	for _, u := range users {
		p, err := uc.Payment.GetCurrentStatus(ctx, u.ID)
		if err == nil && p != nil {
			paidMap[u.ID] = p.IsPaid()
		}
	}
	editMarkup(bot, chatID, messageID, keyboard.PaymentList(users, paidMap))
}
