package library

import (
	"fmt"
	"log/slog"

	"strings"

	"github.com/contre95/soulsolid/src/music"
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

// RenderLibrarySection renders the library page.
func (h *Handler) RenderLibrarySection(c *fiber.Ctx) error {
	slog.Debug("RenderLibrary handler called")

	// Fetch all artists and albums for search form
	artists, err := h.service.GetArtists(c.Context())
	if err != nil {
		slog.Error("Error loading artists for search form", "error", err)
		artists = []*music.Artist{} // Continue with empty list
	}

	albums, err := h.service.GetAlbums(c.Context())
	if err != nil {
		slog.Error("Error loading albums for search form", "error", err)
		albums = []*music.Album{} // Continue with empty list
	}

	data := fiber.Map{
		"Title":               "Library",
		"DefaultDownloadPath": h.service.configManager.Get().DownloadPath,
		"SearchArtists":       artists,
		"SearchAlbums":        albums,
	}
	if c.Get("HX-Request") != "true" {
		data["Section"] = "library"
		return c.Render("main", data)
	}
	return c.Render("sections/library", data)
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

	// Fetch all artists and albums for search form
	artists, err := h.service.GetArtists(c.Context())
	if err != nil {
		slog.Error("Error loading artists for search form", "error", err)
		artists = []*music.Artist{} // Continue with empty list
	}

	albums, err := h.service.GetAlbums(c.Context())
	if err != nil {
		slog.Error("Error loading albums for search form", "error", err)
		albums = []*music.Album{} // Continue with empty list
	}

	return c.Render("library/library_table", fiber.Map{
		"SearchArtists": artists,
		"SearchAlbums":  albums,
	})
}

// SearchResult represents a unified search result item
type SearchResult struct {
	Type        string // "artist", "album", "track"
	ID          string
	PrimaryName string // Artist name, Album title, Track title
	Secondary   string // Artist ID, Album artist names, Track artist names
	Tertiary    string // "", Album year, Track album title
	Duration    int    // Track duration in seconds (for tracks only)
	ImageURL    string // Image for display
}

// GetUnifiedSearch performs a unified search across artists, albums, and tracks.
func (h *Handler) GetUnifiedSearch(c *fiber.Ctx) error {
	slog.Debug("GetUnifiedSearch handler called")

	query := strings.TrimSpace(c.Query("query", ""))
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 50)

	var results []SearchResult
	var totalCount int

	offset := (page - 1) * limit

	if query == "" {
		// When no query, show all artists, albums, and tracks combined
		artists, err := h.service.GetArtists(c.Context())
		if err != nil {
			slog.Error("Error loading artists", "error", err)
			return c.Status(fiber.StatusInternalServerError).SendString("Error loading artists")
		}

		albums, err := h.service.GetAlbums(c.Context())
		if err != nil {
			slog.Error("Error loading albums", "error", err)
			return c.Status(fiber.StatusInternalServerError).SendString("Error loading albums")
		}

		artistsCount := len(artists)
		albumsCount := len(albums)
		totalOther := artistsCount + albumsCount

		tracksCount, err := h.service.GetTracksCount(c.Context())
		if err != nil {
			slog.Error("Error getting tracks count", "error", err)
			return c.Status(fiber.StatusInternalServerError).SendString("Error loading tracks count")
		}

		totalCount = totalOther + tracksCount

		// Determine the slice for this page
		start := offset
		end := min(offset+limit, totalCount)

		// Add artists
		if start < artistsCount {
			artistEnd := min(end, artistsCount)
			for i := start; i < artistEnd; i++ {
				artist := artists[i]
				results = append(results, SearchResult{
					Type:        "artist",
					ID:          artist.ID,
					PrimaryName: artist.Name,
					Secondary:   artist.ID,
					Tertiary:    "",
					ImageURL:    "",
				})
			}
		}

		// Add albums
		if end > artistsCount {
			albumStart := 0
			if start > artistsCount {
				albumStart = start - artistsCount
			}
			albumEnd := min(end-artistsCount, albumsCount)
			for i := albumStart; i < albumEnd; i++ {
				album := albums[i]
				var artistNames strings.Builder
				if len(album.Artists) > 0 {
					for j, ar := range album.Artists {
						if j > 0 {
							artistNames.WriteString(", ")
						}
						artistNames.WriteString(ar.Artist.Name)
					}
				}
				year := ""
				if !album.ReleaseDate.IsZero() {
					year = fmt.Sprintf("%d", album.ReleaseDate.Year())
				}
				results = append(results, SearchResult{
					Type:        "album",
					ID:          album.ID,
					PrimaryName: album.Title,
					Secondary:   artistNames.String(),
					Tertiary:    year,
					ImageURL:    album.ImageSmall,
				})
			}
		}

		// Add tracks
		if end > totalOther {
			trackStart := 0
			if start > totalOther {
				trackStart = start - totalOther
			}
			trackLimit := end - totalOther - trackStart
			if trackLimit > 0 {
				tracks, err := h.service.GetTracksPaginated(c.Context(), trackLimit, trackStart)
				if err != nil {
					slog.Error("Error loading tracks", "error", err)
					return c.Status(fiber.StatusInternalServerError).SendString("Error loading tracks")
				}
				for _, track := range tracks {
					var artistNames strings.Builder
					if len(track.Artists) > 0 {
						for i, ar := range track.Artists {
							if i > 0 {
								artistNames.WriteString(", ")
							}
							artistNames.WriteString(ar.Artist.Name)
						}
					}
					albumTitle := ""
					if track.Album != nil {
						albumTitle = track.Album.Title
					}
					results = append(results, SearchResult{
						Type:        "track",
						ID:          track.ID,
						PrimaryName: track.Title,
						Secondary:   artistNames.String(),
						Tertiary:    albumTitle,
						Duration:    track.Metadata.Duration,
						ImageURL:    "",
					})
				}
			}
		}
	} else {
		// Search artists
		artists, err := h.service.GetArtistsFilteredPaginated(c.Context(), 20, 0, query)
		if err != nil {
			slog.Error("Error searching artists", "error", err)
		} else {
			for _, artist := range artists {
				results = append(results, SearchResult{
					Type:        "artist",
					ID:          artist.ID,
					PrimaryName: artist.Name,
					Secondary:   artist.ID,
					Tertiary:    "",
					ImageURL:    "",
				})
			}
		}

		// Search albums
		albums, err := h.service.GetAlbumsFilteredPaginated(c.Context(), 20, 0, query, []string{})
		if err != nil {
			slog.Error("Error searching albums", "error", err)
		} else {
			for _, album := range albums {
				var artistNames strings.Builder
				if len(album.Artists) > 0 {
					for i, ar := range album.Artists {
						if i > 0 {
							artistNames.WriteString(", ")
						}
						artistNames.WriteString(ar.Artist.Name)
					}
				}
				year := ""
				if !album.ReleaseDate.IsZero() {
					year = fmt.Sprintf("%d", album.ReleaseDate.Year())
				}
				results = append(results, SearchResult{
					Type:        "album",
					ID:          album.ID,
					PrimaryName: album.Title,
					Secondary:   artistNames.String(),
					Tertiary:    year,
					ImageURL:    album.ImageSmall,
				})
			}
		}

		// Search tracks
		filter := &music.TrackFilter{
			Title:     query,
			ArtistIDs: []string{},
			AlbumIDs:  []string{},
		}
		tracks, err := h.service.GetTracksFilteredPaginated(c.Context(), 20, 0, filter)
		if err != nil {
			slog.Error("Error searching tracks", "error", err)
		} else {
			for _, track := range tracks {
				var artistNames strings.Builder
				if len(track.Artists) > 0 {
					for i, ar := range track.Artists {
						if i > 0 {
							artistNames.WriteString(", ")
						}
						artistNames.WriteString(ar.Artist.Name)
					}
				}
				albumTitle := ""
				if track.Album != nil {
					albumTitle = track.Album.Title
				}
				results = append(results, SearchResult{
					Type:        "track",
					ID:          track.ID,
					PrimaryName: track.Title,
					Secondary:   artistNames.String(),
					Tertiary:    albumTitle,
					Duration:    track.Metadata.Duration,
					ImageURL:    "",
				})
			}
		}

		// For search results, paginate the collected results
		totalCount = len(results)
		start := (page - 1) * limit
		end := start + limit
		if start > totalCount {
			start = totalCount
		}
		if end > totalCount {
			end = totalCount
		}
		results = results[start:end]
	}

	pagination := NewPagination(page, limit, totalCount)

	// Check if the request accepts HTML (like an HTMX request)
	acceptHeader := c.Get("Accept")
	hxRequest := c.Get("HX-Request")
	if strings.Contains(acceptHeader, "text/html") || hxRequest == "true" {
		return c.Render("library/unified_search_list", fiber.Map{
			"Results":    results,
			"Pagination": pagination,
			"Query":      query,
		})
	}

	return c.JSON(fiber.Map{
		"results": results,
		"pagination": fiber.Map{
			"page":       page,
			"limit":      limit,
			"totalCount": totalCount,
			"totalPages": (totalCount + limit - 1) / limit,
		},
	})
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

// DeleteTrack deletes a track from the library.
func (h *Handler) DeleteTrack(c *fiber.Ctx) error {
	slog.Debug("DeleteTrack handler called", "trackId", c.Params("trackId"))

	trackID := c.Params("trackId")
	if trackID == "" {
		return c.Status(fiber.StatusBadRequest).SendString("Track ID is required")
	}

	err := h.service.DeleteTrack(c.Context(), trackID)
	if err != nil {
		slog.Error("Failed to delete track", "error", err, "trackId", trackID)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to delete track")
	}

	// Return success toast
	return c.Render("toast/toastOk", fiber.Map{"Msg": "Track deleted successfully"})
}

// DeleteAlbum deletes an album from the library.
func (h *Handler) DeleteAlbum(c *fiber.Ctx) error {
	slog.Debug("DeleteAlbum handler called", "albumId", c.Params("albumId"))

	albumID := c.Params("albumId")
	if albumID == "" {
		return c.Status(fiber.StatusBadRequest).SendString("Album ID is required")
	}

	err := h.service.DeleteAlbum(c.Context(), albumID)
	if err != nil {
		slog.Error("Failed to delete album", "error", err, "albumId", albumID)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to delete album")
	}

	// Return success toast
	return c.Render("toast/toastOk", fiber.Map{"Msg": "Album deleted successfully"})
}

// DeleteArtist deletes an artist from the library.
func (h *Handler) DeleteArtist(c *fiber.Ctx) error {
	slog.Debug("DeleteArtist handler called", "artistId", c.Params("artistId"))

	artistID := c.Params("artistId")
	if artistID == "" {
		return c.Status(fiber.StatusBadRequest).SendString("Artist ID is required")
	}

	err := h.service.DeleteArtist(c.Context(), artistID)
	if err != nil {
		slog.Error("Failed to delete artist", "error", err, "artistId", artistID)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to delete artist")
	}

	// Return success toast
	return c.Render("toast/toastOk", fiber.Map{"Msg": "Artist deleted successfully"})
}

// RenderTagEditForm renders the tag edit form for a track
func (h *Handler) RenderTagEditForm(c *fiber.Ctx) error {
	slog.Debug("RenderTagEditForm handler called", "trackId", c.Params("trackId"))

	trackID := c.Params("trackId")
	if trackID == "" {
		return c.Status(fiber.StatusBadRequest).SendString("Track ID is required")
	}

	// Get track data for editing
	track, err := h.service.GetTrack(c.Context(), trackID)
	if err != nil || track == nil {
		slog.Error("Failed to get track for editing", "error", err, "trackId", trackID)
		return c.Status(fiber.StatusNotFound).SendString("Track not found")
	}

	// Fetch all artists and albums for dropdowns
	artists, err := h.service.GetArtists(c.Context())
	if err != nil {
		slog.Error("Failed to get artists for dropdown", "error", err)
		artists = []*music.Artist{} // Continue with empty list
	}

	albums, err := h.service.GetAlbums(c.Context())
	if err != nil {
		slog.Error("Failed to get albums for dropdown", "error", err)
		albums = []*music.Album{} // Continue with empty list
	}

	// Ensure track's artists are included in the dropdown, even if missing from main query
	artistMap := make(map[string]bool)
	for _, artist := range artists {
		artistMap[artist.ID] = true
	}
	// Add track artists (include those without IDs for fetched data)
	for _, artistRole := range track.Artists {
		if artistRole.Artist != nil {
			artistID := artistRole.Artist.ID
			if artistID == "" {
				// Generate a temporary ID for artists without database IDs (for dropdown display)
				artistID = "temp_" + artistRole.Artist.Name
				artistRole.Artist.ID = artistID
			}
			if !artistMap[artistID] {
				artists = append(artists, artistRole.Artist)
				artistMap[artistID] = true
			}
		}
	}
	// Add album artists (include those without IDs for fetched data)
	if track.Album != nil {
		for _, artistRole := range track.Album.Artists {
			if artistRole.Artist != nil {
				artistID := artistRole.Artist.ID
				if artistID == "" {
					// Generate a temporary ID for artists without database IDs (for dropdown display)
					artistID = "temp_" + artistRole.Artist.Name
					artistRole.Artist.ID = artistID
				}
				if !artistMap[artistID] {
					artists = append(artists, artistRole.Artist)
					artistMap[artistID] = true
				}
			}
		}
	}

	// Ensure track has valid ID for template
	if track.ID == "" {
		track.ID = trackID
	}

	// Determine selected album artist ID for template
	selectedAlbumArtistID := ""
	if track.Album != nil && len(track.Album.Artists) > 0 {
		selectedAlbumArtistID = track.Album.Artists[0].Artist.ID
	}

	// Create map of selected artist IDs for template
	selectedArtistIDs := make(map[string]bool)
	for _, artistRole := range track.Artists {
		if artistRole.Artist != nil && artistRole.Artist.ID != "" {
			selectedArtistIDs[artistRole.Artist.ID] = true
		}
	}

	// Check if request is HTMX or full page
	if c.Get("HX-Request") == "true" {
		// Return the full tag section with button loading HTMX for HTMX requests
		return c.Render("sections/tag", fiber.Map{
			"Track":                 track,
			"Artists":               artists,
			"Albums":                albums,
			"SelectedAlbumArtistID": selectedAlbumArtistID,
			"SelectedArtistIDs":     selectedArtistIDs,
		})
	}

	// Return full page for direct navigation
	return c.Render("main", fiber.Map{
		"Track":                 track,
		"IsTagEdit":             true,
		"Artists":               artists,
		"Albums":                albums,
		"SelectedAlbumArtistID": selectedAlbumArtistID,
		"SelectedArtistIDs":     selectedArtistIDs,
	})
}
