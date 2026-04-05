package guide

import "context"

// Provider is the pluggable interface for fetching platform guides.
// Swap FSProvider for an S3Provider, DBProvider, or URLProvider as needed.
type Provider interface {
	// ListPlatforms returns the available platforms in display order.
	ListPlatforms(ctx context.Context) ([]Platform, error)
	// GetGuide returns raw PDF bytes for the given platform key.
	GetGuide(ctx context.Context, platformKey string) ([]byte, error)
}

// Platform describes an entry shown in the platform selection keyboard.
type Platform struct {
	Key   string // internal key, used in callback_data e.g. "ios"
	Label string // shown to user e.g. "iOS"
}
