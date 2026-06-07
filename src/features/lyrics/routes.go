package lyrics

import (
	"github.com/gofiber/fiber/v2"
)

// RegisterRoutes registers lyrics routes
func RegisterRoutes(app *fiber.App, handler *Handler) {
	tag := app.Group("/tag")
	tag.Get("/:trackId/lyrics", handler.GetLyricsProviders)
	tag.Get("/:trackId/lyrics/text/:provider", handler.GetLyricsText)

	library := app.Group("/library")
	library.Get("/tracks/:id/lyrics", handler.GetTrackLyrics)

	queue := app.Group("/lyrics/queue")
	queue.Get("/header", handler.RenderLyricsQueueHeader)
	queue.Get("/items", handler.RenderLyricsQueueItems)
	queue.Get("/items/grouped", handler.RenderGroupedLyricsQueueItems)
	queue.Post("/:id/:action", handler.ProcessLyricsQueueItem)
	queue.Post("/group/:groupType/:groupKey/:action", handler.ProcessLyricsQueueGroup)
	queue.Post("/clear", handler.ClearLyricsQueue)
	queue.Get("/count", handler.LyricsQueueCount)
	queue.Get("/:id/new_lyrics", handler.GetQueueNewLyrics)

	analyze := app.Group("/analyze")
	analyze.Post("/lyrics", handler.StartLyricsAnalysis)
	analyze.Get("/lyrics", handler.RenderLyricsAnalysisSection)
}
