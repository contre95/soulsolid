package downloading

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/contre95/soulsolid/src/music"
	"github.com/gofiber/fiber/v2"
)

// Handler handles HTTP requests for downloading
type Handler struct {
	service *Service
}

// NewHandler creates a new downloading handler
func NewHandler(service *Service) *Handler {
	return &Handler{
		service: service,
	}
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
		if c.Get("HX-Request") == "true" {
			return c.Render("toast/toastErr", fiber.Map{
				"Msg": "Invalid request body",
			})
		}
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.Query == "" {
		if c.Get("HX-Request") == "true" {
			return c.Render("toast/toastErr", fiber.Map{
				"Msg": "Query parameter is required",
			})
		}
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Query parameter is required",
		})
	}

	albums, err := h.service.SearchAlbums(req.Downloader, req.Query, req.Limit)
	if err != nil {
		slog.Error("Failed to search albums", "error", err)
		if c.Get("HX-Request") == "true" {
			return c.Render("toast/toastErr", fiber.Map{
				"Msg": "Failed to search albums",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to search albums",
		})
	}

	if c.Get("HX-Request") == "true" {
		return h.renderAlbumResults(c, albums, req.Downloader)
	}
	return c.JSON(fiber.Map{
		"albums": albums,
	})
}

// SearchTracks handles track search requests
func (h *Handler) SearchTracks(c *fiber.Ctx) error {
	slog.Debug("SearchTracks handler called")

	var req SearchRequest
	if err := c.BodyParser(&req); err != nil {
		if c.Get("HX-Request") == "true" {
			return c.Render("toast/toastErr", fiber.Map{
				"Msg": "Invalid request body",
			})
		}
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.Query == "" {
		if c.Get("HX-Request") == "true" {
			return c.Render("toast/toastErr", fiber.Map{
				"Msg": "Query parameter is required",
			})
		}
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Query parameter is required",
		})
	}

	tracks, err := h.service.SearchTracks(req.Downloader, req.Query, req.Limit)
	if err != nil {
		slog.Error("Failed to search tracks", "error", err)
		if c.Get("HX-Request") == "true" {
			return c.Render("toast/toastErr", fiber.Map{
				"Msg": "Failed to search tracks",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to search tracks",
		})
	}

	if c.Get("HX-Request") == "true" {
		return h.renderTrackResults(c, tracks, req.Downloader)
	}
	return c.JSON(fiber.Map{
		"tracks": tracks,
	})
}

// Search handles general search requests
func (h *Handler) Search(c *fiber.Ctx) error {
	slog.Debug("Search handler called")

	var req SearchRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.Query == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Query parameter is required",
		})
	}

	if req.Limit == 0 {
		req.Limit = 20
	}

	// Check if it's an HTMX request
	if c.Get("HX-Request") == "true" {
		// Return HTML for HTMX
		switch req.Type {

		case "album":
			albums, err := h.service.SearchAlbums(req.Downloader, req.Query, req.Limit)
			if err != nil {
				slog.Error("Failed to search albums", "error", err)
				return c.Render("toast/toastErr", fiber.Map{
					"Msg": "Failed to search albums",
				})
			}
			return h.renderAlbumResults(c, albums, req.Downloader)
		case "track":
			tracks, err := h.service.SearchTracks(req.Downloader, req.Query, req.Limit)
			if err != nil {
				slog.Error("Failed to search tracks", "error", err)
				return c.Render("toast/toastErr", fiber.Map{
					"Msg": "Failed to search tracks",
				})
			}
			return h.renderTrackResults(c, tracks, req.Downloader)
		case "artist":
			artists, err := h.service.SearchArtists(req.Downloader, req.Query, req.Limit)
			if err != nil {
				slog.Error("Failed to search artists", "error", err)
				return c.Render("toast/toastErr", fiber.Map{
					"Msg": "Failed to search artists",
				})
			}
			return h.renderArtistResults(c, artists, req.Downloader)
		case "link":
			result, err := h.service.SearchLinks(req.Downloader, req.Query, req.Limit)
			if err != nil {
				slog.Error("Failed to search links", "error", err)
				return c.Render("toast/toastErr", fiber.Map{
					"Msg": "Failed to search links",
				})
			}
			// Route to appropriate renderer based on result type
			switch result.Type {
			case "artist":
				return h.renderArtistLinkResults(c, result.Artist, result.Albums, req.Downloader)
			default:
				return h.renderLinkResults(c, result.Tracks, req.Downloader)
			}
		default:
			return c.Render("toast/toastErr", fiber.Map{
				"Msg": "Invalid search type",
			})
		}
	}

	// Return JSON for API
	switch req.Type {

	case "album":
		albums, err := h.service.SearchAlbums(req.Downloader, req.Query, req.Limit)
		if err != nil {
			slog.Error("Failed to search albums", "error", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to search albums",
			})
		}
		return c.JSON(fiber.Map{
			"albums": albums,
		})
	case "track":
		tracks, err := h.service.SearchTracks(req.Downloader, req.Query, req.Limit)
		if err != nil {
			slog.Error("Failed to search tracks", "error", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to search tracks",
			})
		}
		return c.JSON(fiber.Map{
			"tracks": tracks,
		})
	case "artist":
		artists, err := h.service.SearchArtists(req.Downloader, req.Query, req.Limit)
		if err != nil {
			slog.Error("Failed to search artists", "error", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to search artists",
			})
		}
		return c.JSON(fiber.Map{
			"artists": artists,
		})
	case "link":
		result, err := h.service.SearchLinks(req.Downloader, req.Query, req.Limit)
		if err != nil {
			slog.Error("Failed to search links", "error", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to search links",
			})
		}
		return c.JSON(result)
	default:
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid search type",
		})
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
	return c.Render("downloading/spotify_track_results", fiber.Map{
		"Tracks":     tracks,
		"Downloader": downloader,
	})
}

// renderLinkResults renders link search results as HTML for HTMX
func (h *Handler) renderLinkResults(c *fiber.Ctx, tracks []music.Track, downloader string) error {
	// Check if tracks belong to a playlist
	playlistName := ""
	if len(tracks) > 0 && tracks[0].Attributes != nil {
		playlistName = tracks[0].Attributes["playlist_name"]
	}

	return c.Render("downloading/link_results", fiber.Map{
		"Tracks":       tracks,
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

// DownloadTrack handles track download requests
func (h *Handler) DownloadTrack(c *fiber.Ctx) error {
	slog.Debug("DownloadTrack handler called")

	var req DownloadTrackRequest
	if err := c.BodyParser(&req); err != nil {
		if c.Get("HX-Request") == "true" {
			return c.Render("toast/toastErr", fiber.Map{
				"Msg": "Invalid request body",
			})
		}
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.TrackID == "" {
		if c.Get("HX-Request") == "true" {
			return c.Render("toast/toastErr", fiber.Map{
				"Msg": "Track ID is required",
			})
		}
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Track ID is required",
		})
	}

	downloader := strings.Clone(c.Query("downloader", "dummy"))
	slog.Info("DownloadTrack", "downloader", downloader, "trackID", req.TrackID)

	jobID, err := h.service.DownloadTrack(downloader, req.TrackID)
	if err != nil {
		slog.Error("Failed to start track download", "error", err)
		if c.Get("HX-Request") == "true" {
			return c.Render("toast/toastErr", fiber.Map{
				"Msg": "Failed to start track download",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to start download",
		})
	}

	if c.Get("HX-Request") == "true" {
		return c.Render("toast/toastOk", fiber.Map{
			"Msg": "Track download started",
		})
	}
	return c.JSON(fiber.Map{
		"jobId":   jobID,
		"message": "Download started",
	})
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

// DownloadAlbum handles album download requests
func (h *Handler) DownloadAlbum(c *fiber.Ctx) error {
	slog.Debug("DownloadAlbum handler called")

	var req DownloadAlbumRequest
	if err := c.BodyParser(&req); err != nil {
		if c.Get("HX-Request") == "true" {
			return c.Render("toast/toastErr", fiber.Map{
				"Msg": "Invalid request body",
			})
		}
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.AlbumID == "" {
		if c.Get("HX-Request") == "true" {
			return c.Render("toast/toastErr", fiber.Map{
				"Msg": "Album ID is required",
			})
		}
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Album ID is required",
		})
	}

	downloader := strings.Clone(c.Query("downloader", "dummy"))
	jobID, err := h.service.DownloadAlbum(downloader, req.AlbumID)
	if err != nil {
		slog.Error("Failed to start album download", "error", err)
		if c.Get("HX-Request") == "true" {
			return c.Render("toast/toastErr", fiber.Map{
				"Msg": "Failed to start album download",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to start download",
		})
	}

	if c.Get("HX-Request") == "true" {
		return c.Render("toast/toastOk", fiber.Map{
			"Msg": "Album download started",
		})
	}
	return c.JSON(fiber.Map{
		"jobId":   jobID,
		"message": "Download started",
	})
}

// DownloadArtist handles artist download requests
func (h *Handler) DownloadArtist(c *fiber.Ctx) error {
	slog.Debug("DownloadArtist handler called")

	var req DownloadArtistRequest
	if err := c.BodyParser(&req); err != nil {
		if c.Get("HX-Request") == "true" {
			return c.Render("toast/toastErr", fiber.Map{
				"Msg": "Invalid request body",
			})
		}
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.ArtistID == "" {
		if c.Get("HX-Request") == "true" {
			return c.Render("toast/toastErr", fiber.Map{
				"Msg": "Artist ID is required",
			})
		}
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Artist ID is required",
		})
	}

	downloader := strings.Clone(c.Query("downloader", "dummy"))
	jobID, err := h.service.DownloadArtist(downloader, req.ArtistID)
	if err != nil {
		slog.Error("Failed to start artist download", "error", err)
		if c.Get("HX-Request") == "true" {
			return c.Render("toast/toastErr", fiber.Map{
				"Msg": "Failed to start artist download",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to start download",
		})
	}

	if c.Get("HX-Request") == "true" {
		return c.Render("toast/toastOk", fiber.Map{
			"Msg": "Artist download started",
		})
	}
	return c.JSON(fiber.Map{
		"jobId":   jobID,
		"message": "Download started",
	})
}

// DownloadTracks handles multiple track download requests
func (h *Handler) DownloadTracks(c *fiber.Ctx) error {
	slog.Debug("DownloadTracks handler called")

	var req DownloadTracksRequest
	if err := c.BodyParser(&req); err != nil {
		if c.Get("HX-Request") == "true" {
			return c.Render("toast/toastErr", fiber.Map{
				"Msg": "Invalid request body",
			})
		}
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.TrackIDs == "" {
		if c.Get("HX-Request") == "true" {
			return c.Render("toast/toastErr", fiber.Map{
				"Msg": "Track IDs are required",
			})
		}
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Track IDs are required",
		})
	}

	// Split comma-separated track IDs
	trackIDs := strings.Split(req.TrackIDs, ",")
	for i, id := range trackIDs {
		trackIDs[i] = strings.TrimSpace(id)
	}

	downloader := strings.Clone(c.Query("downloader", "dummy"))
	jobID, err := h.service.DownloadTracks(downloader, trackIDs)
	if err != nil {
		slog.Error("Failed to start tracks download", "error", err)
		if c.Get("HX-Request") == "true" {
			return c.Render("toast/toastErr", fiber.Map{
				"Msg": "Failed to start tracks download",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to start download",
		})
	}

	if c.Get("HX-Request") == "true" {
		return c.Render("toast/toastOk", fiber.Map{
			"Msg": "Tracks download started",
		})
	}
	return c.JSON(fiber.Map{
		"jobId":   jobID,
		"message": "Download started",
	})
}

// DownloadPlaylist handles playlist download requests
func (h *Handler) DownloadPlaylist(c *fiber.Ctx) error {
	slog.Debug("DownloadPlaylist handler called")

	var req struct {
		TrackIDs     string `json:"trackIds" form:"trackIds"`
		PlaylistName string `json:"playlistName" form:"playlistName"`
	}
	if err := c.BodyParser(&req); err != nil {
		if c.Get("HX-Request") == "true" {
			return c.Render("toast/toastErr", fiber.Map{
				"Msg": "Invalid request body",
			})
		}
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.TrackIDs == "" {
		if c.Get("HX-Request") == "true" {
			return c.Render("toast/toastErr", fiber.Map{
				"Msg": "Track IDs are required",
			})
		}
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Track IDs are required",
		})
	}

	if req.PlaylistName == "" {
		if c.Get("HX-Request") == "true" {
			return c.Render("toast/toastErr", fiber.Map{
				"Msg": "Playlist name is required",
			})
		}
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Playlist name is required",
		})
	}

	// Split comma-separated track IDs
	trackIDs := strings.Split(req.TrackIDs, ",")
	for i, id := range trackIDs {
		trackIDs[i] = strings.TrimSpace(id)
	}

	downloader := strings.Clone(c.Query("downloader", "dummy"))
	jobID, err := h.service.DownloadPlaylist(downloader, trackIDs, req.PlaylistName)
	if err != nil {
		slog.Error("Failed to start playlist download", "error", err)
		if c.Get("HX-Request") == "true" {
			return c.Render("toast/toastErr", fiber.Map{
				"Msg": "Failed to start playlist download",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to start download",
		})
	}

	if c.Get("HX-Request") == "true" {
		return c.Render("toast/toastOk", fiber.Map{
			"Msg": fmt.Sprintf("Playlist '%s' download started", req.PlaylistName),
		})
	}
	return c.JSON(fiber.Map{
		"jobId":   jobID,
		"message": "Download started",
	})
}

// GetAlbumTracksRequest represents a request to get album tracks
type GetAlbumTracksRequest struct {
	AlbumID string `json:"albumId" form:"albumId"`
}

// GetAlbumTracks handles requests to get tracks from an album
func (h *Handler) GetAlbumTracks(c *fiber.Ctx) error {
	slog.Debug("GetAlbumTracks handler called")

	albumID := c.Params("albumId")
	if albumID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Album ID is required",
		})
	}

	downloader := strings.Clone(c.Query("downloader", "dummy"))
	tracks, err := h.service.GetAlbumTracks(downloader, albumID)
	if err != nil {
		slog.Error("Failed to get album tracks", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get album tracks",
		})
	}

	// Create album object for template
	album := &music.Album{ID: albumID, Title: "Album"} // This should be fetched from the service
	if len(tracks) > 0 && tracks[0].Album != nil {
		album = tracks[0].Album
	}

	// Calculate total duration
	var totalDuration int
	for _, track := range tracks {
		totalDuration += track.Metadata.Duration
	}

	return c.Render("downloading/album_tracks", fiber.Map{
		"Album":         album,
		"Tracks":        tracks,
		"TotalDuration": totalDuration,
		"Downloader":    downloader,
	})
}

// GetChartTracksHTMX handles HTMX requests for chart tracks
func (h *Handler) GetChartTracks(c *fiber.Ctx) error {
	limit := 20
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}

	downloader := strings.Clone(c.Query("downloader", "dummy"))

	// Get downloader capabilities
	var caps DownloaderCapabilities
	if d, exists := h.service.pluginManager.GetDownloader(downloader); exists {
		caps = d.Capabilities()
	}

	// If downloader doesn't support chart tracks, show not supported message
	if !caps.SupportsChartTracks {
		// Get the downloader name
		downloaderName := downloader
		if d, exists := h.service.pluginManager.GetDownloader(downloader); exists {
			downloaderName = d.Name()
		}
		return c.Render("downloading/chart_tracks", fiber.Map{
			"Tracks":         []music.Track{},
			"NotSupported":   true,
			"DownloaderName": downloaderName,
			"Downloader":     downloader,
		})
	}

	// Always try to fetch tracks, even if the downloader is disabled
	tracks, err := h.service.GetChartTracks(downloader, limit)

	// Get status for error message context
	statuses := h.service.GetDownloaderStatuses()

	// Get the downloader name and use it for status lookup
	downloaderName := downloader
	if d, exists := h.service.pluginManager.GetDownloader(downloader); exists {
		downloaderName = d.Name()
	}
	downloaderKey := strings.ToLower(downloaderName)
	downloaderStatus := statuses[downloaderKey]

	if err != nil || downloaderStatus.Status != "valid" {
		return c.Render("downloading/chart_tracks", fiber.Map{
			"Tracks":           []music.Track{},
			"DownloaderStatus": downloaderStatus,
			"DownloaderName":   downloaderName,
			"Downloader":       downloader,
		})
	}
	return c.Render("downloading/chart_tracks", fiber.Map{
		"Tracks":           tracks,
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

	// Get the downloader name and use it for status lookup
	downloaderName := downloader
	if d, exists := h.service.pluginManager.GetDownloader(downloader); exists {
		downloaderName = d.Name()
	}
	downloaderKey := strings.ToLower(downloaderName)
	downloaderStatus := statuses[downloaderKey]

	// Check if any downloaders are available
	hasDownloaders := len(h.service.pluginManager.GetAllDownloaders()) > 0

	// Get downloader capabilities
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
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	return c.JSON(caps)
}
