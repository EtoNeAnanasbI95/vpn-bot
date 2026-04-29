package bot

import (
	"context"
	"log/slog"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/EtoNeAnanasbI95/vpn-bot/internal/bot/callback"
	"github.com/EtoNeAnanasbI95/vpn-bot/internal/bot/handler"
	"github.com/EtoNeAnanasbI95/vpn-bot/internal/bot/middleware"
	"github.com/EtoNeAnanasbI95/vpn-bot/internal/bot/session"
	"github.com/EtoNeAnanasbI95/vpn-bot/internal/config"
)

// Router dispatches Telegram updates to the appropriate handlers.
type Router struct {
	bot      *tgbotapi.BotAPI
	cfg      *config.Config
	uc       *handler.UseCases
	sessions session.Store
}

func NewRouter(
	bot *tgbotapi.BotAPI,
	cfg *config.Config,
	uc *handler.UseCases,
	sessions session.Store,
) *Router {
	return &Router{bot: bot, cfg: cfg, uc: uc, sessions: sessions}
}

func (r *Router) Dispatch(ctx context.Context, update tgbotapi.Update) {
	switch {
	case update.Message != nil:
		r.handleMessage(ctx, update.Message)
	case update.CallbackQuery != nil:
		r.handleCallback(ctx, update.CallbackQuery)
	}
}

func (r *Router) handleMessage(ctx context.Context, msg *tgbotapi.Message) {
	if msg.From == nil {
		return
	}

	ctx, ok := middleware.Auth(ctx, r.bot, msg.From, msg.Chat.ID, r.uc.User, r.cfg)
	if !ok {
		return
	}

	// Multi-step session handling.
	if sess, exists := r.sessions.Get(msg.From.ID); exists && sess.State != session.StateNone {
		// Only admins can have broadcast/connection sessions.
		if handler.HandleSessionMessage(ctx, r.bot, msg, sess, r.sessions, r.uc) {
			return
		}
	}

	// Command routing.
	if msg.IsCommand() {
		switch msg.Command() {
		case "start":
			handler.HandleStart(ctx, r.bot, msg, r.uc, r.cfg)
		case "help":
			handler.HandleHelp(ctx, r.bot, msg.Chat.ID, r.cfg)
		}
		return
	}

	// Text button routing.
	switch msg.Text {
	case "🆕 Запросить подключение":
		handler.HandleUserRequestConnection(ctx, r.bot, msg.Chat.ID, msg.From.ID, r.cfg.AdminIDs, r.uc)

	case "🔗 Мои подключения":
		handler.HandleConnectionListFromMessage(ctx, r.bot, msg.Chat.ID, r.uc)
	case "📖 Гайды":
		handler.HandleGuideListFromMessage(ctx, r.bot, msg.Chat.ID, r.uc)
	case "❓ Помощь":
		handler.HandleHelp(ctx, r.bot, msg.Chat.ID, r.cfg)
	case "⚙️ Панель администратора":
		if middleware.IsAdminFromCtx(ctx) {
			handler.HandleAdminPanel(ctx, r.bot, msg.Chat.ID)
		}
	default:
		slog.Debug("unhandled message", "text", msg.Text, "user", msg.From.ID)
	}
}

func (r *Router) handleCallback(ctx context.Context, q *tgbotapi.CallbackQuery) {
	if q.Message == nil || q.From == nil {
		return
	}

	ctx, ok := middleware.Auth(ctx, r.bot, q.From, q.Message.Chat.ID, r.uc.User, r.cfg)
	if !ok {
		r.bot.Request(tgbotapi.NewCallback(q.ID, "")) //nolint:errcheck
		return
	}

	chatID := q.Message.Chat.ID
	msgID := q.Message.MessageID
	parts := callback.Decode(q.Data)
	action := parts[0]

	// Admin-only actions require guard.
	isAdminAction := len(action) >= 4 && action[:4] == "adm_"
	if isAdminAction && !middleware.AdminGuard(r.bot, q, r.cfg) {
		return
	}

	switch action {
	// ── User actions ─────────────────────────────────────────────────────────
	case callback.ActionConnList:
		handler.HandleConnectionList(ctx, r.bot, chatID, q.ID, r.uc)

	case callback.ActionConnQR:
		if len(parts) < 2 {
			break
		}
		handler.HandleConnectionQR(ctx, r.bot, chatID, q.ID, parts[1], r.uc)

	case callback.ActionConnPay:
		if len(parts) < 2 {
			break
		}
		handler.HandleConnectionPay(ctx, r.bot, chatID, q.ID, parts[1], r.uc)

	case callback.ActionConnPaid:
		if len(parts) < 2 {
			break
		}
		handler.HandleConnectionPaid(ctx, r.bot, chatID, q.ID, msgID, parts[1], r.uc)

	case callback.ActionAdmConnPayOK:
		if len(parts) < 2 {
			break
		}
		handler.HandleAdminConfirmConnPayment(ctx, r.bot, chatID, q.ID, msgID, parts[1], r.uc)

	case callback.ActionAdmConnCreate:
		if len(parts) < 2 {
			break
		}
		isFree := parts[1] == "1"
		handler.HandleAdminConnCreateFinal(ctx, r.bot, chatID, q.ID, q.From.ID, isFree, r.sessions, r.uc)

	case callback.ActionGuideList:
		handler.HandleGuideList(ctx, r.bot, chatID, q.ID, r.uc)

	case callback.ActionGuideGet:
		if len(parts) < 2 {
			break
		}
		handler.HandleGuideGet(ctx, r.bot, chatID, q.ID, parts[1], r.uc)

	case callback.ActionMainMenu:
		// Re-send start message with proper keyboard.
		handler.HandleStart(ctx, r.bot, &tgbotapi.Message{
			Chat: q.Message.Chat,
			From: q.From,
		}, r.uc, r.cfg)
		r.bot.Request(tgbotapi.NewCallback(q.ID, "")) //nolint:errcheck

	// ── Admin actions ────────────────────────────────────────────────────────
	case callback.ActionAdmSetPayInfo:
		handler.HandleAdminPaymentInfoMenu(ctx, r.bot, chatID, q.ID, q.From.ID, r.sessions, r.uc)

	case callback.ActionAdmMenu:
		r.bot.Request(tgbotapi.NewCallback(q.ID, "")) //nolint:errcheck
		handler.HandleAdminPanel(ctx, r.bot, chatID)

	case callback.ActionAdmPayList:
		handler.HandlePaymentList(ctx, r.bot, chatID, q.ID, r.uc)

	case callback.ActionAdmPayConfirm:
		if len(parts) < 2 {
			break
		}
		uid, _ := strconv.ParseInt(parts[1], 10, 64)
		handler.HandlePaymentConfirm(ctx, r.bot, chatID, q.ID, msgID, uid, r.uc)

	case callback.ActionAdmPayUnmark:
		if len(parts) < 2 {
			break
		}
		uid, _ := strconv.ParseInt(parts[1], 10, 64)
		handler.HandlePaymentUnmark(ctx, r.bot, chatID, q.ID, msgID, uid, r.uc)

	case callback.ActionAdmConnUsers:
		handler.HandleAdminConnUsers(ctx, r.bot, chatID, q.ID, r.uc)

	case callback.ActionAdmConnList:
		if len(parts) < 2 {
			break
		}
		uid, _ := strconv.ParseInt(parts[1], 10, 64)
		handler.HandleAdminConnList(ctx, r.bot, chatID, q.ID, uid, r.uc)

	case callback.ActionAdmConnAdd:
		if len(parts) < 2 {
			break
		}
		uid, _ := strconv.ParseInt(parts[1], 10, 64)
		handler.HandleAdminConnAdd(ctx, r.bot, chatID, q.ID, q.From.ID, uid, r.sessions, r.uc)

	case callback.ActionAdmConnDel:
		if len(parts) < 3 {
			break
		}
		uid, _ := strconv.ParseInt(parts[1], 10, 64)
		handler.HandleAdminConnDelete(ctx, r.bot, chatID, q.ID, msgID, uid, parts[2], r.uc)

	case callback.ActionAdmConnToggle:
		if len(parts) < 3 {
			break
		}
		val, _ := strconv.Atoi(parts[2])
		handler.HandleAdminConnToggle(ctx, r.bot, chatID, q.ID, msgID, parts[1], val == 1, r.uc)

	case callback.ActionAdmBcastMenu:
		handler.HandleBroadcastMenu(ctx, r.bot, chatID, q.ID)

	case callback.ActionAdmBcastAll:
		handler.HandleBroadcastAll(ctx, r.bot, chatID, q.ID, q.From.ID, r.sessions)

	case callback.ActionAdmBcastUser:
		if len(parts) == 1 {
			handler.HandleBroadcastSelectUser(ctx, r.bot, chatID, q.ID, r.uc)
		} else {
			uid, _ := strconv.ParseInt(parts[1], 10, 64)
			handler.HandleBroadcastToUser(ctx, r.bot, chatID, q.ID, uid, q.From.ID, r.sessions, r.uc)
		}

	case callback.ActionAdmBcastSelect:
		handler.HandleBroadcastOpenSelect(ctx, r.bot, chatID, q.ID, q.From.ID, r.sessions, r.uc)

	case callback.ActionAdmBcastToggle:
		if len(parts) < 2 {
			break
		}
		uid, _ := strconv.ParseInt(parts[1], 10, 64)
		handler.HandleBroadcastToggle(ctx, r.bot, chatID, q.ID, msgID, q.From.ID, uid, r.sessions, r.uc)

	case callback.ActionAdmBcastConfirm:
		handler.HandleBroadcastConfirmSelect(ctx, r.bot, chatID, q.ID, q.From.ID, r.sessions)

	case callback.ActionAdmUserList:
		handler.HandleAdminUserList(ctx, r.bot, chatID, q.ID, r.uc)

	case callback.ActionAdmUserDelete:
		if len(parts) < 2 {
			break
		}
		uid, _ := strconv.ParseInt(parts[1], 10, 64)
		handler.HandleAdminDeleteUser(ctx, r.bot, chatID, q.ID, uid, r.uc)

	case "adm_user_detail":
		if len(parts) < 2 {
			break
		}
		uid, _ := strconv.ParseInt(parts[1], 10, 64)
		handler.HandleAdminUserDetail(ctx, r.bot, chatID, q.ID, uid, r.uc)

	case callback.ActionAdmFreeFriendList:
		handler.HandleAdminFreeFriendList(ctx, r.bot, chatID, q.ID, r.uc)

	case callback.ActionAdmFreeFriendAdd:
		handler.HandleAdminFreeFriendAdd(ctx, r.bot, chatID, q.ID, r.uc)

	case callback.ActionAdmFreeFriendToggle:
		if len(parts) < 2 {
			break
		}
		uid, _ := strconv.ParseInt(parts[1], 10, 64)
		handler.HandleAdminFreeFriendToggle(ctx, r.bot, chatID, q.ID, uid, r.uc)

	case callback.ActionAdmConnNewUser:
		handler.HandleAdminConnNewUser(ctx, r.bot, chatID, q.ID, q.From.ID, r.sessions)

	case callback.ActionAdmPayDateUsers:
		handler.HandleAdminPayDateUsers(ctx, r.bot, chatID, q.ID, r.uc)

	case callback.ActionAdmPayDateConns:
		if len(parts) < 2 {
			break
		}
		uid, _ := strconv.ParseInt(parts[1], 10, 64)
		handler.HandleAdminPayDateConns(ctx, r.bot, chatID, q.ID, uid, r.uc)

	case callback.ActionAdmPayDateSet:
		if len(parts) < 2 {
			break
		}
		handler.HandleAdminPayDateSetStart(ctx, r.bot, chatID, q.ID, q.From.ID, parts[1], r.sessions, r.uc)

	// ── Connection request flow ──────────────────────────────────────────────
	case callback.ActionAdmReqFree:
		if len(parts) < 2 {
			break
		}
		handler.HandleAdmReqFree(ctx, r.bot, chatID, q.ID, msgID, q.From.ID, parts[1], r.uc)

	case callback.ActionAdmReqPaid:
		if len(parts) < 2 {
			break
		}
		handler.HandleAdmReqPaid(ctx, r.bot, chatID, q.ID, msgID, q.From.ID, parts[1], r.uc)

	case callback.ActionAdmReqPriceBase:
		if len(parts) < 2 {
			break
		}
		handler.HandleAdmReqPriceBase(ctx, r.bot, chatID, q.ID, msgID, q.From.ID, parts[1], r.uc)

	case callback.ActionAdmReqPriceCustom:
		if len(parts) < 2 {
			break
		}
		handler.HandleAdmReqPriceCustom(ctx, r.bot, chatID, q.ID, q.From.ID, parts[1], r.sessions)

	case callback.ActionConnReqCheckPay:
		if len(parts) < 2 {
			break
		}
		handler.HandleConnReqCheckPay(ctx, r.bot, chatID, q.ID, msgID, parts[1], r.uc)

	case callback.ActionAdmReqConfirmPay:
		if len(parts) < 2 {
			break
		}
		handler.HandleAdmReqConfirmPay(ctx, r.bot, chatID, q.ID, msgID, q.From.ID, parts[1], r.uc)

	case callback.ActionAdmCancel:
		r.bot.Request(tgbotapi.NewCallback(q.ID, "")) //nolint:errcheck
		sess, ok := r.sessions.Get(q.From.ID)
		if !ok {
			handler.HandleAdminPanel(ctx, r.bot, chatID)
			break
		}
		state := sess.State
		var connUserID int64
		if state == session.StateAddConnLabel || state == session.StateAddConnPaymentType {
			if s := sess.Data[session.KeyConnUserID]; s != "" {
				connUserID, _ = strconv.ParseInt(s, 10, 64)
			}
		}
		r.sessions.Clear(q.From.ID)
		switch state {
		case session.StateBroadcastAll, session.StateBroadcastToUser, session.StateBroadcastSelected, session.StateBroadcastSelect:
			handler.HandleBroadcastMenu(ctx, r.bot, chatID, "")
		case session.StateAddConnLabel, session.StateAddConnPaymentType:
			if connUserID != 0 {
				handler.HandleAdminConnList(ctx, r.bot, chatID, "", connUserID, r.uc)
			} else {
				handler.HandleAdminConnUsers(ctx, r.bot, chatID, "", r.uc)
			}
		case session.StateAddManualUser:
			handler.HandleAdminConnUsers(ctx, r.bot, chatID, "", r.uc)
		case session.StateAdmReqCustomPrice:
			r.bot.Send(tgbotapi.NewMessage(chatID, "❌ Ввод цены отменён.")) //nolint:errcheck
		case session.StateSetPayDate:
			handler.HandleAdminPayDateUsers(ctx, r.bot, chatID, "", r.uc)
		default:
			handler.HandleAdminPanel(ctx, r.bot, chatID)
		}

	default:
		r.bot.Request(tgbotapi.NewCallback(q.ID, "")) //nolint:errcheck
		slog.Debug("unhandled callback", "action", action, "user", q.From.ID)
	}
}
