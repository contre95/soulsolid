package downloading

import (
	"github.com/gofiber/fiber/v2"
)

// RegisterRoutes registers downloading-related routes
func RegisterRoutes(app *fiber.App, service *Service) {
	handler := NewHandler(service)

	app.Get("/downloads", handler.RenderDownloadSection)

	downloads := app.Group("/downloads")
	downloads.Post("/search", handler.Search)
	downloads.Post("/search/albums", handler.SearchAlbums)
	downloads.Post("/search/tracks", handler.SearchTracks)
	downloads.Get("/album/:albumId/tracks", handler.GetAlbumTracks)
	downloads.Post("/track", handler.DownloadTrack)
	downloads.Post("/album", handler.DownloadAlbum)
	downloads.Post("/artist", handler.DownloadArtist)
	downloads.Post("/tracks", handler.DownloadTracks)
	downloads.Post("/playlist", handler.DownloadPlaylist)
	downloads.Get("/capabilities", handler.GetDownloaderCapabilities)
	downloads.Get("/user/info", handler.GetUserInfo)
	downloads.Get("/chart/tracks", handler.GetChartTracks)
}
