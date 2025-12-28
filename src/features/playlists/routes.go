package playlists

import (
	"github.com/gofiber/fiber/v2"
)

// RegisterRoutes registers the routes for the playlists feature
func RegisterRoutes(app *fiber.App, handler *Handler) {
	// UI routes for HTMX partials
	ui := app.Group("/ui")
	ui.Get("/playlists", handler.RenderPlaylistsSection)
	ui.Get("/playlists/list", handler.GetPlaylists)
	ui.Get("/playlists/new", handler.RenderCreatePlaylistForm)
	ui.Get("/playlists/add-modal", handler.RenderAddToPlaylistModal)
	ui.Get("/playlists/:id", handler.GetPlaylist)

	// API routes for playlist operations
	api := app.Group("/api/playlists")
	api.Get("", handler.GetPlaylists)
	api.Post("", handler.CreatePlaylist)
	api.Get("/:id", handler.GetPlaylist)
	api.Put("/:id", handler.UpdatePlaylist)
	api.Delete("/:id", handler.DeletePlaylist)

	// Track management routes
	api.Post("/:id/tracks", handler.AddTrackToPlaylist)
	api.Delete("/:playlistId/tracks/:trackId", handler.RemoveTrackFromPlaylist)

	// M3U import/export routes
	api.Post("/import/m3u", handler.ImportM3U)
	api.Get("/:id/export/m3u", handler.ExportM3U)
}
