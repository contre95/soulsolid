package playback

import (
	"log/slog"

	"github.com/gofiber/fiber/v2"
)

// Handler handles playback requests
type Handler struct {
	service *Service
}

// NewHandler creates a new playback handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// GetTrackPreview serves a 30-second audio preview of a track
func (h *Handler) GetTrackPreview(c *fiber.Ctx) error {
	slog.Debug("GetTrackPreview handler called", "trackID", c.Params("id"))

	trackID := c.Params("id")
	if trackID == "" {
		return c.Status(fiber.StatusBadRequest).SendString("Track ID is required")
	}

	reader, contentType, err := h.service.GetTrackPreview(c.Context(), trackID)
	if err != nil {
		slog.Error("Failed to get track preview", "trackID", trackID, "error", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to get track preview")
	}

	// Set content type based on track format
	if contentType != "" {
		c.Set("Content-Type", "audio/"+contentType)
	} else {
		c.Set("Content-Type", "audio/mpeg") // default
	}

	// Set additional headers for proper streaming
	c.Set("Accept-Ranges", "bytes")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")

	slog.Debug("Streaming audio", "contentType", contentType, "trackID", trackID)

	// Stream the audio
	err = c.SendStream(reader)
	if err != nil {
		slog.Error("Failed to stream audio", "trackID", trackID, "error", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to stream audio")
	}
	return nil
}
