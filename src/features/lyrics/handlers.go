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
	GetEnabledLyricsProviders() map[string]bool
}

// NewHandler creates a new lyrics handler
func NewHandler(service *Service, metadataService MetadataService) *Handler {
	return &Handler{
		service:         service,
		metadataService: metadataService,
	}
}

// RenderLyricsButtons renders the lyrics provider buttons for a track
func (h *Handler) RenderLyricsButtons(c *fiber.Ctx) error {
	trackID := c.Params("trackId")
	if trackID == "" {
		return c.Status(fiber.StatusBadRequest).SendString("Track ID is required")
	}

	// Get track data for button context
	track, err := h.metadataService.GetTrackFileTags(c.Context(), trackID)
	if err != nil {
		slog.Error("Failed to get track for lyrics buttons", "error", err, "trackId", trackID)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to load track data")
	}

	return c.Render("tag/lyrics_buttons", fiber.Map{
		"Track":                  track,
		"EnabledLyricsProviders": h.metadataService.GetEnabledLyricsProviders(),
	})
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

// GetTrackLyrics returns the lyrics of a track in plain text.
func (h *Handler) GetTrackLyrics(c *fiber.Ctx) error {
	slog.Debug("GetTrackLyrics handler called", "id", c.Params("id"))
	track, err := h.service.libraryRepo.GetTrack(c.Context(), c.Params("id"))
	if err != nil {
		slog.Error("Error loading track", "error", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Error loading track")
	}
	if track == nil {
		return c.Status(fiber.StatusNotFound).SendString("Track not found")
	}
	return c.SendString(track.Metadata.Lyrics)
}
