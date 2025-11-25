package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/contre95/soulsolid/src/features/metadata"
)

// LRCLib API response structures
type lrclibSearchResponse []lrclibSong

type lrclibSong struct {
	ID           int     `json:"id"`
	Name         string  `json:"name"`
	Artist       string  `json:"artist"`
	Album        string  `json:"album"`
	Duration     float64 `json:"duration"`
	PlainLyrics  string  `json:"plainLyrics"`
	SyncedLyrics string  `json:"syncedLyrics"`
}

// LRCLibProvider implements LyricsProvider for LRCLib
type LRCLibProvider struct {
	enabled bool
}

// NewLRCLibProvider creates a new LRCLib provider
func NewLRCLibProvider(enabled bool) *LRCLibProvider {
	return &LRCLibProvider{enabled: enabled}
}

func (p *LRCLibProvider) SearchLyrics(ctx context.Context, params metadata.LyricsSearchParams) (string, error) {
	// Build search query
	var queryParts []string

	if params.Title != "" {
		queryParts = append(queryParts, fmt.Sprintf("track_name=%s", url.QueryEscape(params.Title)))
	}
	if params.Artist != "" {
		queryParts = append(queryParts, fmt.Sprintf("artist_name=%s", url.QueryEscape(params.Artist)))
	}
	if params.Album != "" {
		queryParts = append(queryParts, fmt.Sprintf("album_name=%s", url.QueryEscape(params.Album)))
	}

	if len(queryParts) == 0 {
		return "", fmt.Errorf("insufficient search parameters")
	}

	query := strings.Join(queryParts, "&")

	// Search for lyrics
	searchURL := fmt.Sprintf("https://lrclib.net/api/search?%s", query)

	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "SoulSolid/1.0")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("LRCLib API request failed with status %d", resp.StatusCode)
	}

	var searchResp lrclibSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(searchResp) == 0 {
		return "", fmt.Errorf("no lyrics found")
	}

	// Return the first result's plain lyrics
	song := searchResp[0]
	if song.PlainLyrics != "" {
		return song.PlainLyrics, nil
	}

	// If no plain lyrics, try to extract from synced lyrics
	if song.SyncedLyrics != "" {
		return p.extractPlainLyricsFromSynced(song.SyncedLyrics), nil
	}

	return "", fmt.Errorf("no lyrics content available")
}

func (p *LRCLibProvider) extractPlainLyricsFromSynced(syncedLyrics string) string {
	// LRCLib synced lyrics format is like: [00:00.00] Line 1\n[00:05.00] Line 2
	lines := strings.Split(syncedLyrics, "\n")
	var plainLines []string

	for _, line := range lines {
		// Remove timestamp brackets
		if strings.Contains(line, "]") {
			parts := strings.SplitN(line, "]", 2)
			if len(parts) == 2 {
				plainLines = append(plainLines, strings.TrimSpace(parts[1]))
			}
		}
	}

	return strings.Join(plainLines, "\n")
}

func (p *LRCLibProvider) Name() string    { return "lrclib" }
func (p *LRCLibProvider) IsEnabled() bool { return p.enabled }
