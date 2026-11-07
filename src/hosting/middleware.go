package hosting

import (
	"log/slog"
	"strings"
	"time"

	"github.com/contre95/soulsolid/src/hosting/respond"
	"github.com/gofiber/fiber/v3"
)

// HTMXMiddleware creates middleware for logging HTMX requests
func HTMXMiddleware() fiber.Handler {
	return func(c fiber.Ctx) error {
		start := time.Now()

		// Check if this is an HTMX request
		isHTMX := respond.IsHTMX(c)

		// Process the request
		err := c.Next()

		// Log HTMX-specific information (skip high-frequency paths to prevent spam)
		if isHTMX {
			path := c.Path()
			if strings.Contains(path, "/jobs/list") || strings.Contains(path, "/jobs/count") || strings.Contains(path, "/queue/count") {
				return err // skip logging for polling endpoints
			}
			duration := time.Since(start)

			slog.Debug("HTMX request",
				"method", c.Method(),
				"path", c.Path(),
				"status", c.Response().StatusCode(),
				"duration", duration.String(),
				"hx_trigger", c.Get("HX-Trigger"),
				"hx_target", c.Get("HX-Target"),
				"hx_current_url", c.Get("HX-Current-URL"),
				"user_agent", c.Get("User-Agent"),
			)
		}

		return err
	}
}

// LogAllRequestsMiddleware logs all requests with HTMX context
func LogAllRequestsMiddleware() fiber.Handler {
	return func(c fiber.Ctx) error {
		start := time.Now()

		isHTMX := respond.IsHTMX(c)
		requestType := "normal"
		if isHTMX {
			requestType = "htmx"
		}

		err := c.Next()

		duration := time.Since(start)
		status := c.Response().StatusCode()

		if status >= 400 {
			slog.Error("HTTP request",
				"type", requestType,
				"method", c.Method(),
				"path", c.Path(),
				"status", status,
				"duration", duration.String(),
				"error", err,
			)
		} else {
			slog.Debug("HTTP request",
				"type", requestType,
				"method", c.Method(),
				"path", c.Path(),
				"status", status,
				"duration", duration.String(),
			)
		}
		return err
	}
}
