package repository

import (
	"context"
	"time"

	"github.com/EtoNeAnanasbI95/vpn-bot/internal/domain"
)

type UserRepository interface {
	// Upsert creates the user if not exists, otherwise updates name fields.
	Upsert(ctx context.Context, u *domain.User) error
	GetByID(ctx context.Context, id int64) (*domain.User, error)
	GetByUsername(ctx context.Context, username string) (*domain.User, error)
	GetAll(ctx context.Context) ([]*domain.User, error)
	GetByAdminID(ctx context.Context, adminID int64) ([]*domain.User, error)
	CountByAdminID(ctx context.Context, adminID int64) (int, error)
	Delete(ctx context.Context, id int64) error
	// SetFreeFriend marks or clears the free-friend flag for a user.
	SetFreeFriend(ctx context.Context, userID int64, isFree bool) error
	// GetFreeFriends returns all users with is_free_friend = 1.
	GetFreeFriends(ctx context.Context) ([]*domain.User, error)
}

type ConnectionPaymentRepository interface {
	Create(ctx context.Context, p *domain.ConnPayment) error
	GetByUUID(ctx context.Context, uuid string) (*domain.ConnPayment, error)
	GetAllUnpaid(ctx context.Context) ([]*domain.ConnPayment, error)
	GetOverdue(ctx context.Context, olderThan time.Time) ([]*domain.ConnPayment, error)
	SetStatus(ctx context.Context, uuid string, status domain.ConnPayStatus) error
	GetAdminPaymentInfo(ctx context.Context, adminID int64) (string, error)
	SetAdminPaymentInfo(ctx context.Context, adminID int64, info string) error
	// SetLastPaidAt records (or clears) the last payment date for a connection.
	// If no row exists yet for the uuid, one is created (status='paid').
	SetLastPaidAt(ctx context.Context, uuid string, userID, adminID int64, paidAt *time.Time) error
	// GetConnsWithDuePaidReminder returns connections whose last_paid_at was
	// more than one calendar month ago and whose user is not a free-friend.
	GetConnsWithDuePaidReminder(ctx context.Context) ([]*domain.ConnPayment, error)
}

type PaymentRepository interface {
	// GetOrCreate returns the existing payment for (user, year, month), creating
	// an unpaid record if none exists.
	GetOrCreate(ctx context.Context, userID int64, year, month int) (*domain.Payment, error)
	GetByUserAndPeriod(ctx context.Context, userID int64, year, month int) (*domain.Payment, error)
	// GetUnpaidForPeriod returns all unpaid payments for the given year/month.
	GetUnpaidForPeriod(ctx context.Context, year, month int) ([]*domain.Payment, error)
	MarkPaid(ctx context.Context, userID int64, year, month int, confirmedBy int64, paidAt time.Time) error
	MarkUnpaid(ctx context.Context, userID int64, year, month int) error
}
