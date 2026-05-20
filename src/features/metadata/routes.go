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
	tag.Get("/edit/:trackId", handler.RenderTagEditor)
	tag.Get("/edit/:trackId/artwork", handler.ServeArtwork)
	tag.Get("/edit/:trackId/fetch/:provider", handler.FetchFromProvider)
	tag.Get("/edit/:trackId/search/:provider", handler.SearchTracksFromProvider)
	tag.Get("/edit/:trackId/select/:provider", handler.SelectTrackFromResults)
	tag.Get("/edit/:trackId/fingerprint", handler.CalculateFingerprint)
	tag.Get("/edit/:trackId/fingerprint/view", handler.ViewFingerprint)
	tag.Get("/buttons/metadata/:trackId", handler.RenderMetadataButtons)
	tag.Post("/:trackId", handler.UpdateTags)

	analyze := app.Group("/analyze")
	analyze.Post("/acoustid", handler.StartAcoustIDAnalysis)

	ui.Get("/analyze/metadata", handler.RenderMetadataAnalysisSection)
}
