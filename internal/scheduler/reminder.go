package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/robfig/cron/v3"

	"github.com/EtoNeAnanasbI95/vpn-bot/internal/bot/keyboard"
	"github.com/EtoNeAnanasbI95/vpn-bot/internal/usecase"
)

// Scheduler manages periodic payment reminder jobs.
type Scheduler struct {
	c         *cron.Cron
	bot       *tgbotapi.BotAPI
	userUC    usecase.UserUseCase
	paymentUC usecase.PaymentUseCase
	connUC    usecase.ConnectionUseCase
	adminIDs  []int64
}

func (s *Scheduler) isAdmin(id int64) bool {
	for _, a := range s.adminIDs {
		if a == id {
			return true
		}
	}
	return false
}

// New creates a Scheduler with jobs running in the given timezone.
func New(
	bot *tgbotapi.BotAPI,
	userUC usecase.UserUseCase,
	paymentUC usecase.PaymentUseCase,
	connUC usecase.ConnectionUseCase,
	tz string,
	adminIDs []int64,
) (*Scheduler, error) {
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return nil, fmt.Errorf("load timezone %q: %w", tz, err)
	}

	s := &Scheduler{
		c:         cron.New(cron.WithLocation(loc)),
		bot:       bot,
		userUC:    userUC,
		paymentUC: paymentUC,
		connUC:    connUC,
		adminIDs:  adminIDs,
	}

	// 12:00 and 20:00 in the configured timezone.
	if _, err := s.c.AddFunc("0 12 * * *", s.sendReminders); err != nil {
		return nil, fmt.Errorf("add 12:00 job: %w", err)
	}
	if _, err := s.c.AddFunc("0 20 * * *", s.sendReminders); err != nil {
		return nil, fmt.Errorf("add 20:00 job: %w", err)
	}
	// Daily at 10:00 — block connections overdue by more than 3 days.
	if _, err := s.c.AddFunc("0 10 * * *", s.blockOverdueConnections); err != nil {
		return nil, fmt.Errorf("add overdue block job: %w", err)
	}

	return s, nil
}

func (s *Scheduler) Start() {
	s.c.Start()
	slog.Info("payment reminder scheduler started")
}

func (s *Scheduler) Stop() {
	s.c.Stop()
}

func (s *Scheduler) blockOverdueConnections() {
	ctx := context.Background()

	overdue, err := s.connUC.GetOverduePayments(ctx, 3*24*time.Hour)
	if err != nil {
		slog.Error("scheduler: get overdue connections", "err", err)
		return
	}
	if len(overdue) == 0 {
		return
	}

	slog.Info("scheduler: blocking overdue connections", "count", len(overdue))

	for _, pay := range overdue {
		if s.isAdmin(pay.UserID) {
			continue
		}
		if err := s.connUC.SetEnabled(ctx, pay.UUID, false); err != nil {
			slog.Warn("scheduler: block overdue connection", "uuid", pay.UUID, "err", err)
			continue
		}
		slog.Info("scheduler: connection blocked (overdue)", "uuid", pay.UUID, "user_id", pay.UserID)

		// Notify user.
		userMsg := tgbotapi.NewMessage(pay.UserID,
			"🚫 <b>Подключение заблокировано</b>\n\nОплата не поступила в течение 3 дней. Обратитесь к администратору.")
		userMsg.ParseMode = tgbotapi.ModeHTML
		if _, err := s.bot.Send(userMsg); err != nil {
			slog.Warn("scheduler: notify user (overdue block)", "user_id", pay.UserID, "err", err)
		}

		// Notify admin.
		if pay.AdminID == 0 {
			continue
		}
		user, err := s.userUC.GetUser(ctx, pay.UserID)
		if err != nil {
			continue
		}
		name := user.DisplayName()
		if user.Username != "" {
			name += " (@" + user.Username + ")"
		}
		adminMsg := tgbotapi.NewMessage(pay.AdminID, fmt.Sprintf(
			"🚫 <b>Подключение заблокировано</b>\n\nПользователь <b>%s</b> не оплатил подключение в течение 3 дней — оно автоматически заблокировано.",
			name,
		))
		adminMsg.ParseMode = tgbotapi.ModeHTML
		if _, err := s.bot.Send(adminMsg); err != nil {
			slog.Warn("scheduler: notify admin (overdue block)", "admin_id", pay.AdminID, "err", err)
		}
	}
}

func (s *Scheduler) sendConnPaymentReminders(ctx context.Context) {
	unpaid, err := s.connUC.GetAllUnpaidPayments(ctx)
	if err != nil {
		slog.Error("scheduler: get unpaid connections", "err", err)
		return
	}
	if len(unpaid) == 0 {
		return
	}

	slog.Info("scheduler: sending connection payment reminders", "count", len(unpaid))

	for _, pay := range unpaid {
		if s.isAdmin(pay.UserID) {
			continue
		}
		// Remind the user.
		userMsg := tgbotapi.NewMessage(pay.UserID,
			"⏰ <b>Напоминание об оплате подключения</b>\n\nОдно из ваших подключений ожидает оплаты.\n\nОткройте раздел «🔗 Мои подключения» и нажмите <b>Оплатить</b>.")
		userMsg.ParseMode = tgbotapi.ModeHTML
		if _, err := s.bot.Send(userMsg); err != nil {
			slog.Warn("scheduler: notify user (conn payment)", "user_id", pay.UserID, "err", err)
		}

		// Remind the issuing admin.
		if pay.AdminID == 0 {
			continue
		}
		user, err := s.userUC.GetUser(ctx, pay.UserID)
		if err != nil {
			continue
		}
		name := user.DisplayName()
		if user.Username != "" {
			name += " (@" + user.Username + ")"
		}
		adminMsg := tgbotapi.NewMessage(pay.AdminID, fmt.Sprintf(
			"💳 <b>Неоплаченное подключение</b>\n\nПользователь <b>%s</b> ещё не оплатил выданное подключение.",
			name,
		))
		adminMsg.ParseMode = tgbotapi.ModeHTML
		if _, err := s.bot.Send(adminMsg); err != nil {
			slog.Warn("scheduler: notify admin (conn payment)", "admin_id", pay.AdminID, "err", err)
		}
	}
}

func (s *Scheduler) sendReminders() {
	ctx := context.Background()
	s.sendConnPaymentReminders(ctx)

	unpaid, err := s.paymentUC.GetUnpaidUsers(ctx)
	if err != nil {
		slog.Error("scheduler: get unpaid users", "err", err)
		return
	}

	slog.Info("scheduler: sending payment reminders", "count", len(unpaid))

	for _, payment := range unpaid {
		if s.isAdmin(payment.UserID) {
			continue
		}

		user, err := s.userUC.GetUser(ctx, payment.UserID)
		if err != nil {
			slog.Warn("scheduler: get user", "user_id", payment.UserID, "err", err)
			continue
		}

		// Skip free friends — they never receive payment reminders.
		if user.IsFreeFriend {
			continue
		}

		// Send reminder to user.
		userMsg := tgbotapi.NewMessage(user.ID, fmt.Sprintf(
			"⏰ <b>Напоминание об оплате</b>\n\nОплата за <b>%s</b> ещё не подтверждена.\n\nПожалуйста, произведите оплату для продолжения обслуживания VPN.",
			payment.PeriodLabel(),
		))
		userMsg.ParseMode = tgbotapi.ModeHTML
		if _, err := s.bot.Send(userMsg); err != nil {
			slog.Warn("scheduler: notify user", "user_id", user.ID, "err", err)
		}

		// Send notification to assigned admin.
		if user.AdminID == 0 {
			continue
		}

		name := user.DisplayName()
		if user.Username != "" {
			name += " (@" + user.Username + ")"
		}

		adminMsg := tgbotapi.NewMessage(user.AdminID, fmt.Sprintf(
			"💳 <b>Не оплачено</b>\n\nПользователь <b>%s</b> не оплатил VPN за <b>%s</b>.\n\nID: <code>%d</code>",
			name, payment.PeriodLabel(), user.ID,
		))
		adminMsg.ParseMode = tgbotapi.ModeHTML
		adminMsg.ReplyMarkup = keyboard.PayConfirmButton(user.ID)
		if _, err := s.bot.Send(adminMsg); err != nil {
			slog.Warn("scheduler: notify admin", "admin_id", user.AdminID, "err", err)
		}
	}
}

