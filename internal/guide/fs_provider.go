package guide

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

// FSProvider reads PDF guides from a directory on disk.
// Place files as <guidesDir>/<platform_key>.pdf
type FSProvider struct {
	dir       string
	platforms []Platform
}

// NewFSProvider creates a provider that reads PDFs from the given directory.
// The platforms list defines order and labels; only platforms whose PDF exists are returned.
func NewFSProvider(dir string) *FSProvider {
	return &FSProvider{
		dir: dir,
		platforms: []Platform{
			{Key: "ios", Label: "iOS"},
			{Key: "android", Label: "Android"},
			{Key: "windows", Label: "Windows"},
			{Key: "macos", Label: "macOS"},
			{Key: "linux", Label: "Linux"},
		},
	}
}

func (p *FSProvider) ListPlatforms(_ context.Context) ([]Platform, error) {
	var available []Platform
	for _, platform := range p.platforms {
		path := filepath.Join(p.dir, platform.Key+".pdf")
		if _, err := os.Stat(path); err == nil {
			available = append(available, platform)
		}
	}
	return available, nil
}

func (p *FSProvider) GetGuide(_ context.Context, platformKey string) ([]byte, error) {
	// Sanitize key to prevent path traversal
	for _, platform := range p.platforms {
		if platform.Key == platformKey {
			path := filepath.Join(p.dir, platformKey+".pdf")
			data, err := os.ReadFile(path)
			if err != nil {
				return nil, fmt.Errorf("guide for %q not found", platformKey)
			}
			return data, nil
		}
	}
	return nil, fmt.Errorf("unknown platform %q", platformKey)
}
