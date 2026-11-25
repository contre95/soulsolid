package albumcompleteness

import (
	"github.com/gofiber/fiber/v3"
)

// RegisterRoutes registers the routes for the album completeness feature.
func RegisterRoutes(app *fiber.App, service *Service) {
	handler := NewHandler(service)

	// Navigable analyze section.
	app.Get("/analyze/completeness", handler.RenderCompletenessSection)

	g := app.Group("/completeness")
	g.Post("/analyze", handler.StartAnalysis)
	g.Get("/incomplete", handler.RenderIncompleteAlbums)
	g.Get("/albums/:id", handler.RenderAlbumBadge)
	g.Post("/albums/:id/check", handler.CheckAlbum)
	g.Post("/albums/:id/expected", handler.SetExpectedTracks)
}
