package metadata

import (
	"github.com/gofiber/fiber/v2"
)

// RegisterRoutes registers the routes for the tag feature
func RegisterRoutes(app *fiber.App, service *Service) {
	handler := NewHandler(service)

	tag := app.Group("/tag")
	tag.Get("/:trackId/metadata", handler.GetMetadataProviders)
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

	app.Get("/analyze/metadata", handler.RenderMetadataAnalysisSection)
}
