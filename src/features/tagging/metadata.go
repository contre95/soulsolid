package tagging

import (
	"context"

	"github.com/contre95/soulsolid/src/music"
)

// MetadataProvider defines the interface for fetching metadata from external services
type MetadataProvider interface {
	// FetchMetadata fetches track metadata using a fingerprint
	FetchMetadata(ctx context.Context, fingerprint string) (*music.Track, error)

	// Name returns the provider name
	Name() string

	// IsEnabled returns whether the provider is enabled
	IsEnabled() bool
}
