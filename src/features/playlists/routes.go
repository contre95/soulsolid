package playlists

import (
	"github.com/gofiber/fiber/v2"
)

// RegisterRoutes registers the routes for the playlists feature.
func RegisterRoutes(app *fiber.App, service *Service) {
	handler := NewHandler(service)

	// UI routes — always return HTML (full page or HTMX partial)
	ui := app.Group("/ui")
	ui.Get("/playlists", handler.RenderPlaylistsSection)
	ui.Get("/playlists/:id", handler.RenderPlaylist)

	// HTMX UI component routes (modals, fragments) — static segments first
	api := app.Group("/playlists")
	api.Get("/create-modal", handler.GetPlaylistCreationModal)
	api.Get("/:type/:id/playlists", handler.GetPlaylistsForItem)
	api.Get("/:id/export", handler.ExportM3U)

	// API routes — always return JSON
	api.Get("/", handler.GetAllPlaylists)
	api.Get("/:id", handler.GetPlaylist)

	// Mutation routes — return JSON for API callers, toast for HTMX
	api.Post("/", handler.CreatePlaylist)
	api.Post("/items", handler.AddItemToPlaylist)
	api.Put("/:id", handler.UpdatePlaylist)
	api.Delete("/:id", handler.DeletePlaylist)
	api.Delete("/:playlistId/tracks/:trackId", handler.RemoveTrackFromPlaylist)
}
