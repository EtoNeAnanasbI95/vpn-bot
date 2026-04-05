package middleware

import (
	"context"
	"log/slog"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/EtoNeAnanasbI95/vpn-bot/internal/config"
	"github.com/EtoNeAnanasbI95/vpn-bot/internal/domain"
	"github.com/EtoNeAnanasbI95/vpn-bot/internal/usecase"
)

type contextKey string

const (
	ctxUser    contextKey = "user"
	ctxIsAdmin contextKey = "is_admin"
)

func WithUser(ctx context.Context, u *domain.User) context.Context {
	return context.WithValue(ctx, ctxUser, u)
}

func UserFromCtx(ctx context.Context) *domain.User {
	u, _ := ctx.Value(ctxUser).(*domain.User)
	return u
}

func WithIsAdmin(ctx context.Context, isAdmin bool) context.Context {
	return context.WithValue(ctx, ctxIsAdmin, isAdmin)
}

func IsAdminFromCtx(ctx context.Context) bool {
	v, _ := ctx.Value(ctxIsAdmin).(bool)
	return v
}

// Auth fetches or creates the user and injects them into the context.
// Returns the enriched context and whether the handler should proceed.
func Auth(
	ctx context.Context,
	bot *tgbotapi.BotAPI,
	from *tgbotapi.User,
	chatID int64,
	userUC usecase.UserUseCase,
	cfg *config.Config,
) (context.Context, bool) {
	tu := domain.TelegramUser{
		ID:        from.ID,
		Username:  from.UserName,
		FirstName: from.FirstName,
		LastName:  from.LastName,
	}

	u, isNew, err := userUC.RegisterOrGet(ctx, tu)
	if err != nil {
		slog.Error("auth: register user", "user_id", from.ID, "username", from.UserName, "err", err)
		return ctx, false
	}

	if isNew {
		slog.Info("auth: new user registered", "user_id", u.ID, "username", u.Username, "name", u.DisplayName())
	}

	ctx = WithUser(ctx, u)
	ctx = WithIsAdmin(ctx, cfg.IsAdmin(from.ID))
	return ctx, true
}

// AdminGuard answers the callback with an alert if the user is not an admin.
// Returns true if the user is an admin.
func AdminGuard(bot *tgbotapi.BotAPI, query *tgbotapi.CallbackQuery, cfg *config.Config) bool {
	if cfg.IsAdmin(query.From.ID) {
		return true
	}
	alert := tgbotapi.NewCallback(query.ID, "⛔️ Только для администраторов")
	alert.ShowAlert = true
	bot.Request(alert) //nolint:errcheck
	return false
}
