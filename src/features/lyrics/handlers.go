package lyrics

import (
	"context"
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
	GetArtists(ctx context.Context) ([]*music.Artist, error)
	GetAlbums(ctx context.Context) ([]*music.Album, error)
	GetEnabledMetadataProviders() map[string]bool
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

// GetLyricsText returns plain lyrics text for HTMX to set in textarea
func (h *Handler) GetLyricsText(c *fiber.Ctx) error {
	trackID := c.Params("trackId")
	providerName := c.Params("provider")

	if trackID == "" || providerName == "" {
		return c.Status(fiber.StatusBadRequest).SendString("Track ID and provider name are required")
	}

	// Fetch lyrics
	lyrics, err := h.service.SearchLyrics(c.Context(), trackID, providerName)
	if err != nil {
		slog.Error("Failed to fetch lyrics", "error", err, "trackId", trackID, "provider", providerName)
		return c.SendString("") // Return empty string on error for HTMX
	}

	slog.Info("Lyrics fetched successfully", "trackId", trackID, "provider", providerName, "lyricsLength", len(lyrics))

	// Return plain lyrics text for HTMX to set in textarea
	return c.SendString(lyrics)
}
