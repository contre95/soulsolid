package library

import (
	"github.com/gofiber/fiber/v2"
)

// RegisterRoutes registers the routes for the library feature.
func RegisterRoutes(app *fiber.App, service *Service) {
	handler := NewHandler(service)

	app.Get("/library", handler.RenderLibrarySection)

	library := app.Group("/library")
	library.Get("/table", handler.GetLibraryTable)
	library.Get("/tracks/:trackId/overview", handler.RenderTrackOverviewPanel)
	library.Get("/search", handler.GetUnifiedSearch)
	library.Get("/artists/count", handler.GetArtistsCount)
	library.Get("/albums/count", handler.GetAlbumsCount)
	library.Get("/tracks/count", handler.GetTracksCount)
	library.Get("/storage/size", handler.GetStorageSize)
	library.Get("/artists/:id", handler.GetArtist)
	library.Get("/albums/:id", handler.GetAlbum)
	library.Get("/tracks/:id", handler.GetTrack)
	library.Get("/tree", handler.GetLibraryFileTree)
	library.Delete("/tracks/:trackId", handler.DeleteTrack)
	library.Delete("/albums/:albumId", handler.DeleteAlbum)
	library.Delete("/artists/:artistId", handler.DeleteArtist)
}
