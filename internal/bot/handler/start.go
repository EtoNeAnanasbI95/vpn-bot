package handler

import (
	"context"
	"fmt"
	"log/slog"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/EtoNeAnanasbI95/vpn-bot/internal/bot/keyboard"
	"github.com/EtoNeAnanasbI95/vpn-bot/internal/bot/middleware"
	"github.com/EtoNeAnanasbI95/vpn-bot/internal/config"
)

// HandleStart handles the /start command.
func HandleStart(
	ctx context.Context,
	bot *tgbotapi.BotAPI,
	msg *tgbotapi.Message,
	uc *UseCases,
	cfg *config.Config,
) {
	u := middleware.UserFromCtx(ctx)
	isAdmin := middleware.IsAdminFromCtx(ctx)

	slog.Info("/start", "user_id", u.ID, "username", u.Username, "is_admin", isAdmin)

	// Ensure payment record for current month exists.
	uc.Payment.GetCurrentStatus(ctx, u.ID) //nolint:errcheck

	greeting := fmt.Sprintf("Привет, <b>%s</b>! 👋\n\nЯ помогу тебе управлять VPN-подключениями.", u.FirstName)

	reply := tgbotapi.NewMessage(msg.Chat.ID, greeting)
	reply.ParseMode = tgbotapi.ModeHTML

	if isAdmin {
		reply.ReplyMarkup = keyboard.AdminMainMenu()
	} else {
		reply.ReplyMarkup = keyboard.MainMenu()
	}
	send(bot, reply)
}
