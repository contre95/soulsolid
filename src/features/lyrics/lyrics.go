package lyrics

import (
	"context"
	"errors"

	"github.com/contre95/soulsolid/src/music"
)

// ErrNotFound is returned when lyrics cannot be found for a track
var ErrNotFound = errors.New("lyrics not found")

// LyricsProvider defines the interface for fetching lyrics from external services
type LyricsProvider interface {
	// SearchLyrics searches for lyrics using metadata parameters and returns lyrics text
	SearchLyrics(ctx context.Context, params music.LyricsSearchParams) (string, error)

	// Name returns the provider name
	Name() string

	// DisplayName returns the human-readable display name for the UI
	DisplayName() string

	// IsEnabled returns whether the provider is enabled
	IsEnabled() bool
}
