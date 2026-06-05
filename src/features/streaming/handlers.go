package streaming

import (
	"log/slog"

	"github.com/gofiber/fiber/v2"
)

// Handler handles audio streaming requests.
type Handler struct {
	service *Service
}

// NewHandler creates a new streaming handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// StreamQueueItem streams a pending track from the download folder.
func (h *Handler) StreamQueueItem(c *fiber.Ctx) error {
	id := c.Params("id")
	path, mimeType, err := h.service.QueueTrackStream(id)
	if err != nil {
		slog.Error("StreamQueueItem: failed to resolve path", "id", id, "error", err)
		return c.Status(fiber.StatusNotFound).SendString("track not found")
	}
	c.Set("Content-Type", mimeType)
	c.Set("Accept-Ranges", "bytes")
	return c.SendFile(path)
}

// StreamLibraryTrack streams an imported library track.
func (h *Handler) StreamLibraryTrack(c *fiber.Ctx) error {
	id := c.Params("id")
	path, mimeType, err := h.service.LibraryTrackStream(c.Context(), id)
	if err != nil {
		slog.Error("StreamLibraryTrack: failed to resolve path", "id", id, "error", err)
		return c.Status(fiber.StatusNotFound).SendString("track not found")
	}
	c.Set("Content-Type", mimeType)
	c.Set("Accept-Ranges", "bytes")
	return c.SendFile(path)
}
