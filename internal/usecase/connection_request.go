package usecase

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/EtoNeAnanasbI95/vpn-bot/internal/domain"
	"github.com/EtoNeAnanasbI95/vpn-bot/internal/repository"
)

type connRequestUseCase struct {
	repo repository.ConnRequestRepository
}

func NewConnRequestUseCase(repo repository.ConnRequestRepository) ConnRequestUseCase {
	return &connRequestUseCase{repo: repo}
}

// Create creates a new pending connection request.
// Returns an error if the user already has an active (non-completed) request.
func (uc *connRequestUseCase) Create(ctx context.Context, userID int64) (*domain.ConnRequest, error) {
	if existing, err := uc.repo.GetActiveByUserID(ctx, userID); err == nil && existing != nil {
		return nil, fmt.Errorf("already_active")
	}
	req := &domain.ConnRequest{
		UUID:   uuid.New().String(),
		UserID: userID,
		Status: domain.ConnReqPending,
	}
	if err := uc.repo.Create(ctx, req); err != nil {
		return nil, err
	}
	return req, nil
}

func (uc *connRequestUseCase) GetByUUID(ctx context.Context, id string) (*domain.ConnRequest, error) {
	return uc.repo.GetByUUID(ctx, id)
}

// Claim atomically claims a pending request for an admin.
// Returns false if another admin already claimed it.
func (uc *connRequestUseCase) Claim(ctx context.Context, id string, adminID int64, status domain.ConnRequestStatus) (bool, error) {
	return uc.repo.Claim(ctx, id, adminID, status)
}

func (uc *connRequestUseCase) SetAmount(ctx context.Context, id string, amount int) error {
	return uc.repo.SetAmount(ctx, id, amount)
}

func (uc *connRequestUseCase) MarkPaymentPending(ctx context.Context, id string) error {
	return uc.repo.UpdateStatus(ctx, id, domain.ConnReqPaymentPending)
}

func (uc *connRequestUseCase) Complete(ctx context.Context, id string) error {
	return uc.repo.UpdateStatus(ctx, id, domain.ConnReqCompleted)
}
