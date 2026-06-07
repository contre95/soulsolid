package streaming

import (
	"log/slog"
	"net/url"

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

// Stream serves the file at the given path query parameter.
func (h *Handler) Stream(c *fiber.Ctx) error {
	rawPath := c.Query("path")
	path, err := url.QueryUnescape(rawPath)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("invalid path")
	}
	resolved, mimeType, err := h.service.Stream(path)
	if err != nil {
		slog.Error("Stream: rejected path", "path", path, "error", err)
		return c.Status(fiber.StatusNotFound).SendString("track not found")
	}
	c.Set("Content-Type", mimeType)
	c.Set("Accept-Ranges", "bytes")
	return c.SendFile(resolved)
}
