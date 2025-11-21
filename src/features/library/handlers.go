package library

import (
	"fmt"
	"log/slog"

	"strings"

	"github.com/gofiber/fiber/v2"
)

// Handler is the handler for the library feature.
type Handler struct {
	service *Service
}

// NewHandler creates a new handler for the library feature.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Pagination represents pagination information
type Pagination struct {
	Page       int
	Limit      int
	TotalCount int
	TotalPages int
	NextPage   int
	PrevPage   int
	HasNext    bool
	HasPrev    bool
}

// NewPagination creates a new Pagination instance with calculated values
func NewPagination(page, limit, totalCount int) Pagination {
	totalPages := (totalCount + limit - 1) / limit
	nextPage := page + 1
	prevPage := page - 1

	return Pagination{
		Page:       page,
		Limit:      limit,
		TotalCount: totalCount,
		TotalPages: totalPages,
		NextPage:   nextPage,
		PrevPage:   prevPage,
		HasNext:    page < totalPages,
		HasPrev:    page > 1,
	}
}

// GetArtists is the handler for getting all artists.
func (h *Handler) GetArtists(c *fiber.Ctx) error {
	slog.Debug("GetArtists handler called")

	// Check if pagination parameters are provided
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 50)

	// Always use pagination to avoid loading all records
	if true {
		// Use paginated version
		offset := (page - 1) * limit
		artists, err := h.service.GetArtistsPaginated(c.Context(), limit, offset)
		if err != nil {
			slog.Error("Error loading paginated artists", "error", err)
			return c.Status(fiber.StatusInternalServerError).SendString("Error loading artists")
		}

		// Get total count for pagination
		totalCount, err := h.service.GetArtistsCount(c.Context())
		if err != nil {
			slog.Error("Error getting artists count", "error", err)
			return c.Status(fiber.StatusInternalServerError).SendString("Error loading artists")
		}

		// Check if the request accepts HTML (like an HTMX request)
		acceptHeader := c.Get("Accept")
		hxRequest := c.Get("HX-Request")
		if strings.Contains(acceptHeader, "text/html") || hxRequest == "true" {
			pagination := NewPagination(page, limit, totalCount)
			return c.Render("library/artists_list", fiber.Map{
				"Artists":    artists,
				"Pagination": pagination,
			})
		}

		return c.JSON(fiber.Map{
			"artists": artists,
			"pagination": fiber.Map{
				"page":       page,
				"limit":      limit,
				"totalCount": totalCount,
				"totalPages": (totalCount + limit - 1) / limit,
			},
		})
	}

	// Fall back to getting all artists
	artists, err := h.service.GetArtists(c.Context())
	if err != nil {
		slog.Error("Error loading artists", "error", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Error loading artists")
	}

	// Check if the request accepts HTML (like an HTMX request)
	acceptHeader := c.Get("Accept")
	hxRequest := c.Get("HX-Request")
	if strings.Contains(acceptHeader, "text/html") || hxRequest == "true" {
		return c.Render("library/artists_list", fiber.Map{
			"Artists": artists,
			"Pagination": fiber.Map{
				"Page":       1,
				"Limit":      50,
				"TotalCount": len(artists),
				"TotalPages": 1,
			},
		})
	}

	return c.JSON(artists)
}

// GetAlbums is the handler for getting all albums.
func (h *Handler) GetAlbums(c *fiber.Ctx) error {
	slog.Debug("GetAlbums handler called")

	// Check if pagination parameters are provided
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 50)

	// Always use pagination to avoid loading all records
	if true {
		// Use paginated version
		offset := (page - 1) * limit
		albums, err := h.service.GetAlbumsPaginated(c.Context(), limit, offset)
		if err != nil {
			slog.Error("Error loading paginated albums", "error", err)
			return c.Status(fiber.StatusInternalServerError).SendString("Error loading albums")
		}

		// Get total count for pagination
		totalCount, err := h.service.GetAlbumsCount(c.Context())
		if err != nil {
			slog.Error("Error getting albums count", "error", err)
			return c.Status(fiber.StatusInternalServerError).SendString("Error loading albums")
		}

		// Check if the request accepts HTML (like an HTMX request)
		acceptHeader := c.Get("Accept")
		hxRequest := c.Get("HX-Request")
		if strings.Contains(acceptHeader, "text/html") || hxRequest == "true" {
			pagination := NewPagination(page, limit, totalCount)
			return c.Render("library/albums_list", fiber.Map{
				"Albums":     albums,
				"Pagination": pagination,
			})
		}

		return c.JSON(fiber.Map{
			"albums": albums,
			"pagination": fiber.Map{
				"page":       page,
				"limit":      limit,
				"totalCount": totalCount,
				"totalPages": (totalCount + limit - 1) / limit,
			},
		})
	}

	// Fall back to getting all albums
	albums, err := h.service.GetAlbums(c.Context())
	if err != nil {
		slog.Error("Error loading albums", "error", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Error loading albums")
	}

	// Check if the request accepts HTML (like an HTMX request)
	acceptHeader := c.Get("Accept")
	hxRequest := c.Get("HX-Request")
	if strings.Contains(acceptHeader, "text/html") || hxRequest == "true" {
		return c.Render("library/albums_list", fiber.Map{
			"Albums": albums,
			"Pagination": fiber.Map{
				"Page":       1,
				"Limit":      50,
				"TotalCount": len(albums),
				"TotalPages": 1,
			},
		})
	}

	return c.JSON(albums)
}

// GetTracks is the handler for getting all tracks.
func (h *Handler) GetTracks(c *fiber.Ctx) error {
	slog.Debug("GetTracks handler called")

	// Check if pagination parameters are provided
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 50)

	// Always use pagination to avoid loading all records
	if true {
		// Use paginated version
		offset := (page - 1) * limit
		tracks, err := h.service.GetTracksPaginated(c.Context(), limit, offset)
		if err != nil {
			slog.Error("Error loading paginated tracks", "error", err)
			return c.Status(fiber.StatusInternalServerError).SendString("Error loading tracks")
		}

		// Get total count for pagination
		totalCount, err := h.service.GetTracksCount(c.Context())
		if err != nil {
			slog.Error("Error getting tracks count", "error", err)
			return c.Status(fiber.StatusInternalServerError).SendString("Error loading tracks")
		}

		// Check if the request accepts HTML (like an HTMX request)
		acceptHeader := c.Get("Accept")
		hxRequest := c.Get("HX-Request")
		if strings.Contains(acceptHeader, "text/html") || hxRequest == "true" {
			pagination := NewPagination(page, limit, totalCount)
			return c.Render("library/tracks_list", fiber.Map{
				"Tracks":     tracks,
				"Pagination": pagination,
			})
		}

		return c.JSON(fiber.Map{
			"tracks": tracks,
			"pagination": fiber.Map{
				"page":       page,
				"limit":      limit,
				"totalCount": totalCount,
				"totalPages": (totalCount + limit - 1) / limit,
			},
		})
	}

	// Fall back to getting all tracks
	tracks, err := h.service.GetTracks(c.Context())
	if err != nil {
		slog.Error("Error loading tracks", "error", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Error loading tracks")
	}

	// Check if the request accepts HTML (like an HTMX request)
	acceptHeader := c.Get("Accept")
	hxRequest := c.Get("HX-Request")
	if strings.Contains(acceptHeader, "text/html") || hxRequest == "true" {
		return c.Render("library/tracks_list", fiber.Map{
			"Tracks": tracks,
			"Pagination": fiber.Map{
				"Page":       1,
				"Limit":      50,
				"TotalCount": len(tracks),
				"TotalPages": 1,
			},
		})
	}

	return c.JSON(tracks)
}

// GetArtist is the handler for getting a single artist.
func (h *Handler) GetArtist(c *fiber.Ctx) error {
	slog.Debug("GetArtist handler called", "id", c.Params("id"))
	artist, err := h.service.GetArtist(c.Context(), c.Params("id"))
	if err != nil {
		slog.Error("Error loading artist", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	return c.JSON(artist)
}

// GetAlbum is the handler for getting a single album.
func (h *Handler) GetAlbum(c *fiber.Ctx) error {
	slog.Debug("GetAlbum handler called", "id", c.Params("id"))
	album, err := h.service.GetAlbum(c.Context(), c.Params("id"))
	if err != nil {
		slog.Error("Error loading album", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	return c.JSON(album)
}

// GetTrack is the handler for getting a single track.
func (h *Handler) GetTrack(c *fiber.Ctx) error {
	slog.Debug("GetTrack handler called", "id", c.Params("id"))
	track, err := h.service.GetTrack(c.Context(), c.Params("id"))
	if err != nil {
		slog.Error("Error loading track", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	return c.JSON(track)
}

// GetArtistsCount returns the count of artists in the library.
func (h *Handler) GetArtistsCount(c *fiber.Ctx) error {
	slog.Debug("GetArtistsCount handler called")
	artists, err := h.service.GetArtists(c.Context())
	if err != nil {
		slog.Error("Error loading artists count", "error", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Error loading artists count")
	}
	return c.SendString(fmt.Sprintf("%d", len(artists)))
}

// GetAlbumsCount returns the count of albums in the library.
func (h *Handler) GetAlbumsCount(c *fiber.Ctx) error {
	slog.Debug("GetAlbumsCount handler called")
	albums, err := h.service.GetAlbums(c.Context())
	if err != nil {
		slog.Error("Error loading albums count", "error", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Error loading albums count")
	}
	return c.SendString(fmt.Sprintf("%d", len(albums)))
}

// GetTracksCount returns the count of tracks in the library.
func (h *Handler) GetTracksCount(c *fiber.Ctx) error {
	slog.Debug("GetTracksCount handler called")
	tracks, err := h.service.GetTracks(c.Context())
	if err != nil {
		slog.Error("Error loading tracks count", "error", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Error loading tracks count")
	}
	return c.SendString(fmt.Sprintf("%d", len(tracks)))
}

// GetLibraryTable renders the library table section with tabs.
func (h *Handler) GetLibraryTable(c *fiber.Ctx) error {
	slog.Debug("GetLibraryTable handler called")
	return c.Render("library/library_table", fiber.Map{})
}

// GetLibraryFileTree returns a tree structure of the library path.
func (h *Handler) GetLibraryFileTree(c *fiber.Ctx) error {
	slog.Debug("GetLibraryFileTree handler called")
	var tree string
	var err error
	folder := c.Query("folder", "library")
	switch folder {
	case "library":
		tree, err = h.service.GetLibraryFileTree()
	case "downloads":
		tree, err = h.service.GetDownloadsFileTree()
	}
	if err != nil {
		slog.Error("Error getting library file tree", "error", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to get library file tree")
	}
	return c.SendString(tree)
}
