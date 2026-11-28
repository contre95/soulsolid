package albumcompleteness

import (
	"errors"
	"log/slog"
	"strconv"
	"strings"

	"github.com/contre95/soulsolid/src/hosting/respond"
	"github.com/gofiber/fiber/v3"
)

// incompletePageSize is how many incomplete albums the list shows per page.
const incompletePageSize = 25

// Handler handles HTTP requests for the album completeness feature.
type Handler struct {
	service *Service
}

// NewHandler creates a new album completeness handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Pagination represents pagination information for the incomplete albums list.
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

// NewPagination creates a Pagination with calculated values.
func NewPagination(page, limit, totalCount int) Pagination {
	totalPages := (totalCount + limit - 1) / limit
	return Pagination{
		Page:       page,
		Limit:      limit,
		TotalCount: totalCount,
		TotalPages: totalPages,
		NextPage:   page + 1,
		PrevPage:   page - 1,
		HasNext:    page < totalPages,
		HasPrev:    page > 1,
	}
}

// RenderCompletenessSection renders the album completeness analyze section (full page or HTMX partial).
func (h *Handler) RenderCompletenessSection(c fiber.Ctx) error {
	return respond.Section(c, "analyze_completeness", fiber.Map{"Title": "Album Completeness"})
}

// StartAnalysis starts the album completeness job. Pass force=true to re-check every album.
func (h *Handler) StartAnalysis(c fiber.Ctx) error {
	force := c.FormValue("force") == "true"
	jobID, err := h.service.StartAnalysis(c.Context(), force)
	if err != nil {
		slog.Error("Failed to start album completeness analysis", "error", err)
		return respond.ToastErr(c, fiber.StatusInternalServerError, "Failed to start analysis: "+err.Error())
	}
	respond.Trigger(c, "refreshJobList")
	return respond.ToastJob(c, jobID, "Album completeness analysis started")
}

// RenderIncompleteAlbums renders one page of the albums that are missing tracks.
func (h *Handler) RenderIncompleteAlbums(c fiber.Ctx) error {
	page := max(fiber.Query[int](c, "page", 1), 1)
	albums, count, err := h.service.GetIncompleteAlbums(c.Context(), incompletePageSize, (page-1)*incompletePageSize)
	if err != nil {
		slog.Error("Failed to list incomplete albums", "error", err)
		return respond.ToastErr(c, fiber.StatusInternalServerError, "Failed to load incomplete albums")
	}
	// A stale page (e.g. albums completed since the link was built) falls back to the last one.
	if lastPage := max((count+incompletePageSize-1)/incompletePageSize, 1); page > lastPage {
		page = lastPage
		albums, _, err = h.service.GetIncompleteAlbums(c.Context(), incompletePageSize, (page-1)*incompletePageSize)
		if err != nil {
			slog.Error("Failed to list incomplete albums", "error", err)
			return respond.ToastErr(c, fiber.StatusInternalServerError, "Failed to load incomplete albums")
		}
	}
	return respond.Partial(c, "albumcompleteness/incomplete_list", fiber.Map{
		"Albums":     albums,
		"Pagination": NewPagination(page, incompletePageSize, count),
	})
}

// RenderAlbumBadge renders the completeness badge of an album from stored data, without contacting any provider.
func (h *Handler) RenderAlbumBadge(c fiber.Ctx) error {
	return h.renderBadge(c, c.Params("id"), "")
}

// CheckAlbum asks the providers for an album's official track count and renders the refreshed badge.
func (h *Handler) CheckAlbum(c fiber.Ctx) error {
	albumID := c.Params("id")
	_, err := h.service.CheckAlbum(c.Context(), albumID)
	switch {
	case err == nil:
		return h.renderBadge(c, albumID, "")
	case errors.Is(err, ErrAlbumNotFound):
		return respond.ToastErr(c, fiber.StatusNotFound, "Album not found")
	case errors.Is(err, ErrNoMatch):
		return h.renderBadge(c, albumID, err.Error())
	default:
		slog.Error("Failed to check album completeness", "albumID", albumID, "error", err)
		return h.renderBadge(c, albumID, err.Error())
	}
}

// SetExpectedTracks stores a manually entered track count and renders the refreshed badge.
func (h *Handler) SetExpectedTracks(c fiber.Ctx) error {
	albumID := c.Params("id")
	expected, err := strconv.Atoi(strings.TrimSpace(c.FormValue("expected")))
	if err != nil {
		return h.renderBadge(c, albumID, "Enter the number of tracks as a whole number")
	}
	if _, err := h.service.SetExpectedTracks(c.Context(), albumID, expected); err != nil {
		if errors.Is(err, ErrAlbumNotFound) {
			return respond.ToastErr(c, fiber.StatusNotFound, "Album not found")
		}
		return h.renderBadge(c, albumID, err.Error())
	}
	return h.renderBadge(c, albumID, "")
}

// renderBadge renders the badge of an album. errMsg is shown next to it when not empty.
func (h *Handler) renderBadge(c fiber.Ctx, albumID, errMsg string) error {
	completeness, err := h.service.GetAlbumCompleteness(c.Context(), albumID)
	if err != nil {
		if errors.Is(err, ErrAlbumNotFound) {
			return respond.ToastErr(c, fiber.StatusNotFound, "Album not found")
		}
		slog.Error("Failed to get album completeness", "albumID", albumID, "error", err)
		return respond.ToastErr(c, fiber.StatusInternalServerError, "Failed to load album completeness")
	}
	return respond.Partial(c, "albumcompleteness/badge", fiber.Map{
		"C":     completeness,
		"Error": errMsg,
	})
}
