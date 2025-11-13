package hosting

import (
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v2"
)

var debugPaths = []string{"/jobs/", "/ui/jobs/"}

// HTMXMiddleware creates middleware for logging HTMX requests
func HTMXMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		// Check if this is an HTMX request
		isHTMX := c.Get("HX-Request") == "true"

		// Process the request
		err := c.Next()

		// Log HTMX-specific information
		if isHTMX {
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

// HTMXDebugMiddleware provides detailed debugging for HTMX requests
func HTMXDebugMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		isHTMX := c.Get("HX-Request") == "true"

		if isHTMX {
			slog.Debug("HTMX request received",
				"method", c.Method(),
				"path", c.Path(),
				"headers", getHTMXHeaders(c),
			)
		}

		err := c.Next()

		if isHTMX && err == nil {
			slog.Debug("HTMX response sent",
				"status", c.Response().StatusCode(),
				"response_headers", getHTMXResponseHeaders(c),
			)
		}

		return err
	}
}

// getHTMXHeaders extracts all HTMX-related headers from the request
func getHTMXHeaders(c *fiber.Ctx) map[string]string {
	headers := make(map[string]string)

	// List of HTMX headers to check
	htmxHeaders := []string{
		"HX-Request",
		"HX-Trigger",
		"HX-Trigger-Name",
		"HX-Target",
		"HX-Current-URL",
		"HX-Prompt",
		"HX-Boosted",
		"HX-History-Restore-Request",
	}

	for _, header := range htmxHeaders {
		if value := c.Get(header); value != "" {
			headers[header] = value
		}
	}

	return headers
}

// getHTMXResponseHeaders extracts HTMX-related response headers
func getHTMXResponseHeaders(c *fiber.Ctx) map[string]string {
	headers := make(map[string]string)

	// Check for HTMX response headers
	response := c.Response()
	htmxResponseHeaders := []string{
		"HX-Location",
		"HX-Push-Url",
		"HX-Redirect",
		"HX-Refresh",
		"HX-Replace-Url",
		"HX-Reswap",
		"HX-Retarget",
		"HX-Reselect",
		"HX-Trigger",
		"HX-Trigger-After-Settle",
		"HX-Trigger-After-Swap",
	}

	for _, header := range htmxResponseHeaders {
		if value := response.Header.Peek(header); len(value) > 0 {
			headers[header] = string(value)
		}
	}

	return headers
}

// LogAllRequestsMiddleware logs all requests with HTMX context
func LogAllRequestsMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		isHTMX := c.Get("HX-Request") == "true"
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
