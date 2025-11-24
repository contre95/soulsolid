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

	client := &http.Client{
		// Follow redirects but limit them
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch lyrics page: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("lyrics page request failed with status %d", resp.StatusCode)
	}

	// Check if we were redirected to an error page
	finalURL := resp.Request.URL.String()
	if strings.Contains(finalURL, "404") || strings.Contains(finalURL, "error") {
		return "", fmt.Errorf("redirected to error page: %s", finalURL)
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

	html := string(htmlContent)

	// Quick check for obvious error indicators
	if strings.Contains(html, "34 ContributorsTranslations") ||
		strings.Contains(html, "This page didn't load") ||
		strings.Contains(html, "Page not found") ||
		len(html) < 1000 { // Very short pages are likely errors
		return "", fmt.Errorf("page appears to be an error or empty page")
	}

	// Extract lyrics from HTML
	lyrics, err := p.extractLyricsFromHTML(html)
	if err != nil {
		return "", fmt.Errorf("failed to extract lyrics: %w", err)
	}

	return lyrics, nil
}

func (p *GeniusProvider) extractLyricsFromHTML(html string) (string, error) {
	// First check if this looks like an error page or non-lyrics page
	if strings.Contains(html, "34 ContributorsTranslations") ||
		strings.Contains(html, "404") ||
		strings.Contains(html, "Page not found") ||
		strings.Contains(html, "error") ||
		strings.Contains(html, "Error") {
		return "", fmt.Errorf("page appears to be an error page or non-lyrics content")
	}

	// Check if the page has the expected Genius structure
	if !strings.Contains(html, "genius.com") && !strings.Contains(html, "Genius") {
		return "", fmt.Errorf("page does not appear to be a valid Genius page")
	}

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
		re := regexp.MustCompile(`(?s)` + pattern) // (?s) makes . match newlines
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
			// Only include if it looks like actual lyrics (not just HTML or error content)
			if len(lyrics) > 10 && !strings.Contains(lyrics, "<") && !strings.Contains(lyrics, ">") &&
				!strings.Contains(lyrics, "ContributorsTranslations") &&
				!strings.Contains(lyrics, "404") &&
				!strings.Contains(lyrics, "error") {
				lyricsParts = append(lyricsParts, lyrics)
			}
		}
	}

	if len(lyricsParts) == 0 {
		return "", fmt.Errorf("no valid lyrics content extracted")
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
