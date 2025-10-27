package downloading

import (
	"fmt"
	"regexp"
)

// ParsedURL represents a parsed music URL
type ParsedURL struct {
	Service string // e.g., "deezer"
	Type    string // "album", "track"
	ID      string
}

// ParseMusicURL parses a music service URL and extracts service, type, and ID
func ParseMusicURL(url string) (*ParsedURL, error) {
	// Deezer patterns
	deezerAlbumRegex := regexp.MustCompile(`(?:https?://)?(?:www\.)?deezer\.com/(?:[a-z]{2}/)?album/(\d+)`)
	deezerTrackRegex := regexp.MustCompile(`(?:https?://)?(?:www\.)?deezer\.com/(?:[a-z]{2}/)?track/(\d+)`)

	if matches := deezerAlbumRegex.FindStringSubmatch(url); len(matches) > 1 {
		return &ParsedURL{
			Service: "deezer",
			Type:    "album",
			ID:      matches[1],
		}, nil
	}

	if matches := deezerTrackRegex.FindStringSubmatch(url); len(matches) > 1 {
		return &ParsedURL{
			Service: "deezer",
			Type:    "track",
			ID:      matches[1],
		}, nil
	}

	// Add more services here as needed
	// e.g., Spotify, etc.

	return nil, fmt.Errorf("unsupported URL format: %s", url)
}

// IsValidMusicURL checks if a URL is a valid music service URL
func IsValidMusicURL(url string) bool {
	_, err := ParseMusicURL(url)
	return err == nil
}
