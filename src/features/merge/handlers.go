package merge

import (
	"fmt"
	"log/slog"

	"github.com/contre95/soulsolid/src/features/hosting/respond"
	"github.com/gofiber/fiber/v2"
)

// Handler handles HTTP requests for the merge feature.
type Handler struct {
	service *Service
}

// NewHandler creates a new merge handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RenderMergeSection renders the Merge analyze section (full page or HTMX partial).
func (h *Handler) RenderMergeSection(c *fiber.Ctx) error {
	return respond.Section(c, "analyze_merge", fiber.Map{"Title": "Merge Metadata"})
}

// RenderArtistGroups scans for and renders the candidate artist merge groups. The response also
// carries an out-of-band toast (see the group templates) reporting how many groups were found.
func (h *Handler) RenderArtistGroups(c *fiber.Ctx) error {
	groups, err := h.service.FindArtistGroups(c.Context())
	if err != nil {
		slog.Error("failed to find artist merge groups", "error", err)
		return respond.ToastErr(c, fiber.StatusInternalServerError, "Failed to find duplicate artists: "+err.Error())
	}
	return respond.HTMX(c, "merge/artist_groups", fiber.Map{"Groups": groups, "Msg": scanMsg("artist", len(groups))})
}

// RenderAlbumGroups scans for and renders the candidate album merge groups.
func (h *Handler) RenderAlbumGroups(c *fiber.Ctx) error {
	groups, err := h.service.FindAlbumGroups(c.Context())
	if err != nil {
		slog.Error("failed to find album merge groups", "error", err)
		return respond.ToastErr(c, fiber.StatusInternalServerError, "Failed to find duplicate albums: "+err.Error())
	}
	return respond.HTMX(c, "merge/album_groups", fiber.Map{"Groups": groups, "Msg": scanMsg("album", len(groups))})
}

// RenderGenreGroups scans for and renders the candidate genre merge groups.
func (h *Handler) RenderGenreGroups(c *fiber.Ctx) error {
	groups, err := h.service.FindGenreGroups(c.Context())
	if err != nil {
		slog.Error("failed to find genre merge groups", "error", err)
		return respond.ToastErr(c, fiber.StatusInternalServerError, "Failed to find duplicate genres: "+err.Error())
	}
	return respond.HTMX(c, "merge/genre_groups", fiber.Map{"Groups": groups, "Msg": scanMsg("genre", len(groups))})
}

// RenderTrackGroups scans for and renders the candidate duplicate-track groups (same AcoustID).
func (h *Handler) RenderTrackGroups(c *fiber.Ctx) error {
	groups, err := h.service.FindTrackGroups(c.Context())
	if err != nil {
		slog.Error("failed to find duplicate track groups", "error", err)
		return respond.ToastErr(c, fiber.StatusInternalServerError, "Failed to find duplicate tracks: "+err.Error())
	}
	return respond.HTMX(c, "merge/track_groups", fiber.Map{"Groups": groups, "Msg": scanMsg("duplicate track", len(groups))})
}

// scanMsg builds the human-readable result message shown in the scan toast.
func scanMsg(kind string, n int) string {
	switch n {
	case 0:
		return fmt.Sprintf("No duplicate %ss found", kind)
	case 1:
		return fmt.Sprintf("Found 1 %s group to review", kind)
	default:
		return fmt.Sprintf("Found %d %s groups to review", n, kind)
	}
}

// MergeArtists starts a merge job for a group of artists.
func (h *Handler) MergeArtists(c *fiber.Ctx) error { return h.startMerge(c, KindArtist) }

// MergeAlbums starts a merge job for a group of albums.
func (h *Handler) MergeAlbums(c *fiber.Ctx) error { return h.startMerge(c, KindAlbum) }

// MergeGenres starts a merge job for a group of genres.
func (h *Handler) MergeGenres(c *fiber.Ctx) error { return h.startMerge(c, KindGenre) }

// MergeTracks starts a merge job for a group of duplicate tracks: every checked file ("delete")
// is hard-deleted from the library and from disk; the unchecked ones stay.
func (h *Handler) MergeTracks(c *fiber.Ctx) error {
	all := formValues(c, "all")
	remove := formValues(c, "delete")
	cardID := c.FormValue("card_id")
	if _, err := h.service.StartTrackMerge(c.Context(), all, remove); err != nil {
		slog.Error("failed to start track merge", "error", err)
		return h.mergeResult(c, "", false, "Failed to start merge: "+err.Error())
	}
	c.Set("HX-Trigger", "refreshJobList")
	return h.mergeResult(c, cardID, true, "Merge started")
}

// KeepTracks marks a duplicate-track group as intentional: every track of the group is recorded
// as a known duplicate in the database (files and tags untouched) and the group leaves future
// scans. The checkboxes play no role here — all files are kept.
func (h *Handler) KeepTracks(c *fiber.Ctx) error {
	acoustID := c.FormValue("acoustid")
	members := formValues(c, "all")
	cardID := c.FormValue("card_id")
	if err := h.service.KeepTracks(c.Context(), acoustID, members); err != nil {
		slog.Error("failed to keep duplicate tracks", "acoustID", acoustID, "error", err)
		return h.mergeResult(c, "", false, "Failed to keep duplicates: "+err.Error())
	}
	return h.mergeResult(c, cardID, true, "Kept all files — group marked as intentional duplicates")
}

func (h *Handler) startMerge(c *fiber.Ctx, kind Kind) error {
	canonical := c.FormValue("canonical")
	members := formValues(c, "members")
	cardID := c.FormValue("card_id")
	if canonical == "" || len(members) < 2 {
		return h.mergeResult(c, "", false, "Select at least two items and a canonical value to merge")
	}
	if _, err := h.service.StartMerge(c.Context(), kind, canonical, members); err != nil {
		slog.Error("failed to start merge", "kind", kind, "error", err)
		return h.mergeResult(c, "", false, "Failed to start merge: "+err.Error())
	}
	c.Set("HX-Trigger", "refreshJobList")
	return h.mergeResult(c, cardID, true, "Merge started")
}

// mergeResult returns a 200 response carrying only out-of-band swaps: a toast and, on success,
// a directive that deletes the merged group's card (cardID) from its panel. Using OOB + a 200
// status (instead of relying on error-status swaps, which this app does not configure) keeps the
// behaviour consistent for both the success and error paths.
func (h *Handler) mergeResult(c *fiber.Ctx, cardID string, ok bool, msg string) error {
	return respond.HTMX(c, "merge/merge_result", fiber.Map{"OK": ok, "CardID": cardID, "Msg": msg})
}

// formValues returns every value submitted under key (Fiber's FormValue only returns the first).
func formValues(c *fiber.Ctx, key string) []string {
	var out []string
	c.Request().PostArgs().VisitAll(func(k, v []byte) {
		if string(k) == key {
			out = append(out, string(v))
		}
	})
	return out
}
