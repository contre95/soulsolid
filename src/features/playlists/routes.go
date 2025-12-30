package playlists

import (
	"github.com/gofiber/fiber/v2"
)

// RegisterRoutes registers the routes for the playlists feature.
func RegisterRoutes(app *fiber.App, service *Service) {
	handler := NewHandler(service)

	ui := app.Group("/ui")
	ui.Get("/playlists", handler.RenderPlaylistsSection)
	ui.Get("/playlists/:id", handler.GetPlaylist)

	playlists := app.Group("/playlists")
	playlists.Get("/create-modal", handler.GetCreatePlaylistModal)
	playlists.Post("/", handler.CreatePlaylist)
	playlists.Put("/:id", handler.UpdatePlaylist)
	playlists.Delete("/:id", handler.DeletePlaylist)
	playlists.Post("/items", handler.AddItemToPlaylist)
	playlists.Delete("/:playlistId/tracks/:trackId", handler.RemoveTrackFromPlaylist)
	playlists.Get("/:type/:id/playlists", handler.GetPlaylistsForItem)

	playlists.Get("/:id/export", handler.ExportM3U)
}
