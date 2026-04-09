package usecase

import (
	"context"
	"time"

	"github.com/EtoNeAnanasbI95/vpn-bot/internal/domain"
)

type UserUseCase interface {
	// RegisterOrGet upserts the user and assigns an admin on first encounter.
	// Returns the user and whether it was newly created.
	RegisterOrGet(ctx context.Context, tu domain.TelegramUser) (*domain.User, bool, error)
	GetUser(ctx context.Context, id int64) (*domain.User, error)
	GetByUsername(ctx context.Context, username string) (*domain.User, error)
	GetAll(ctx context.Context) ([]*domain.User, error)
	GetByAdmin(ctx context.Context, adminID int64) ([]*domain.User, error)
	DeleteUser(ctx context.Context, userID int64) error
	// SetFreeFriend marks or clears the free-friend flag. Free friends are excluded
	// from all payment reminders.
	SetFreeFriend(ctx context.Context, userID int64, isFree bool) error
	GetFreeFriends(ctx context.Context) ([]*domain.User, error)
	// SetLastPaidAt records the date a user last paid (nil clears it).
	// One month after this date, the scheduler sends them a renewal reminder.
	SetLastPaidAt(ctx context.Context, userID int64, paidAt *time.Time) error
	// GetUsersWithDueReminder returns users whose last_paid_at was 1+ months ago.
	GetUsersWithDueReminder(ctx context.Context) ([]*domain.User, error)
}

type ConnectionUseCase interface {
	// ListForUser returns all connections for the given Telegram user from 3x-ui.
	ListForUser(ctx context.Context, userID int64) ([]*domain.Connection, error)
	// GetByUUID returns a single connection by its 3x-ui client UUID.
	GetByUUID(ctx context.Context, uuid string) (*domain.Connection, error)
	// GenerateQR returns a PNG-encoded QR code for the given VLESS link.
	GenerateQR(ctx context.Context, link string) ([]byte, error)
	// Create creates a client on the Reality inbound in 3x-ui.
	// adminID is the admin issuing the connection. isFree=true skips payment tracking.
	Create(ctx context.Context, userID, adminID int64, tgTag, label string, isFree bool) (*domain.Connection, error)
	// Remove deletes a client from 3x-ui and its payment record.
	Remove(ctx context.Context, clientUUID string) error
	// SetEnabled enables or disables a client in 3x-ui.
	SetEnabled(ctx context.Context, clientUUID string, enabled bool) error
	// GetAllUnpaidPayments returns all connections with status "unpaid" (for scheduler).
	GetAllUnpaidPayments(ctx context.Context) ([]*domain.ConnPayment, error)
	// GetOverduePayments returns unpaid connections older than the given duration.
	GetOverduePayments(ctx context.Context, olderThan time.Duration) ([]*domain.ConnPayment, error)
	// GetAdminPaymentInfo returns the payment credentials for the admin who issued the connection.
	GetAdminPaymentInfo(ctx context.Context, connUUID string) (string, error)
	// SetPaymentPending marks a connection payment as pending (user claims paid).
	SetPaymentPending(ctx context.Context, connUUID string) error
	// ConfirmConnPayment marks payment as paid and returns the userID to notify.
	ConfirmConnPayment(ctx context.Context, connUUID string) (int64, error)
	// SetAdminPaymentInfo saves payment credentials for an admin.
	SetAdminPaymentInfo(ctx context.Context, adminID int64, info string) error
	// GetAdminOwnPaymentInfo returns the payment credentials set by the given admin.
	GetAdminOwnPaymentInfo(ctx context.Context, adminID int64) (string, error)
}

type PaymentUseCase interface {
	GetCurrentStatus(ctx context.Context, userID int64) (*domain.Payment, error)
	ConfirmPayment(ctx context.Context, userID int64, confirmedByAdminID int64) error
	UnmarkPayment(ctx context.Context, userID int64) error
	GetUnpaidUsers(ctx context.Context) ([]*domain.Payment, error)
}

type GuideUseCase interface {
	ListPlatforms(ctx context.Context) ([]domain.Platform, error)
	GetGuide(ctx context.Context, platformKey string) ([]byte, error)
}
