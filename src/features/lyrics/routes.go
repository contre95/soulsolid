package lyrics

import (
	"github.com/gofiber/fiber/v2"
)

// RegisterRoutes registers lyrics routes
func RegisterRoutes(app *fiber.App, handler *Handler) {
	// UI routes for HTMX partials
	ui := app.Group("/ui")
	// Lyrics queue UI routes
	ui.Get("/lyrics/queue/header", handler.RenderLyricsQueueHeader)
	ui.Get("/lyrics/queue/items", handler.RenderLyricsQueueItems)
	ui.Get("/lyrics/queue/items/grouped", handler.RenderGroupedLyricsQueueItems)
	tagGroup := ui.Group("/tag")

	// Lyrics routes - these are accessed from the metadata/tag UI
	tagGroup.Get("/edit/:trackId/lyrics/text/:provider", handler.GetLyricsText)
	tagGroup.Get("/buttons/lyrics/:trackId", handler.RenderLyricsButtons)

	// Library routes for lyrics
	library := app.Group("/library")
	library.Get("/tracks/:id/lyrics", handler.GetTrackLyrics)

	// Lyrics queue routes
	queue := app.Group("/lyrics/queue")
	queue.Get("/items", handler.RenderLyricsQueueItems)
	queue.Get("/items/grouped", handler.RenderGroupedLyricsQueueItems)
	queue.Post("/:id/:action", handler.ProcessLyricsQueueItem)
	queue.Post("/group/:groupType/:groupKey/:action", handler.ProcessLyricsQueueGroup)
	queue.Post("/clear", handler.ClearLyricsQueue)
	queue.Get("/count", handler.LyricsQueueCount)
	queue.Get("/:id/new_lyrics", handler.GetQueueNewLyrics)

	// Analyze routes - lyrics analysis
	analyze := app.Group("/analyze")
	analyze.Post("/lyrics", handler.StartLyricsAnalysis)

	// UI routes for lyrics analysis section
	ui.Get("/analyze/lyrics", handler.RenderLyricsAnalysisSection)
}
