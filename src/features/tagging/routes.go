package tagging

import (
	"github.com/gofiber/fiber/v2"
)

// RegisterRoutes registers the routes for the tag feature
func RegisterRoutes(app *fiber.App, service *Service) {
	handler := NewHandler(service)

	// UI routes for page rendering
	ui := app.Group("/ui")
	ui.Get("/tag/edit/:trackId", handler.RenderTagEditor)
	ui.Get("/tag/edit/:trackId/fetch/:provider", handler.FetchFromProvider)
	ui.Get("/tag/edit/:trackId/search/:provider", handler.SearchTracksFromProvider)
	ui.Get("/tag/edit/:trackId/select/:provider", handler.SelectTrackFromResults)
	ui.Get("/tag/edit/:trackId/lyrics/:provider", handler.FetchLyricsFromProvider)

	// API routes for data operations
	tagGroup := app.Group("/tag")
	tagGroup.Get("/edit/:trackId", handler.RenderTagEditor)
	tagGroup.Get("/tag/edit/:trackId/fetch/:provider", handler.FetchFromProvider)
	tagGroup.Get("/tag/edit/:trackId/search/:provider", handler.SearchTracksFromProvider)
	tagGroup.Get("/tag/edit/:trackId/select/:provider", handler.SelectTrackFromResults)
	tagGroup.Get("/tag/edit/:trackId/lyrics/:provider", handler.FetchLyricsFromProvider)
	tagGroup.Post("/:trackId", handler.UpdateTags)
}
