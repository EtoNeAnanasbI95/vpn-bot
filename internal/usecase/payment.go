package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/EtoNeAnanasbI95/vpn-bot/internal/domain"
	"github.com/EtoNeAnanasbI95/vpn-bot/internal/repository"
)

type paymentUseCase struct {
	paymentRepo repository.PaymentRepository
}

func NewPaymentUseCase(paymentRepo repository.PaymentRepository) PaymentUseCase {
	return &paymentUseCase{paymentRepo: paymentRepo}
}

func (uc *paymentUseCase) GetCurrentStatus(ctx context.Context, userID int64) (*domain.Payment, error) {
	now := time.Now()
	payment, err := uc.paymentRepo.GetOrCreate(ctx, userID, now.Year(), int(now.Month()))
	if err != nil {
		return nil, fmt.Errorf("get payment status: %w", err)
	}
	return payment, nil
}

func (uc *paymentUseCase) ConfirmPayment(ctx context.Context, userID int64, confirmedByAdminID int64) error {
	now := time.Now()
	// Ensure the record exists before marking it paid.
	if _, err := uc.paymentRepo.GetOrCreate(ctx, userID, now.Year(), int(now.Month())); err != nil {
		return fmt.Errorf("get or create payment: %w", err)
	}
	return uc.paymentRepo.MarkPaid(ctx, userID, now.Year(), int(now.Month()), confirmedByAdminID, now)
}

func (uc *paymentUseCase) UnmarkPayment(ctx context.Context, userID int64) error {
	now := time.Now()
	return uc.paymentRepo.MarkUnpaid(ctx, userID, now.Year(), int(now.Month()))
}

func (uc *paymentUseCase) GetUnpaidUsers(ctx context.Context) ([]*domain.Payment, error) {
	now := time.Now()
	return uc.paymentRepo.GetUnpaidForPeriod(ctx, now.Year(), int(now.Month()))
}
