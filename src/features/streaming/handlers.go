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
	if rawPath == "" {
		return c.Status(fiber.StatusBadRequest).SendString("missing path")
	}
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
	// Fiber's SendFile feeds the path to fasthttp as a request URI, so characters
	// like '?' or '#' in a filename are parsed as query/fragment and truncate the
	// path, causing a spurious 404. Percent-encode the path so it round-trips
	// through fasthttp's URI decoding intact.
	return c.SendFile((&url.URL{Path: resolved}).EscapedPath())
}
