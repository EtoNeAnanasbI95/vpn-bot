package handler

import (
	"context"
	"fmt"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/EtoNeAnanasbI95/vpn-bot/internal/bot/keyboard"
	"github.com/EtoNeAnanasbI95/vpn-bot/internal/bot/session"
)

// HandleSessionMessage processes incoming text messages when the user has an
// active session state (multi-step flow). Returns true if the message was handled.
func HandleSessionMessage(
	ctx context.Context,
	bot *tgbotapi.BotAPI,
	msg *tgbotapi.Message,
	sess *session.Session,
	sessions session.Store,
	uc *UseCases,
) bool {
	switch sess.State {
	case session.StateBroadcastAll:
		sessions.Clear(msg.From.ID)
		ExecuteBroadcastAll(ctx, bot, msg.Chat.ID, msg.Text, uc)
		return true

	case session.StateBroadcastToUser:
		targetIDStr := sess.Data[session.KeyTargetUserID]
		targetID, err := strconv.ParseInt(targetIDStr, 10, 64)
		if err != nil {
			sessions.Clear(msg.From.ID)
			send(bot, tgbotapi.NewMessage(msg.Chat.ID, "Ошибка: неверный ID пользователя."))
			return true
		}
		sessions.Clear(msg.From.ID)
		ExecuteBroadcastToUser(ctx, bot, msg.Chat.ID, targetID, msg.Text)
		return true

	case session.StateBroadcastSelected:
		selected := parseSelectedIDs(sess.Data[session.KeyBcastSelectedIDs])
		sessions.Clear(msg.From.ID)
		ExecuteBroadcastSelected(ctx, bot, msg.Chat.ID, msg.Text, selected, uc)
		return true

	case session.StateSetPaymentInfo:
		sessions.Clear(msg.From.ID)
		HandleSessionSetPaymentInfo(ctx, bot, msg, uc)
		return true

	case session.StateAddConnLabel:
		// Store label and transition to payment type selection.
		sessions.Set(msg.From.ID, &session.Session{
			State: session.StateAddConnPaymentType,
			Data: map[string]string{
				session.KeyConnUserID:  sess.Data[session.KeyConnUserID],
				session.KeyConnTgTag:   sess.Data[session.KeyConnTgTag],
				session.KeyConnAdminID: fmt.Sprintf("%d", msg.From.ID),
				session.KeyConnLabel:   msg.Text,
			},
		})
		reply := tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("Лейбл: <b>%s</b>\n\nВыберите тип подключения:", msg.Text))
		reply.ParseMode = tgbotapi.ModeHTML
		reply.ReplyMarkup = keyboard.ConnPaymentTypeSelect()
		send(bot, reply)
		return true

	case session.StateAddManualUser:
		sessions.Clear(msg.From.ID)
		HandleSessionAddManualUser(ctx, bot, msg, uc)
		return true

	case session.StateAdmReqCustomPrice:
		sessions.Clear(msg.From.ID)
		HandleSessionAdmReqCustomPrice(ctx, bot, msg, sess, uc)
		return true

	case session.StateSetPayDate:
		HandleSessionSetPayDate(ctx, bot, msg, sess, sessions, uc)
		return true
	}

	return false
}

