package metadata

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/contre95/soulsolid/src/features/tagging"
)

// Genius API response structures
type geniusSearchResponse struct {
	Response struct {
		Hits []geniusHit `json:"hits"`
	} `json:"response"`
}

type geniusHit struct {
	Result geniusSong `json:"result"`
}

type geniusSong struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	ArtistNames string `json:"artist_names"`
	Path        string `json:"path"`
}

// GeniusProvider implements LyricsProvider for Genius
type GeniusProvider struct {
	enabled bool
}

// NewGeniusProvider creates a new Genius provider
func NewGeniusProvider(enabled bool) *GeniusProvider {
	return &GeniusProvider{enabled: enabled}
}

func (p *GeniusProvider) SearchLyrics(ctx context.Context, params tagging.LyricsSearchParams) (string, error) {
	// Build search query
	var queryParts []string

	if params.Title != "" {
		queryParts = append(queryParts, params.Title)
	}
	if params.Artist != "" {
		queryParts = append(queryParts, params.Artist)
	}

	if len(queryParts) == 0 {
		return "", fmt.Errorf("insufficient search parameters")
	}

	query := strings.Join(queryParts, " ")

	// Search for the song
	songURL, err := p.searchSong(ctx, query)
	if err != nil {
		return "", fmt.Errorf("failed to search song: %w", err)
	}

	// Fetch lyrics from the song page
	lyrics, err := p.fetchLyrics(ctx, songURL)
	if err != nil {
		return "", fmt.Errorf("failed to fetch lyrics: %w", err)
	}

	return lyrics, nil
}

func (p *GeniusProvider) searchSong(ctx context.Context, query string) (string, error) {
	searchURL := fmt.Sprintf("https://genius.com/api/search?q=%s", url.QueryEscape(query))

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
		return "", fmt.Errorf("Genius API request failed with status %d", resp.StatusCode)
	}

	var searchResp geniusSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(searchResp.Response.Hits) == 0 {
		return "", fmt.Errorf("no songs found")
	}

	// Return the first result
	songURL := "https://genius.com" + searchResp.Response.Hits[0].Result.Path
	return songURL, nil
}

func (p *GeniusProvider) fetchLyrics(ctx context.Context, songURL string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", songURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "SoulSolid/1.0")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch lyrics page: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("lyrics page request failed with status %d", resp.StatusCode)
	}

	// Read the HTML content
	htmlContent := make([]byte, 0)
	buf := make([]byte, 1024)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			htmlContent = append(htmlContent, buf[:n]...)
		}
		if err != nil {
			break
		}
	}

	// Extract lyrics from HTML
	lyrics, err := p.extractLyricsFromHTML(string(htmlContent))
	if err != nil {
		return "", fmt.Errorf("failed to extract lyrics: %w", err)
	}

	return lyrics, nil
}

func (p *GeniusProvider) extractLyricsFromHTML(html string) (string, error) {
	// Look for the lyrics container in the HTML
	// Genius uses various classes, try multiple patterns
	patterns := []string{
		`<div[^>]*data-lyrics-container="true"[^>]*>(.*?)</div>`,
		`<div[^>]*class="Lyrics__Container[^"]*"[^>]*>(.*?)</div>`,
		`<div[^>]*class="lyrics"[^>]*>(.*?)</div>`,
		`<div[^>]*class="[^"]*lyrics[^"]*"[^>]*>(.*?)</div>`,
		`<p[^>]*class="[^"]*lyrics[^"]*"[^>]*>(.*?)</p>`,
		`<div[^>]*class="song_body-lyrics"[^>]*>(.*?)</div>`,
	}

	var allMatches [][]string
	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllStringSubmatch(html, -1)
		if len(matches) > 0 {
			allMatches = append(allMatches, matches...)
		}
	}

	if len(allMatches) == 0 {
		// Try a more general approach - look for any div with lyrics-related content
		re := regexp.MustCompile(`(?s)<div[^>]*>(.*?lyrics.*?)<\/div>`)
		matches := re.FindAllStringSubmatch(html, -1)
		if len(matches) > 0 {
			allMatches = matches
		}
	}

	if len(allMatches) == 0 {
		return "", fmt.Errorf("lyrics not found in page")
	}

	var lyricsParts []string
	for _, match := range allMatches {
		if len(match) > 1 {
			// Clean up HTML tags
			lyrics := p.cleanLyricsHTML(match[1])
			// Only include if it looks like actual lyrics (not just HTML)
			if len(lyrics) > 10 && !strings.Contains(lyrics, "<") && !strings.Contains(lyrics, ">") {
				lyricsParts = append(lyricsParts, lyrics)
			}
		}
	}

	if len(lyricsParts) == 0 {
		return "", fmt.Errorf("no lyrics content extracted")
	}

	return strings.Join(lyricsParts, "\n\n"), nil
}

func (p *GeniusProvider) cleanLyricsHTML(html string) string {
	// Remove HTML tags
	re := regexp.MustCompile(`<[^>]*>`)
	clean := re.ReplaceAllString(html, "")

	// Decode HTML entities (basic)
	clean = strings.ReplaceAll(clean, "&amp;", "&")
	clean = strings.ReplaceAll(clean, "&lt;", "<")
	clean = strings.ReplaceAll(clean, "&gt;", ">")
	clean = strings.ReplaceAll(clean, "&quot;", "\"")
	clean = strings.ReplaceAll(clean, "&#x27;", "'")
	clean = strings.ReplaceAll(clean, "&apos;", "'")

	// Clean up extra whitespace
	clean = strings.TrimSpace(clean)
	clean = regexp.MustCompile(`\n\s*\n`).ReplaceAllString(clean, "\n\n")

	return clean
}

func (p *GeniusProvider) Name() string    { return "genius" }
func (p *GeniusProvider) IsEnabled() bool { return p.enabled }
