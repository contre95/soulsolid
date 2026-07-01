package merge

import (
	"github.com/gofiber/fiber/v2"
)

// RegisterRoutes registers the routes for the merge feature.
func RegisterRoutes(app *fiber.App, service *Service) {
	handler := NewHandler(service)

	// Navigable analyze section.
	app.Get("/analyze/merge", handler.RenderMergeSection)

	g := app.Group("/merge")
	g.Get("/groups/artists", handler.RenderArtistGroups)
	g.Get("/groups/albums", handler.RenderAlbumGroups)
	g.Get("/groups/genres", handler.RenderGenreGroups)
	g.Post("/artists", handler.MergeArtists)
	g.Post("/albums", handler.MergeAlbums)
	g.Post("/genres", handler.MergeGenres)
}
