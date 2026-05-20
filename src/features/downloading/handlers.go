package downloading

import (
	"log/slog"
	"strconv"
	"strings"

	"github.com/contre95/soulsolid/src/features/hosting/respond"
	"github.com/contre95/soulsolid/src/music"
	"github.com/gofiber/fiber/v2"
)

// Handler handles HTTP requests for downloading
type Handler struct {
	service *Service
}

// NewHandler creates a new downloading handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RenderDownloadSection renders the download page.
func (h *Handler) RenderDownloadSection(c *fiber.Ctx) error {
	slog.Debug("RenderDownloadSection handler called")
	downloader := c.Query("downloader", "")
	if downloader == "" {
		cfg := h.service.configManager.Get()
		if len(cfg.Downloaders.Plugins) > 0 {
			downloader = cfg.Downloaders.Plugins[0].Name
		}
	}
	return respond.Section(c, "download", fiber.Map{
		"Title":             "Download",
		"CurrentDownloader": downloader,
	})
}

// SearchRequest represents a search request
type SearchRequest struct {
	Query      string `json:"query" form:"query"`
	Type       string `json:"type" form:"type"`
	Limit      int    `json:"limit" form:"limit"`
	Downloader string `json:"downloader" form:"downloader"`
}

// SearchAlbums handles album search requests
func (h *Handler) SearchAlbums(c *fiber.Ctx) error {
	slog.Debug("SearchAlbums handler called")

	var req SearchRequest
	if err := c.BodyParser(&req); err != nil {
		return respond.Err(c, fiber.StatusBadRequest, "Invalid request body")
	}
	if req.Query == "" {
		return respond.Err(c, fiber.StatusBadRequest, "Query parameter is required")
	}

	albums, err := h.service.SearchAlbums(req.Downloader, req.Query, req.Limit)
	if err != nil {
		slog.Error("Failed to search albums", "error", err)
		return respond.Err(c, fiber.StatusInternalServerError, "Failed to search albums")
	}

	if c.Get("HX-Request") == "true" {
		return h.renderAlbumResults(c, albums, req.Downloader)
	}
	return c.JSON(fiber.Map{"albums": albums})
}

// SearchTracks handles track search requests
func (h *Handler) SearchTracks(c *fiber.Ctx) error {
	slog.Debug("SearchTracks handler called")

	var req SearchRequest
	if err := c.BodyParser(&req); err != nil {
		return respond.Err(c, fiber.StatusBadRequest, "Invalid request body")
	}
	if req.Query == "" {
		return respond.Err(c, fiber.StatusBadRequest, "Query parameter is required")
	}

	tracks, err := h.service.SearchTracks(req.Downloader, req.Query, req.Limit)
	if err != nil {
		slog.Error("Failed to search tracks", "error", err)
		return respond.Err(c, fiber.StatusInternalServerError, "Failed to search tracks")
	}

	if c.Get("HX-Request") == "true" {
		return h.renderTrackResults(c, tracks, req.Downloader)
	}
	return c.JSON(fiber.Map{"tracks": tracks})
}

// Search handles general search requests
func (h *Handler) Search(c *fiber.Ctx) error {
	slog.Debug("Search handler called")

	var req SearchRequest
	if err := c.BodyParser(&req); err != nil {
		return respond.Err(c, fiber.StatusBadRequest, "Invalid request body")
	}
	if req.Query == "" {
		return respond.Err(c, fiber.StatusBadRequest, "Query parameter is required")
	}
	if req.Limit == 0 {
		req.Limit = 20
	}

	htmx := c.Get("HX-Request") == "true"

	switch req.Type {
	case "album":
		albums, err := h.service.SearchAlbums(req.Downloader, req.Query, req.Limit)
		if err != nil {
			slog.Error("Failed to search albums", "error", err)
			return respond.Err(c, fiber.StatusInternalServerError, "Failed to search albums")
		}
		if htmx {
			return h.renderAlbumResults(c, albums, req.Downloader)
		}
		return c.JSON(fiber.Map{"albums": albums})

	case "track":
		tracks, err := h.service.SearchTracks(req.Downloader, req.Query, req.Limit)
		if err != nil {
			slog.Error("Failed to search tracks", "error", err)
			return respond.Err(c, fiber.StatusInternalServerError, "Failed to search tracks")
		}
		if htmx {
			return h.renderTrackResults(c, tracks, req.Downloader)
		}
		return c.JSON(fiber.Map{"tracks": tracks})

	case "artist":
		artists, err := h.service.SearchArtists(req.Downloader, req.Query, req.Limit)
		if err != nil {
			slog.Error("Failed to search artists", "error", err)
			return respond.Err(c, fiber.StatusInternalServerError, "Failed to search artists")
		}
		if htmx {
			return h.renderArtistResults(c, artists, req.Downloader)
		}
		return c.JSON(fiber.Map{"artists": artists})

	case "link":
		result, err := h.service.SearchLinks(req.Downloader, req.Query, req.Limit)
		if err != nil {
			slog.Error("Failed to search links", "error", err)
			return respond.Err(c, fiber.StatusInternalServerError, "Failed to search links")
		}
		if htmx {
			switch result.Type {
			case "artist":
				return h.renderArtistLinkResults(c, result.Artist, result.Albums, req.Downloader)
			default:
				return h.renderLinkResults(c, result.Tracks, req.Downloader)
			}
		}
		return c.JSON(result)

	default:
		return respond.Err(c, fiber.StatusBadRequest, "Invalid search type")
	}
}

// renderAlbumResults renders album search results as HTML for HTMX
func (h *Handler) renderAlbumResults(c *fiber.Ctx, albums []music.Album, downloader string) error {
	return c.Render("downloading/album_results", fiber.Map{
		"Albums":     albums,
		"Downloader": downloader,
	})
}

// renderTrackResults renders track search results as HTML for HTMX
func (h *Handler) renderTrackResults(c *fiber.Ctx, tracks []music.Track, downloader string) error {
	trackPtrs := make([]*music.Track, len(tracks))
	for i := range tracks {
		trackPtrs[i] = &tracks[i]
	}
	return c.Render("downloading/spotify_track_results", fiber.Map{
		"Tracks":     trackPtrs,
		"Downloader": downloader,
	})
}

// renderLinkResults renders link search results as HTML for HTMX
func (h *Handler) renderLinkResults(c *fiber.Ctx, tracks []music.Track, downloader string) error {
	playlistName := ""
	if len(tracks) > 0 && tracks[0].Attributes != nil {
		playlistName = tracks[0].Attributes["playlist_name"]
	}
	trackPtrs := make([]*music.Track, len(tracks))
	for i := range tracks {
		trackPtrs[i] = &tracks[i]
	}
	return c.Render("downloading/link_results", fiber.Map{
		"Tracks":       trackPtrs,
		"Downloader":   downloader,
		"PlaylistName": playlistName,
	})
}

// renderArtistLinkResults renders artist link results (artist info + albums) as HTML for HTMX
func (h *Handler) renderArtistLinkResults(c *fiber.Ctx, artist *music.Artist, albums []music.Album, downloader string) error {
	return c.Render("downloading/artist_link_results", fiber.Map{
		"Artist":     artist,
		"Albums":     albums,
		"Downloader": downloader,
	})
}

// renderArtistResults renders artist search results as HTML for HTMX
func (h *Handler) renderArtistResults(c *fiber.Ctx, artists []music.Artist, downloader string) error {
	return c.Render("downloading/artist_results", fiber.Map{
		"Artists":    artists,
		"Downloader": downloader,
	})
}

// DownloadTrackRequest represents a download track request
type DownloadTrackRequest struct {
	TrackID string `json:"trackId" form:"trackId"`
}

// DownloadAlbumRequest represents a download album request
type DownloadAlbumRequest struct {
	AlbumID string `json:"albumId" form:"albumId"`
}

// DownloadArtistRequest represents a download artist request
type DownloadArtistRequest struct {
	ArtistID string `json:"artistId" form:"artistId"`
}

// DownloadTracksRequest represents a download tracks request
type DownloadTracksRequest struct {
	TrackIDs string `json:"trackIds" form:"trackIds"` // Comma-separated track IDs
}

// DownloadTrack handles track download requests
func (h *Handler) DownloadTrack(c *fiber.Ctx) error {
	slog.Debug("DownloadTrack handler called")

	var req DownloadTrackRequest
	if err := c.BodyParser(&req); err != nil {
		return respond.Err(c, fiber.StatusBadRequest, "Invalid request body")
	}
	if req.TrackID == "" {
		return respond.Err(c, fiber.StatusBadRequest, "Track ID is required")
	}

	downloader := strings.Clone(c.Query("downloader", "dummy"))
	slog.Info("DownloadTrack", "downloader", downloader, "trackID", req.TrackID)

	jobID, err := h.service.DownloadTrack(downloader, req.TrackID)
	if err != nil {
		slog.Error("Failed to start track download", "error", err)
		return respond.Err(c, fiber.StatusInternalServerError, "Failed to start track download")
	}

	if c.Get("HX-Request") == "true" {
		return c.Render("toast/toastOk", fiber.Map{"Msg": "Track download started"})
	}
	return c.JSON(fiber.Map{"jobId": jobID, "message": "Download started"})
}

// DownloadAlbum handles album download requests
func (h *Handler) DownloadAlbum(c *fiber.Ctx) error {
	slog.Debug("DownloadAlbum handler called")

	var req DownloadAlbumRequest
	if err := c.BodyParser(&req); err != nil {
		return respond.Err(c, fiber.StatusBadRequest, "Invalid request body")
	}
	if req.AlbumID == "" {
		return respond.Err(c, fiber.StatusBadRequest, "Album ID is required")
	}

	downloader := strings.Clone(c.Query("downloader", "dummy"))
	jobID, err := h.service.DownloadAlbum(downloader, req.AlbumID)
	if err != nil {
		slog.Error("Failed to start album download", "error", err)
		return respond.Err(c, fiber.StatusInternalServerError, "Failed to start album download")
	}

	if c.Get("HX-Request") == "true" {
		return c.Render("toast/toastOk", fiber.Map{"Msg": "Album download started"})
	}
	return c.JSON(fiber.Map{"jobId": jobID, "message": "Download started"})
}

// DownloadArtist handles artist download requests
func (h *Handler) DownloadArtist(c *fiber.Ctx) error {
	slog.Debug("DownloadArtist handler called")

	var req DownloadArtistRequest
	if err := c.BodyParser(&req); err != nil {
		return respond.Err(c, fiber.StatusBadRequest, "Invalid request body")
	}
	if req.ArtistID == "" {
		return respond.Err(c, fiber.StatusBadRequest, "Artist ID is required")
	}

	downloader := strings.Clone(c.Query("downloader", "dummy"))
	jobID, err := h.service.DownloadArtist(downloader, req.ArtistID)
	if err != nil {
		slog.Error("Failed to start artist download", "error", err)
		return respond.Err(c, fiber.StatusInternalServerError, "Failed to start artist download")
	}

	if c.Get("HX-Request") == "true" {
		return c.Render("toast/toastOk", fiber.Map{"Msg": "Artist download started"})
	}
	return c.JSON(fiber.Map{"jobId": jobID, "message": "Download started"})
}

// DownloadTracks handles multiple track download requests
func (h *Handler) DownloadTracks(c *fiber.Ctx) error {
	slog.Debug("DownloadTracks handler called")

	var req DownloadTracksRequest
	if err := c.BodyParser(&req); err != nil {
		return respond.Err(c, fiber.StatusBadRequest, "Invalid request body")
	}
	if req.TrackIDs == "" {
		return respond.Err(c, fiber.StatusBadRequest, "Track IDs are required")
	}

	trackIDs := strings.Split(req.TrackIDs, ",")
	for i, id := range trackIDs {
		trackIDs[i] = strings.TrimSpace(id)
	}

	downloader := strings.Clone(c.Query("downloader", "dummy"))
	jobID, err := h.service.DownloadTracks(downloader, trackIDs)
	if err != nil {
		slog.Error("Failed to start tracks download", "error", err)
		return respond.Err(c, fiber.StatusInternalServerError, "Failed to start tracks download")
	}

	if c.Get("HX-Request") == "true" {
		return c.Render("toast/toastOk", fiber.Map{"Msg": "Tracks download started"})
	}
	return c.JSON(fiber.Map{"jobId": jobID, "message": "Download started"})
}

// DownloadPlaylist handles playlist download requests
func (h *Handler) DownloadPlaylist(c *fiber.Ctx) error {
	slog.Debug("DownloadPlaylist handler called")

	var req struct {
		TrackIDs     string `json:"trackIds" form:"trackIds"`
		PlaylistName string `json:"playlistName" form:"playlistName"`
	}
	if err := c.BodyParser(&req); err != nil {
		return respond.Err(c, fiber.StatusBadRequest, "Invalid request body")
	}
	if req.TrackIDs == "" {
		return respond.Err(c, fiber.StatusBadRequest, "Track IDs are required")
	}
	if req.PlaylistName == "" {
		return respond.Err(c, fiber.StatusBadRequest, "Playlist name is required")
	}

	trackIDs := strings.Split(req.TrackIDs, ",")
	for i, id := range trackIDs {
		trackIDs[i] = strings.TrimSpace(id)
	}

	downloader := strings.Clone(c.Query("downloader", "dummy"))
	jobID, err := h.service.DownloadPlaylist(downloader, trackIDs, req.PlaylistName)
	if err != nil {
		slog.Error("Failed to start playlist download", "error", err)
		return respond.Err(c, fiber.StatusInternalServerError, "Failed to start playlist download")
	}

	if c.Get("HX-Request") == "true" {
		return c.Render("toast/toastOk", fiber.Map{"Msg": "Playlist '" + req.PlaylistName + "' download started"})
	}
	return c.JSON(fiber.Map{"jobId": jobID, "message": "Download started"})
}

// GetAlbumTracks handles requests to get tracks from an album
func (h *Handler) GetAlbumTracks(c *fiber.Ctx) error {
	slog.Debug("GetAlbumTracks handler called")

	albumID := c.Params("albumId")
	if albumID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Album ID is required"})
	}

	downloader := strings.Clone(c.Query("downloader", "dummy"))
	tracks, err := h.service.GetAlbumTracks(downloader, albumID)
	if err != nil {
		slog.Error("Failed to get album tracks", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to get album tracks"})
	}

	album := &music.Album{ID: albumID, Title: "Album"}
	if len(tracks) > 0 && tracks[0].Album != nil {
		album = tracks[0].Album
	}

	var totalDuration int
	for _, track := range tracks {
		totalDuration += track.Metadata.Duration
	}

	trackPtrs := make([]*music.Track, len(tracks))
	for i := range tracks {
		trackPtrs[i] = &tracks[i]
	}
	return c.Render("downloading/album_tracks", fiber.Map{
		"Album":         album,
		"Tracks":        trackPtrs,
		"TotalDuration": totalDuration,
		"Downloader":    downloader,
	})
}

// GetChartTracks handles chart track requests
func (h *Handler) GetChartTracks(c *fiber.Ctx) error {
	limit := 20
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}

	downloader := strings.Clone(c.Query("downloader", "dummy"))

	var caps DownloaderCapabilities
	if d, exists := h.service.pluginManager.GetDownloader(downloader); exists {
		caps = d.Capabilities()
	}

	if !caps.SupportsChartTracks {
		downloaderName := downloader
		if d, exists := h.service.pluginManager.GetDownloader(downloader); exists {
			downloaderName = d.Name()
		}
		return c.Render("downloading/chart_tracks", fiber.Map{
			"Tracks":         []*music.Track{},
			"NotSupported":   true,
			"DownloaderName": downloaderName,
			"Downloader":     downloader,
		})
	}

	tracks, err := h.service.GetChartTracks(downloader, limit)
	statuses := h.service.GetDownloaderStatuses()

	downloaderName := downloader
	if d, exists := h.service.pluginManager.GetDownloader(downloader); exists {
		downloaderName = d.Name()
	}
	downloaderKey := strings.ToLower(downloaderName)
	downloaderStatus := statuses[downloaderKey]

	if err != nil || downloaderStatus.Status != "valid" {
		return c.Render("downloading/chart_tracks", fiber.Map{
			"Tracks":           []*music.Track{},
			"DownloaderStatus": downloaderStatus,
			"DownloaderName":   downloaderName,
			"Downloader":       downloader,
		})
	}

	trackPtrs := make([]*music.Track, len(tracks))
	for i := range tracks {
		trackPtrs[i] = &tracks[i]
	}
	return c.Render("downloading/chart_tracks", fiber.Map{
		"Tracks":           trackPtrs,
		"DownloaderStatus": downloaderStatus,
		"DownloaderName":   downloaderName,
		"Downloader":       downloader,
	})
}

// GetUserInfo handles requests for user information
func (h *Handler) GetUserInfo(c *fiber.Ctx) error {
	downloader := strings.Clone(c.Query("downloader", "dummy"))
	userInfo := h.service.GetUserInfo(downloader)
	statuses := h.service.GetDownloaderStatuses()

	downloaderName := downloader
	if d, exists := h.service.pluginManager.GetDownloader(downloader); exists {
		downloaderName = d.Name()
	}
	downloaderKey := strings.ToLower(downloaderName)
	downloaderStatus := statuses[downloaderKey]
	hasDownloaders := len(h.service.pluginManager.GetAllDownloaders()) > 0

	var caps DownloaderCapabilities
	if d, exists := h.service.pluginManager.GetDownloader(downloader); exists {
		caps = d.Capabilities()
	}

	if c.Get("HX-Request") == "true" {
		return c.Render("downloading/user_info", fiber.Map{
			"UserInfo":          userInfo,
			"Statuses":          statuses,
			"DownloaderName":    downloaderName,
			"DownloaderStatus":  downloaderStatus,
			"HasDownloaders":    hasDownloaders,
			"CurrentDownloader": downloader,
			"Capabilities":      caps,
		})
	}
	return c.JSON(fiber.Map{
		"userInfo": userInfo,
		"statuses": statuses,
	})
}

// GetDownloaderCapabilities handles requests for downloader capabilities
func (h *Handler) GetDownloaderCapabilities(c *fiber.Ctx) error {
	downloader := strings.Clone(c.Query("downloader", "dummy"))
	caps, err := h.service.GetDownloaderCapabilities(downloader)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(caps)
}
