package downloading

import (
	"github.com/contre95/soulsolid/src/features/jobs"
	"github.com/gofiber/fiber/v2"
)

// RegisterRoutes registers downloading-related routes
func RegisterRoutes(app *fiber.App, service *Service, jobService jobs.JobService) {
	handler := NewHandler(service, jobService)

	// API routes for downloading
	api := app.Group("/downloads")

	// Search endpoints
	api.Post("/search", handler.Search)
	api.Post("/search/albums", handler.SearchAlbums)
	api.Post("/search/tracks", handler.SearchTracks)

	// Navigation endpoints

	api.Get("/album/:albumId/tracks", handler.GetAlbumTracks)

	// Download endpoints
	api.Post("/track", handler.DownloadTrack)
	api.Post("/album", handler.DownloadAlbum)
	api.Post("/artist", handler.DownloadArtist)
	api.Post("/tracks", handler.DownloadTracks)
	api.Post("/playlist", handler.DownloadPlaylist)

	// Capabilities endpoint
	api.Get("/capabilities", handler.GetDownloaderCapabilities)

	// User info endpoint
	api.Get("/user/info", handler.GetUserInfo)

	ui := app.Group("/ui")
	ui.Get("/downloading/chart/tracks", handler.GetChartTracks)
}
