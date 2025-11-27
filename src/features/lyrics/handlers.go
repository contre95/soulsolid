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
