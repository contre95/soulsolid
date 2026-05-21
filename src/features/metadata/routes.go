package metadata

import (
	"github.com/gofiber/fiber/v2"
)

// RegisterRoutes registers the routes for the tag feature.
// Note: the /:trackId/:provider catch-all is NOT registered here — call
// RegisterProviderCatchAll after all other /tag routes (e.g. lyrics) are registered,
// so those more specific routes are matched first by Fiber.
func RegisterRoutes(app *fiber.App, service *Service) *Handler {
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

	analyze := app.Group("/analyze")
	analyze.Post("/acoustid", handler.StartAcoustIDAnalysis)

	app.Get("/analyze/metadata", handler.RenderMetadataAnalysisSection)

	return handler
}

// RegisterProviderCatchAll registers the /:trackId/:provider catch-all route.
// Must be called after all other /tag routes are registered (including lyrics routes).
func RegisterProviderCatchAll(app *fiber.App, handler *Handler) {
	app.Get("/tag/:trackId/:provider", handler.FetchFromProvider)
}
