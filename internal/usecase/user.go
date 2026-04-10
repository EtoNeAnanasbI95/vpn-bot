package usecase

import (
	"context"
	"fmt"

	"github.com/EtoNeAnanasbI95/vpn-bot/internal/domain"
	"github.com/EtoNeAnanasbI95/vpn-bot/internal/repository"
)

type userUseCase struct {
	userRepo repository.UserRepository
	adminIDs []int64
}

func NewUserUseCase(userRepo repository.UserRepository, adminIDs []int64) UserUseCase {
	return &userUseCase{userRepo: userRepo, adminIDs: adminIDs}
}

func (uc *userUseCase) RegisterOrGet(ctx context.Context, tu domain.TelegramUser) (*domain.User, bool, error) {
	existing, err := uc.userRepo.GetByID(ctx, tu.ID)
	if err == nil && existing != nil {
		// User already exists — update name fields via upsert, return as not new
		existing.Username = tu.Username
		existing.FirstName = tu.FirstName
		existing.LastName = tu.LastName
		if err := uc.userRepo.Upsert(ctx, existing); err != nil {
			return nil, false, fmt.Errorf("update user: %w", err)
		}
		return existing, false, nil
	}

	adminID, err := uc.pickAdmin(ctx)
	if err != nil {
		return nil, false, fmt.Errorf("pick admin: %w", err)
	}

	u := &domain.User{
		ID:        tu.ID,
		Username:  tu.Username,
		FirstName: tu.FirstName,
		LastName:  tu.LastName,
		AdminID:   adminID,
	}
	if err := uc.userRepo.Upsert(ctx, u); err != nil {
		return nil, false, fmt.Errorf("create user: %w", err)
	}
	return u, true, nil
}

func (uc *userUseCase) GetUser(ctx context.Context, id int64) (*domain.User, error) {
	return uc.userRepo.GetByID(ctx, id)
}

func (uc *userUseCase) GetByUsername(ctx context.Context, username string) (*domain.User, error) {
	return uc.userRepo.GetByUsername(ctx, username)
}

func (uc *userUseCase) GetAll(ctx context.Context) ([]*domain.User, error) {
	return uc.userRepo.GetAll(ctx)
}

func (uc *userUseCase) GetByAdmin(ctx context.Context, adminID int64) ([]*domain.User, error) {
	return uc.userRepo.GetByAdminID(ctx, adminID)
}

func (uc *userUseCase) DeleteUser(ctx context.Context, userID int64) error {
	return uc.userRepo.Delete(ctx, userID)
}

func (uc *userUseCase) SetFreeFriend(ctx context.Context, userID int64, isFree bool) error {
	return uc.userRepo.SetFreeFriend(ctx, userID, isFree)
}

func (uc *userUseCase) GetFreeFriends(ctx context.Context) ([]*domain.User, error) {
	return uc.userRepo.GetFreeFriends(ctx)
}


// pickAdmin returns the admin ID with the fewest assigned users.
func (uc *userUseCase) pickAdmin(ctx context.Context) (int64, error) {
	if len(uc.adminIDs) == 0 {
		return 0, fmt.Errorf("no admins configured")
	}

	minCount := -1
	picked := uc.adminIDs[0]

	for _, adminID := range uc.adminIDs {
		count, err := uc.userRepo.CountByAdminID(ctx, adminID)
		if err != nil {
			return 0, err
		}
		if minCount < 0 || count < minCount {
			minCount = count
			picked = adminID
		}
	}
	return picked, nil
}
