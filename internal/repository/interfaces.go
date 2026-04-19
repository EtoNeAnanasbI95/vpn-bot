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
	// GetNonFriends returns users with is_free_friend = 0 (regular clients).
	GetNonFriends(ctx context.Context) ([]*domain.User, error)
}

type ConnectionPaymentRepository interface {
	Create(ctx context.Context, p *domain.ConnPayment) error
	GetByUUID(ctx context.Context, uuid string) (*domain.ConnPayment, error)
	GetAllUnpaid(ctx context.Context) ([]*domain.ConnPayment, error)
	GetOverdue(ctx context.Context, olderThan time.Time) ([]*domain.ConnPayment, error)
	SetStatus(ctx context.Context, uuid string, status domain.ConnPayStatus) error
	GetAdminPaymentInfo(ctx context.Context, adminID int64) (string, error)
	SetAdminPaymentInfo(ctx context.Context, adminID int64, info string) error
}

type ConnRequestRepository interface {
	Create(ctx context.Context, r *domain.ConnRequest) error
	GetByUUID(ctx context.Context, uuid string) (*domain.ConnRequest, error)
	// GetActiveByUserID returns the most recent non-completed request for the user.
	GetActiveByUserID(ctx context.Context, userID int64) (*domain.ConnRequest, error)
	// Claim atomically sets admin_id and status if the request is still 'pending'.
	Claim(ctx context.Context, uuid string, adminID int64, status domain.ConnRequestStatus) (bool, error)
	UpdateStatus(ctx context.Context, uuid string, status domain.ConnRequestStatus) error
	SetAmount(ctx context.Context, uuid string, amount int) error
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
