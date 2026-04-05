package usecase

import (
	"context"
	"fmt"

	"github.com/EtoNeAnanasbI95/vpn-bot/internal/domain"
	"github.com/EtoNeAnanasbI95/vpn-bot/internal/guide"
)

type guideUseCase struct {
	provider guide.Provider
}

func NewGuideUseCase(provider guide.Provider) GuideUseCase {
	return &guideUseCase{provider: provider}
}

func (uc *guideUseCase) ListPlatforms(ctx context.Context) ([]domain.Platform, error) {
	platforms, err := uc.provider.ListPlatforms(ctx)
	if err != nil {
		return nil, fmt.Errorf("list platforms: %w", err)
	}
	result := make([]domain.Platform, len(platforms))
	for i, p := range platforms {
		result[i] = domain.Platform{Key: p.Key, Label: p.Label}
	}
	return result, nil
}

func (uc *guideUseCase) GetGuide(ctx context.Context, platformKey string) ([]byte, error) {
	data, err := uc.provider.GetGuide(ctx, platformKey)
	if err != nil {
		return nil, fmt.Errorf("get guide %q: %w", platformKey, err)
	}
	return data, nil
}
