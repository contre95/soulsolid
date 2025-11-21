package library

import (
	"github.com/gofiber/fiber/v2"
)

// RegisterRoutes registers the routes for the library feature.
func RegisterRoutes(app *fiber.App, service *Service) {
	handler := NewHandler(service)

	ui := app.Group("/ui")
	ui.Get("/library/table", handler.GetLibraryTable)

	library := app.Group("/library")
	library.Get("/artists/count", handler.GetArtistsCount)
	library.Get("/albums/count", handler.GetAlbumsCount)
	library.Get("/tracks/count", handler.GetTracksCount)
	library.Get("/artists", handler.GetArtists)
	library.Get("/albums", handler.GetAlbums)
	library.Get("/tracks", handler.GetTracks)
	library.Get("/artists/:id", handler.GetArtist)
	library.Get("/albums/:id", handler.GetAlbum)
	library.Get("/tracks/:id", handler.GetTrack)
	library.Get("/tree", handler.GetLibraryFileTree)
}
