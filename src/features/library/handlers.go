package library

import (
	"fmt"
	"log/slog"
	"math"
	"regexp"
	"strings"

	"github.com/contre95/soulsolid/src/hosting/respond"
	"github.com/contre95/soulsolid/src/music"
	"github.com/gofiber/fiber/v3"
)

var lrcTimestamp = regexp.MustCompile(`^\[\d+:\d+\.\d+\]\s*`)

func stripLRC(lyrics string) string {
	var lines []string
	for _, line := range strings.Split(lyrics, "\n") {
		line = lrcTimestamp.ReplaceAllString(strings.TrimSpace(line), "")
		if line != "" {
			lines = append(lines, line)
		}
	}
	return strings.Join(lines, "\n")
}

// Handler is the handler for the library feature.
type Handler struct {
	service *Service
}

// NewHandler creates a new handler for the library feature.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RenderLibrarySection renders the library page.
func (h *Handler) RenderLibrarySection(c fiber.Ctx) error {
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

	return respond.Section(c, "library", fiber.Map{
		"Title":               "Library",
		"DefaultDownloadPath": h.service.configManager.Get().DownloadPath,
		"SearchArtists":       artists,
		"SearchAlbums":        albums,
	})
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
func (h *Handler) GetArtist(c fiber.Ctx) error {
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
func (h *Handler) GetAlbum(c fiber.Ctx) error {
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
func (h *Handler) GetTrack(c fiber.Ctx) error {
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
func (h *Handler) GetArtistsCount(c fiber.Ctx) error {
	slog.Debug("GetArtistsCount handler called")
	artists, err := h.service.GetArtists(c.Context())
	if err != nil {
		slog.Error("Error loading artists count", "error", err)
		return respond.ToastErr(c, fiber.StatusInternalServerError, "Error loading artists count")
	}
	return respond.Text(c, "artists_count", len(artists))
}

// GetAlbumsCount returns the count of albums in the library.
func (h *Handler) GetAlbumsCount(c fiber.Ctx) error {
	slog.Debug("GetAlbumsCount handler called")
	albums, err := h.service.GetAlbums(c.Context())
	if err != nil {
		slog.Error("Error loading albums count", "error", err)
		return respond.ToastErr(c, fiber.StatusInternalServerError, "Error loading albums count")
	}
	return respond.Text(c, "albums_count", len(albums))
}

// GetTracksCount returns the count of tracks in the library.
func (h *Handler) GetTracksCount(c fiber.Ctx) error {
	slog.Debug("GetTracksCount handler called")
	count, err := h.service.GetTracksCount(c.Context())
	if err != nil {
		slog.Error("Error loading tracks count", "error", err)
		return respond.ToastErr(c, fiber.StatusInternalServerError, "Error loading tracks count")
	}
	return respond.Text(c, "tracks_count", count, fmt.Sprintf("%d tracks", count))
}

// GetStorageSize returns the storage size of the library.
func (h *Handler) GetStorageSize(c fiber.Ctx) error {
	slog.Debug("GetStorageSize handler called")
	size, err := h.service.GetStorageSize(c.Context())
	if err != nil {
		slog.Error("Error loading storage size", "error", err)
		return respond.ToastErr(c, fiber.StatusInternalServerError, "Error loading storage size")
	}

	var formatted string
	if size >= 1_000_000_000_000 {
		formatted = fmt.Sprintf("%.1f TB", float64(size)/math.Pow(10, 12))
	} else if size >= 1_000_000_000 {
		formatted = fmt.Sprintf("%.1f GB", float64(size)/math.Pow(10, 9))
	} else if size >= 1_000_000 {
		formatted = fmt.Sprintf("%.1f MB", float64(size)/math.Pow(10, 6))
	} else if size >= 1_000 {
		formatted = fmt.Sprintf("%.1f KB", float64(size)/math.Pow(10, 3))
	} else {
		formatted = fmt.Sprintf("%d B", size)
	}
	return respond.Text(c, "storage_size_bytes", size, formatted)
}

// GetLibraryTable renders the library table section with tabs.
func (h *Handler) GetLibraryTable(c fiber.Ctx) error {
	slog.Debug("GetLibraryTable handler called")

	// Fetch all artists and albums for search form
	artists, err := h.service.GetArtists(c.Context())
	if err != nil {
		slog.Error("Error loading artists for search form", "error", err)
		artists = []*music.Artist{}
	}

	albums, err := h.service.GetAlbums(c.Context())
	if err != nil {
		slog.Error("Error loading albums for search form", "error", err)
		albums = []*music.Album{}
	}

	genres, err := h.service.GetGenres(c.Context())
	if err != nil {
		slog.Error("Error loading genres for search form", "error", err)
		genres = []string{}
	}

	return respond.Partial(c, "library/library_table", fiber.Map{
		"SearchArtists": artists,
		"SearchAlbums":  albums,
		"Genres":        genres,
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
	Path        string // File path (tracks only) — used to stream via /stream?path=
}

// parseBoolFilter converts "true"/"false" query params to *bool; anything else returns nil.
func parseBoolFilter(s string) *bool {
	if s == "true" {
		v := true
		return &v
	}
	if s == "false" {
		v := false
		return &v
	}
	return nil
}

// trackToSearchResult converts a music.Track to a SearchResult.
func trackToSearchResult(track *music.Track) SearchResult {
	var artistNames strings.Builder
	for _, ar := range track.Artists {
		if ar.Artist == nil {
			continue
		}
		if artistNames.Len() > 0 {
			artistNames.WriteString(", ")
		}
		artistNames.WriteString(ar.Artist.Name)
	}
	albumTitle := ""
	if track.Album != nil {
		albumTitle = track.Album.Title
	}
	return SearchResult{
		Type:        "track",
		ID:          track.ID,
		PrimaryName: track.Title,
		Secondary:   artistNames.String(),
		Tertiary:    albumTitle,
		Duration:    track.Metadata.Duration,
		Path:        track.Path,
	}
}

// albumToSearchResult converts a music.Album to a SearchResult.
func albumToSearchResult(album *music.Album) SearchResult {
	var artistNames strings.Builder
	for _, ar := range album.Artists {
		if ar.Artist == nil {
			continue
		}
		if artistNames.Len() > 0 {
			artistNames.WriteString(", ")
		}
		artistNames.WriteString(ar.Artist.Name)
	}
	year := ""
	if !album.ReleaseDate.IsZero() {
		year = fmt.Sprintf("%d", album.ReleaseDate.Year())
	}
	return SearchResult{
		Type:        "album",
		ID:          album.ID,
		PrimaryName: album.Title,
		Secondary:   artistNames.String(),
		Tertiary:    year,
	}
}

// artistToSearchResult converts a music.Artist to a SearchResult.
func artistToSearchResult(artist *music.Artist) SearchResult {
	return SearchResult{
		Type:        "artist",
		ID:          artist.ID,
		PrimaryName: artist.Name,
		Secondary:   artist.ID,
	}
}

// GetUnifiedSearch performs a unified search across artists, albums, and tracks.
func (h *Handler) GetUnifiedSearch(c fiber.Ctx) error {
	slog.Debug("GetUnifiedSearch handler called")

	query := strings.TrimSpace(c.Query("query", ""))
	page := fiber.Query[int](c, "page", 1)
	limit := fiber.Query[int](c, "limit", 50)
	genre := c.Query("genre", "")
	hasAcoustID := parseBoolFilter(c.Query("has_acoustid", ""))
	lyricsFilter := c.Query("lyrics_filter", "")
	lyricsText := strings.TrimSpace(c.Query("lyrics_text", ""))
	addedAfter := strings.TrimSpace(c.Query("added_after", ""))
	addedBefore := strings.TrimSpace(c.Query("added_before", ""))

	if c.Query("view") == "albums" {
		return h.renderAlbumGrid(c, query, genre)
	}

	var results []SearchResult
	var totalCount int

	offset := (page - 1) * limit

	hasActiveFilters := genre != "" || hasAcoustID != nil || lyricsFilter != "" || lyricsText != "" || addedAfter != "" || addedBefore != ""

	if query == "" && !hasActiveFilters {
		// Browse-all: paginated tracks only.
		tracksCount, err := h.service.GetTracksCount(c.Context())
		if err != nil {
			slog.Error("Error getting tracks count", "error", err)
			return c.Status(fiber.StatusInternalServerError).SendString("Error loading tracks count")
		}
		totalCount = tracksCount

		if offset < tracksCount {
			trackLimit := min(offset+limit, tracksCount) - offset
			if trackLimit > 0 {
				tracks, err := h.service.GetTracksPaginated(c.Context(), trackLimit, offset)
				if err != nil {
					slog.Error("Error loading tracks", "error", err)
					return c.Status(fiber.StatusInternalServerError).SendString("Error loading tracks")
				}
				for _, track := range tracks {
					results = append(results, trackToSearchResult(track))
				}
			}
		}
	} else {
		// Search/filter: albums → artists → tracks order.
		// Artist/album matches only apply to a text query and are capped (not paginated).
		trackFilter := &music.TrackFilter{
			TextSearch:   query,
			Genre:        genre,
			HasAcoustID:  hasAcoustID,
			LyricsFilter: lyricsFilter,
			LyricsText:   lyricsText,
			AddedAfter:   addedAfter,
			AddedBefore:  addedBefore,
		}
		trackCount, err := h.service.GetTracksFilteredCount(c.Context(), trackFilter)
		if err != nil {
			slog.Error("Error counting tracks", "error", err)
		}

		var albums []*music.Album
		var artists []*music.Artist
		if query != "" {
			albums, err = h.service.SearchAlbums(c.Context(), query, 20, 0)
			if err != nil {
				slog.Error("Error searching albums", "error", err)
				albums = nil
			}
			artists, err = h.service.GetArtistsFilteredPaginated(c.Context(), 20, 0, query)
			if err != nil {
				slog.Error("Error searching artists", "error", err)
				artists = nil
			}
		}
		albumsCount := len(albums)
		artistsCount := len(artists)
		totalCount = albumsCount + artistsCount + trackCount

		start := offset
		end := min(offset+limit, totalCount)

		// Albums: [0, albumsCount)
		if start < albumsCount {
			albumEnd := min(end, albumsCount)
			for i := start; i < albumEnd; i++ {
				results = append(results, albumToSearchResult(albums[i]))
			}
		}

		// Artists: [albumsCount, albumsCount+artistsCount)
		if end > albumsCount {
			artistStart := max(0, start-albumsCount)
			artistEnd := min(end-albumsCount, artistsCount)
			for i := artistStart; i < artistEnd; i++ {
				results = append(results, artistToSearchResult(artists[i]))
			}
		}

		// Tracks: [albumsCount+artistsCount, totalCount)
		trackOffset := albumsCount + artistsCount
		if end > trackOffset {
			trackStart := max(0, start-trackOffset)
			trackLimit := (end - trackOffset) - trackStart
			if trackLimit > 0 {
				tracks, err := h.service.GetTracksFilteredPaginated(c.Context(), trackLimit, trackStart, trackFilter)
				if err != nil {
					slog.Error("Error searching tracks", "error", err)
				} else {
					for _, track := range tracks {
						results = append(results, trackToSearchResult(track))
					}
				}
			}
		}
	}

	pagination := NewPagination(page, limit, totalCount)

	return respond.Partial(c, "library/unified_search_list", fiber.Map{
		"Results":    results,
		"Pagination": pagination,
		"Query":      query,
	})
}

// albumGridPageSize is divisible by every column count the album grid uses (2–6),
// so full pages never end with a partial row.
const albumGridPageSize = 48

// renderAlbumGrid renders the album browser: a paginated cover grid filtered by the
// library search form. Track-only filters (lyrics, AcoustID, dates) don't apply here.
func (h *Handler) renderAlbumGrid(c fiber.Ctx, query, genre string) error {
	page := max(fiber.Query[int](c, "page", 1), 1)
	limit := fiber.Query[int](c, "limit", albumGridPageSize)
	if limit <= 0 {
		limit = albumGridPageSize
	}
	filter := &music.AlbumFilter{
		TextSearch: query,
		Genre:      genre,
		Sort:       music.AlbumSort(c.Query("album_sort", string(music.AlbumSortArtist))),
	}

	count, err := h.service.GetAlbumSummariesCount(c.Context(), filter)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Error loading albums")
	}
	// Clamp a stale page (e.g. albums removed since the link was built) to the last one.
	if lastPage := max((count+limit-1)/limit, 1); page > lastPage {
		page = lastPage
	}
	albums, err :=h.service.GetAlbumSummaries(c.Context(), limit, (page-1)*limit, filter)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Error loading albums")
	}

	return respond.Partial(c, "library/album_grid", fiber.Map{
		"Albums":     albums,
		"Pagination": NewPagination(page, limit, count),
		"Query":      query,
		"Filtered":   query != "" || genre != "",
	})
}

// AlbumTrackRow is one track line in the album view.
type AlbumTrackRow struct {
	ID          string
	Number      int
	Title       string
	Artists     string
	ShowArtists bool // only when the track artists differ from the album artists
	Duration    int
	Path        string
}

// AlbumDisc groups an album's tracks by disc number.
type AlbumDisc struct {
	Number int
	Tracks []AlbumTrackRow
}

// joinArtistNames returns the non-nil artist names of the roles, comma separated.
func joinArtistNames(roles []music.ArtistRole) string {
	names := make([]string, 0, len(roles))
	for _, ar := range roles {
		if ar.Artist != nil {
			names = append(names, ar.Artist.Name)
		}
	}
	return strings.Join(names, ", ")
}

// RenderAlbumView renders an album's header and tracklist in place of the album grid.
func (h *Handler) RenderAlbumView(c fiber.Ctx) error {
	albumID := c.Params("id")
	album, err := h.service.GetAlbum(c.Context(), albumID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Error loading album")
	}
	if album == nil {
		return c.Status(fiber.StatusNotFound).SendString("Album not found")
	}
	tracks, err := h.service.GetAlbumTracks(c.Context(), albumID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Error loading album tracks")
	}

	albumArtists := joinArtistNames(album.Artists)
	// Same rule as the album grid: release date year, else the newest track year.
	// Years 0 and 1 are the placeholders written for unknown release dates.
	releaseYear := album.ReleaseDate.Year()
	year := 0
	totalDuration := 0
	var discs []AlbumDisc
	for _, t := range tracks {
		totalDuration += t.Metadata.Duration
		year = max(year, t.Metadata.Year)
		// Tracks come ordered by disc, so a new disc number starts a new group.
		if len(discs) == 0 || discs[len(discs)-1].Number != t.Metadata.DiscNumber {
			discs = append(discs, AlbumDisc{Number: t.Metadata.DiscNumber})
		}
		title := t.Title
		if t.TitleVersion != "" {
			title += " (" + t.TitleVersion + ")"
		}
		artists := joinArtistNames(t.Artists)
		d := &discs[len(discs)-1]
		d.Tracks = append(d.Tracks, AlbumTrackRow{
			ID:          t.ID,
			Number:      t.Metadata.TrackNumber,
			Title:       title,
			Artists:     artists,
			ShowArtists: artists != "" && artists != albumArtists,
			Duration:    t.Metadata.Duration,
			Path:        t.Path,
		})
	}

	if releaseYear > 1 {
		year = releaseYear
	}

	coverTrackID := ""
	if len(tracks) > 0 {
		coverTrackID = tracks[0].ID
	}

	return respond.Partial(c, "library/album_view", fiber.Map{
		"Album":         album,
		"AlbumArtists":  albumArtists,
		"Year":          year,
		"TrackCount":    len(tracks),
		"TotalDuration": totalDuration,
		"CoverTrackID":  coverTrackID,
		"Discs":         discs,
		"MultiDisc":     len(discs) > 1,
		"BackPage":      max(fiber.Query[int](c, "page", 1), 1),
	})
}

// GetLibraryFileTree returns a tree structure of the library path.
func (h *Handler) GetLibraryFileTree(c fiber.Ctx) error {
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
		return respond.ToastErr(c, fiber.StatusInternalServerError, "Failed to get library file tree")
	}
	return respond.Text(c, "file_tree", tree)
}

// DeleteTrack deletes a track from the library.
func (h *Handler) DeleteTrack(c fiber.Ctx) error {
	slog.Debug("DeleteTrack handler called", "trackId", c.Params("trackId"))

	trackID := c.Params("trackId")
	if trackID == "" {
		return respond.ToastErr(c, fiber.StatusBadRequest, "Track ID is required")
	}
	if err := h.service.DeleteTrack(c.Context(), trackID); err != nil {
		slog.Error("Failed to delete track", "error", err, "trackId", trackID)
		return respond.ToastErr(c, fiber.StatusInternalServerError, "Failed to delete track")
	}
	return respond.ToastOk(c, "Track deleted successfully")
}

// DeleteAlbum deletes an album from the library.
func (h *Handler) DeleteAlbum(c fiber.Ctx) error {
	slog.Debug("DeleteAlbum handler called", "albumId", c.Params("albumId"))

	albumID := c.Params("albumId")
	if albumID == "" {
		return respond.ToastErr(c, fiber.StatusBadRequest, "Album ID is required")
	}
	if err := h.service.DeleteAlbum(c.Context(), albumID); err != nil {
		slog.Error("Failed to delete album", "error", err, "albumId", albumID)
		return respond.ToastErr(c, fiber.StatusInternalServerError, "Failed to delete album")
	}
	return respond.ToastOk(c, "Album deleted successfully")
}

// DeleteArtist deletes an artist from the library.
func (h *Handler) DeleteArtist(c fiber.Ctx) error {
	slog.Debug("DeleteArtist handler called", "artistId", c.Params("artistId"))

	artistID := c.Params("artistId")
	if artistID == "" {
		return respond.ToastErr(c, fiber.StatusBadRequest, "Artist ID is required")
	}
	if err := h.service.DeleteArtist(c.Context(), artistID); err != nil {
		slog.Error("Failed to delete artist", "error", err, "artistId", artistID)
		return respond.ToastErr(c, fiber.StatusInternalServerError, "Failed to delete artist")
	}
	return respond.ToastOk(c, "Artist deleted successfully")
}

// RenderTrackOverviewPanel renders the floating track overview panel.
func (h *Handler) RenderTrackOverviewPanel(c fiber.Ctx) error {
	slog.Debug("RenderTrackOverviewPanel handler called", "trackId", c.Params("trackId"))

	trackID := c.Params("trackId")
	if trackID == "" {
		return c.Status(fiber.StatusBadRequest).SendString("Track ID is required")
	}

	track, err := h.service.GetTrack(c.Context(), trackID)
	if err != nil || track == nil {
		slog.Error("Failed to get track for overview", "error", err, "trackId", trackID)
		return c.Status(fiber.StatusNotFound).SendString("Track not found")
	}

	var artistNames strings.Builder
	for i, ar := range track.Artists {
		if i > 0 {
			artistNames.WriteString(", ")
		}
		if ar.Artist != nil {
			artistNames.WriteString(ar.Artist.Name)
		}
	}

	lyricsPreview := ""
	if track.Metadata.Lyrics != "" {
		plain := stripLRC(track.Metadata.Lyrics)
		lines := strings.Split(plain, "\n")
		if len(lines) > 20 {
			lines = lines[:20]
		}
		lyricsPreview = strings.Join(lines, "\n")
	}

	duplicates, err := h.service.GetKeptDuplicates(c.Context(), trackID)
	if err != nil {
		slog.Error("Failed to get kept duplicates for overview", "error", err, "trackId", trackID)
	}

	return respond.Partial(c, "library/track_overview_panel", fiber.Map{
		"Track":         track,
		"Artists":       artistNames.String(),
		"LyricsPreview": lyricsPreview,
		"Duplicates":    duplicates,
	})
}
