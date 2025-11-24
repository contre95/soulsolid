package metadata

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/contre95/soulsolid/src/features/tagging"
)

// TekstowoProvider implements LyricsProvider for Tekstowo.pl
type TekstowoProvider struct {
	enabled bool
}

// NewTekstowoProvider creates a new Tekstowo provider
func NewTekstowoProvider(enabled bool) *TekstowoProvider {
	return &TekstowoProvider{enabled: enabled}
}

func (p *TekstowoProvider) SearchLyrics(ctx context.Context, params tagging.LyricsSearchParams) (string, error) {
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

func (p *TekstowoProvider) searchSong(ctx context.Context, query string) (string, error) {
	// Try different search URL formats - Tekstowo uses Polish URL structure
	searchURLs := []string{
		fmt.Sprintf("https://www.tekstowo.pl/szukaj,%s.html", url.QueryEscape(query)),
		fmt.Sprintf("https://www.tekstowo.pl/szukaj.html?q=%s", url.QueryEscape(query)),
		fmt.Sprintf("https://www.tekstowo.pl/search?q=%s", url.QueryEscape(query)),
	}

	var lastErr error
	for _, searchURL := range searchURLs {
		req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
		if err != nil {
			lastErr = fmt.Errorf("failed to create request: %w", err)
			continue
		}

		req.Header.Set("User-Agent", "SoulSolid/1.0")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("failed to make request: %w", err)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("Tekstowo search request failed with status %d for URL %s", resp.StatusCode, searchURL)
			continue
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

		// Extract first song URL from search results
		songURL, err := p.extractSongURLFromSearch(string(htmlContent))
		if err != nil {
			lastErr = fmt.Errorf("failed to extract song URL: %w", err)
			continue
		}

		return songURL, nil
	}

	return "", lastErr
}

func (p *TekstowoProvider) extractSongURLFromSearch(html string) (string, error) {
	// Look for song links in search results - try multiple patterns
	patterns := []string{
		`<a[^>]*href="(/piosenka/[^"]+\.html)"[^>]*>`,
		`<a[^>]*href="(https://www\.tekstowo\.pl/piosenka/[^"]+\.html)"[^>]*>`,
		`<a[^>]*href="([^"]*piosenka[^"]*\.html)"[^>]*>`,
		`<a[^>]*href="(/[^"]*\.html)"[^>]*class="[^"]*title[^"]*"[^>]*>`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllStringSubmatch(html, -1)

		for _, match := range matches {
			if len(match) > 1 {
				songPath := match[1]
				// Make sure it's actually a song link, not a navigation link
				if strings.Contains(songPath, "piosenka") || strings.Contains(songPath, "song") {
					if strings.HasPrefix(songPath, "http") {
						return songPath, nil
					}
					return "https://www.tekstowo.pl" + songPath, nil
				}
			}
		}
	}

	return "", fmt.Errorf("no songs found in search results")
}

func (p *TekstowoProvider) fetchLyrics(ctx context.Context, songURL string) (string, error) {
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

func (p *TekstowoProvider) extractLyricsFromHTML(html string) (string, error) {
	// Look for the lyrics container in the HTML - try multiple patterns
	patterns := []string{
		`<div[^>]*class="song-text"[^>]*>(.*?)</div>`,
		`<div[^>]*id="songText"[^>]*>(.*?)</div>`,
		`<div[^>]*class="lyrics"[^>]*>(.*?)</div>`,
		`<div[^>]*class="tekst"[^>]*>(.*?)</div>`, // Polish for "text"
		`<div[^>]*class="[^"]*text[^"]*"[^>]*>(.*?)</div>`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(`(?s)` + pattern) // (?s) makes . match newlines
		matches := re.FindAllStringSubmatch(html, -1)

		for _, match := range matches {
			if len(match) >= 2 {
				// Clean up HTML tags
				lyrics := p.cleanLyricsHTML(match[1])
				// Check if this looks like actual lyrics (not navigation text)
				if len(lyrics) > 20 && !strings.Contains(lyrics, "Przeglądaj") && !strings.Contains(lyrics, "wykonawców") {
					return lyrics, nil
				}
			}
		}
	}

	return "", fmt.Errorf("lyrics not found in page")
}

func (p *TekstowoProvider) cleanLyricsHTML(html string) string {
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

func (p *TekstowoProvider) Name() string    { return "tekstowo" }
func (p *TekstowoProvider) IsEnabled() bool { return p.enabled }
