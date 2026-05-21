package metadata

import (
	"github.com/gofiber/fiber/v2"
)

// RegisterRoutes registers the routes for the tag feature
func RegisterRoutes(app *fiber.App, service *Service) {
	handler := NewHandler(service)

	// UI routes for page rendering
	ui := app.Group("/ui")
	tag := app.Group("/tag")
	tag.Get("/buttons/metadata/:trackId", handler.RenderMetadataButtons)
	tag.Get("/:trackId/artwork", handler.ServeArtwork)
	tag.Get("/:trackId/fingerprint", handler.CalculateFingerprint)
	tag.Get("/:trackId/fingerprint/view", handler.ViewFingerprint)
	tag.Get("/:trackId/search/:provider", handler.SearchTracksFromProvider)
	tag.Get("/:trackId/select/:provider", handler.SelectTrackFromResults)
	tag.Get("/:trackId", handler.RenderTagEditor)
	tag.Post("/:trackId", handler.UpdateTags)
	tag.Get("/:trackId/:provider", handler.FetchFromProvider)

	analyze := app.Group("/analyze")
	analyze.Post("/acoustid", handler.StartAcoustIDAnalysis)

	ui.Get("/analyze/metadata", handler.RenderMetadataAnalysisSection)
}
