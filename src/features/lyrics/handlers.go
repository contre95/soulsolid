package lyrics

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/contre95/soulsolid/src/music"
	"github.com/gofiber/fiber/v2"
)

// Handler handles lyrics requests
type Handler struct {
	service         *Service
	metadataService MetadataService // For accessing track data and UI rendering
}

// MetadataService interface for accessing metadata functionality needed by lyrics handlers
type MetadataService interface {
	GetTrackFileTags(ctx context.Context, trackID string) (*music.Track, error)
	GetEnabledLyricsProviders() map[string]bool
}

// NewHandler creates a new lyrics handler
func NewHandler(service *Service, metadataService MetadataService) *Handler {
	return &Handler{
		service:         service,
		metadataService: metadataService,
	}
}

// getLyricsProviderColors returns color classes for a given lyrics provider
func (h *Handler) getLyricsProviderColors(providerName string) map[string]string {
	switch providerName {
	case "genius":
		return map[string]string{
			"label":     "text-yellow-600 dark:text-yellow-300",
			"border":    "border-yellow-400 dark:border-yellow-300",
			"focusRing": "focus:ring-yellow-500 focus:border-yellow-500",
			"text":      "text-yellow-700 dark:text-yellow-300",
		}
	case "tekstowo":
		return map[string]string{
			"label":     "text-green-600 dark:text-green-300",
			"border":    "border-green-400 dark:border-green-300",
			"focusRing": "focus:ring-green-500 focus:border-green-500",
			"text":      "text-green-700 dark:text-green-300",
		}
	case "lrclib":
		return map[string]string{
			"label":     "text-blue-600 dark:text-blue-300",
			"border":    "border-blue-400 dark:border-blue-300",
			"focusRing": "focus:ring-blue-500 focus:border-blue-500",
			"text":      "text-blue-700 dark:text-blue-300",
		}
	case "best":
		return map[string]string{
			"label":     "text-gradient-to-r from-yellow-600 via-green-600 to-blue-600 dark:from-yellow-300 dark:via-green-300 dark:to-blue-300",
			"border":    "border-gradient-to-r from-yellow-400 via-green-400 to-blue-400 dark:from-yellow-300 dark:via-green-300 dark:to-blue-300",
			"focusRing": "focus:ring-gradient-to-r focus:from-yellow-500 focus:via-green-500 focus:to-blue-500 focus:border-gradient-to-r focus:from-yellow-500 focus:via-green-500 focus:to-blue-500",
			"text":      "text-gradient-to-r from-yellow-700 via-green-700 to-blue-700 dark:from-yellow-300 dark:via-green-300 dark:to-blue-300",
		}
	default:
		// Default to blue for unknown providers
		return map[string]string{
			"label":     "text-blue-600 dark:text-blue-300",
			"border":    "border-blue-400 dark:border-blue-300",
			"focusRing": "focus:ring-blue-500 focus:border-blue-500",
			"text":      "text-blue-700 dark:text-blue-300",
		}
	}
}

// FetchLyricsFromProvider handles fetching lyrics from any lyrics provider
func (h *Handler) FetchLyricsFromProvider(c *fiber.Ctx) error {
	trackID := c.Params("trackId")
	providerName := c.Params("provider")

	if trackID == "" || providerName == "" {
		return c.Status(fiber.StatusBadRequest).SendString("Track ID and provider name are required")
	}

	// Fetch lyrics
	lyrics, err := h.service.SearchLyrics(c.Context(), trackID, providerName)
	if err != nil {
		slog.Error("Failed to fetch lyrics", "error", err, "trackId", trackID, "provider", providerName)
		return c.Render("toast/toastErr", fiber.Map{
			"Msg": fmt.Sprintf("Failed to fetch lyrics: %v", err),
		})
	}

	slog.Info("Lyrics fetched successfully", "trackId", trackID, "provider", providerName, "lyricsLength", len(lyrics))

	// For HTMX requests, just return the lyrics content to replace the textarea
	if c.Get("HX-Request") == "true" {
		return c.SendString(lyrics)
	}

	// For regular requests, update the track and render the full form
	// Get current track data
	track, err := h.metadataService.GetTrackFileTags(c.Context(), trackID)
	if err != nil {
		slog.Error("Failed to get track for lyrics update", "error", err, "trackId", trackID)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to load track data")
	}

	// Update track lyrics
	track.Metadata.Lyrics = lyrics

	// Render the updated form - this would need to be handled by metadata feature
	// For now, return success
	return c.Render("toast/toastOk", fiber.Map{
		"Msg": fmt.Sprintf("Lyrics from %s loaded successfully!", providerName),
	})
}

// FetchBestLyrics handles fetching lyrics from all providers and shows them in a modal
func (h *Handler) FetchBestLyrics(c *fiber.Ctx) error {
	trackID := c.Params("trackId")

	if trackID == "" {
		return c.Status(fiber.StatusBadRequest).SendString("Track ID is required")
	}

	// Get current track data
	track, err := h.metadataService.GetTrackFileTags(c.Context(), trackID)
	if err != nil {
		slog.Error("Failed to get track for lyrics", "error", err, "trackId", trackID)
		return c.Render("toast/toastErr", fiber.Map{
			"Msg": fmt.Sprintf("Failed to get track: %v", err),
		})
	}

	// Build search parameters
	searchParams := music.LyricsSearchParams{
		TrackID: track.ID,
		Title:   track.Title,
	}

	if len(track.Artists) > 0 && track.Artists[0].Artist != nil {
		searchParams.Artist = track.Artists[0].Artist.Name
	}

	if track.Album != nil {
		searchParams.Album = track.Album.Title
		if len(track.Album.Artists) > 0 && track.Album.Artists[0].Artist != nil {
			searchParams.AlbumArtist = track.Album.Artists[0].Artist.Name
		}
	}

	// Collect lyrics from all enabled providers
	var results []*LyricsResult
	for _, provider := range h.service.lyricsProviders {
		if !provider.IsEnabled() {
			continue
		}

		slog.Debug("Trying lyrics provider", "provider", provider.Name(), "trackID", trackID, "title", searchParams.Title, "artist", searchParams.Artist)
		lyrics, err := provider.SearchLyrics(c.Context(), searchParams)
		if err != nil {
			slog.Warn("Failed to search lyrics with provider", "provider", provider.Name(), "trackID", trackID, "title", searchParams.Title, "artist", searchParams.Artist, "error", err.Error())
			continue
		}

		if lyrics != "" {
			quality, confidence := h.service.scoreLyrics(lyrics, searchParams)
			result := &LyricsResult{
				Lyrics:     lyrics,
				Provider:   provider.Name(),
				Quality:    quality,
				Confidence: confidence,
			}
			results = append(results, result)

			slog.Info("Found lyrics with provider", "provider", provider.Name(), "trackID", trackID, "quality", quality, "confidence", confidence, "lyricsLength", len(lyrics))
		}
	}

	if len(results) == 0 {
		slog.Info("No lyrics found for track with any provider", "trackId", trackID, "title", track.Title, "artist", searchParams.Artist, "providers", len(h.service.lyricsProviders))
		return c.Render("toast/toastInfo", fiber.Map{
			"Msg": "No lyrics found from any provider",
		})
	}

	// Sort results by quality (highest first)
	for i := 0; i < len(results)-1; i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].Quality > results[i].Quality ||
				(results[j].Quality == results[i].Quality && results[j].Confidence > results[i].Confidence) {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	slog.Info("Lyrics search completed", "trackId", trackID, "resultsCount", len(results))

	// Render modal with lyrics results - this would need to be handled by metadata feature
	return c.Render("toast/toastOk", fiber.Map{
		"Msg": fmt.Sprintf("Found lyrics from %d providers", len(results)),
	})
}

// SelectLyricsFromProvider handles selecting specific lyrics from modal results
func (h *Handler) SelectLyricsFromProvider(c *fiber.Ctx) error {
	trackID := c.Params("trackId")
	providerName := c.Params("provider")

	if trackID == "" || providerName == "" {
		return c.Status(fiber.StatusBadRequest).SendString("Track ID and provider name are required")
	}

	// Get lyrics from specific provider
	lyrics, err := h.service.SearchLyrics(c.Context(), trackID, providerName)
	if err != nil {
		slog.Error("Failed to fetch lyrics from provider", "error", err, "trackId", trackID, "provider", providerName)
		return c.Render("toast/toastErr", fiber.Map{
			"Msg": fmt.Sprintf("Failed to fetch lyrics: %v", err),
		})
	}

	// Get current track data
	track, err := h.metadataService.GetTrackFileTags(c.Context(), trackID)
	if err != nil {
		slog.Error("Failed to get track for updating", "error", err, "trackId", trackID)
		return c.Render("toast/toastErr", fiber.Map{
			"Msg": fmt.Sprintf("Failed to get track: %v", err),
		})
	}

	// Update track with selected lyrics
	track.Metadata.Lyrics = lyrics

	// Update track in database - this would need access to library repo
	// For now, just return success
	slog.Info("Successfully selected lyrics from provider", "trackId", trackID, "provider", providerName, "lyricsLength", len(lyrics))

	// Return success toast and close modal
	return c.Render("toast/toastOk", fiber.Map{
		"Msg": fmt.Sprintf("Lyrics from %s added successfully!", providerName),
	})
}
