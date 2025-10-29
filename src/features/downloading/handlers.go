package downloading

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/contre95/soulsolid/src/features/jobs"
	"github.com/contre95/soulsolid/src/music"
	"github.com/gofiber/fiber/v2"
)

// Handler handles HTTP requests for downloading
type Handler struct {
	service    *Service
	jobService jobs.JobService
}

// NewHandler creates a new downloading handler
func NewHandler(service *Service, jobService jobs.JobService) *Handler {
	return &Handler{
		service:    service,
		jobService: jobService,
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

	// If type is "link", treat as search with URL
	if req.Type == "link" {
		tracks, err := h.service.SearchTracks(req.Downloader, req.Query, req.Limit)
		if err != nil {
			if c.Get("HX-Request") == "true" {
				return c.Render("toast/toastErr", fiber.Map{
					"Msg": "Failed to process link: " + err.Error(),
				})
			}
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Failed to process link",
			})
		}
		if c.Get("HX-Request") == "true" {
			return h.renderTrackResults(c, tracks, req.Downloader)
		}
		return c.JSON(fiber.Map{
			"tracks": tracks,
		})
	}

	// Auto-detect URLs for link-supporting downloaders
	if strings.Contains(req.Query, "://") {
		if d, exists := h.service.pluginManager.GetDownloader(req.Downloader); exists {
			caps := d.Capabilities()
			if caps.SupportsDirectLinks {
				tracks, err := h.service.SearchTracks(req.Downloader, req.Query, req.Limit)
				if err == nil {
					if c.Get("HX-Request") == "true" {
						return h.renderTrackResults(c, tracks, req.Downloader)
					}
					return c.JSON(fiber.Map{
						"tracks": tracks,
					})
				}
				// If failed, fall through to normal search
			}
		}
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
		case "track", "all":
			tracks, err := h.service.SearchTracks(req.Downloader, req.Query, req.Limit)
			if err != nil {
				slog.Error("Failed to search tracks", "error", err)
				return c.Render("toast/toastErr", fiber.Map{
					"Msg": "Failed to search tracks",
				})
			}
			return h.renderTrackResults(c, tracks, req.Downloader)
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
	case "track", "all":
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

	jobID, err := h.service.DownloadTrack(c.Query("downloader", "dummy"), req.TrackID)
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

	jobID, err := h.service.DownloadAlbum(c.Query("downloader", "dummy"), req.AlbumID)
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

// GetDownloadStatus returns the status of a download job
func (h *Handler) GetDownloadStatus(c *fiber.Ctx) error {
	jobID := c.Params("jobId")
	slog.Debug("GetDownloadStatus handler called", "jobID", jobID)

	job, exists := h.jobService.GetJob(jobID)
	if !exists {
		if c.Get("HX-Request") == "true" {
			return c.Render("toast/toastErr", fiber.Map{
				"Msg": "Job not found",
			})
		}
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Job not found",
		})
	}

	if c.Get("HX-Request") == "true" {
		return c.Render("toast/toastInfo", fiber.Map{
			"Msg": fmt.Sprintf("Job %s: %s", job.Status, job.Message),
		})
	}
	return c.JSON(job)
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

	tracks, err := h.service.GetAlbumTracks(c.Query("downloader", "dummy"), albumID)
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
		"Downloader":    c.Query("downloader", "dummy"),
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

	downloader := c.Query("downloader", "dummy")

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
	downloader := c.Query("downloader", "dummy")
	userInfo := h.service.GetUserInfo(downloader)
	config := h.service.configManager.Get()
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
			"Config":            config,
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
	downloader := c.Query("downloader", "dummy")
	caps, err := h.service.GetDownloaderCapabilities(downloader)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	return c.JSON(caps)
}
