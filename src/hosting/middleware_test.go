package hosting

import (
	"bytes"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
)

// The response status picks the level: 5xx is a server fault, 4xx is the
// client's, and anything else is routine traffic.
func TestLogAllRequestsMiddlewareLevelFollowsStatus(t *testing.T) {
	previous := slog.Default()
	t.Cleanup(func() { slog.SetDefault(previous) })

	cases := []struct {
		status int
		level  string
	}{
		{200, "DEBUG"},
		{304, "DEBUG"},
		{404, "WARN"},
		{422, "WARN"},
		{500, "ERROR"},
		{503, "ERROR"},
	}

	for _, tc := range cases {
		var logged bytes.Buffer
		slog.SetDefault(slog.New(slog.NewTextHandler(&logged, &slog.HandlerOptions{Level: slog.LevelDebug})))

		app := fiber.New()
		app.Use(LogAllRequestsMiddleware())
		app.Get("/artwork", func(c fiber.Ctx) error { return c.SendStatus(tc.status) })

		if _, err := app.Test(httptest.NewRequest("GET", "/artwork", nil)); err != nil {
			t.Fatalf("status %d: request failed: %v", tc.status, err)
		}

		line := logged.String()
		if !strings.Contains(line, "level="+tc.level) {
			t.Errorf("status %d: expected level=%s, got %q", tc.status, tc.level, line)
		}
		if !strings.Contains(line, "path=/artwork") {
			t.Errorf("status %d: expected the path in the line, got %q", tc.status, line)
		}
	}
}
