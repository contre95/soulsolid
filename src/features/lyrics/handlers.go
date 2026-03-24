package lyrics

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"sort"
	"time"

	"github.com/contre95/soulsolid/src/music"
	"github.com/gofiber/fiber/v2"
)

// Handler handles lyrics requests
type Handler struct {
	service         *Service
	metadataService MetadataService // For accessing track data and UI rendering
}

// MetadataService interface for accessing metadata functionality needed by lyrics handlers
type MetadataService interface {
	GetTrackFileTags(ctx context.Context, trackID string) (*music.Track, error)
	GetArtists(ctx context.Context) ([]*music.Artist, error)
	GetAlbums(ctx context.Context) ([]*music.Album, error)
	GetEnabledMetadataProviders() map[string]bool
	GetEnabledLyricsProviders() map[string]bool
}

// NewHandler creates a new lyrics handler
func NewHandler(service *Service, metadataService MetadataService) *Handler {
	return &Handler{
		service:         service,
		metadataService: metadataService,
	}
}

// queueItemView is a view model for queue items that includes the track
type queueItemView struct {
	ID           string
	Type         string
	Timestamp    time.Time
	Track        *music.Track
	ItemMetadata map[string]string
}

// groupView is a view model for grouped queue items
type groupView struct {
	Items []queueItemView
}

// convertQueueItem converts a music.QueueItem to queueItemView
func convertQueueItem(item music.QueueItem) (queueItemView, error) {
	if item.Track == nil {
		slog.Error("Queue item has no track", "itemID", item.ID, "type", item.Type)
		return queueItemView{}, errors.New("queue item has no track")
	}
	return queueItemView{
		ID:           item.ID,
		Type:         string(item.Type),
		Timestamp:    item.Timestamp,
		Track:        item.Track,
		ItemMetadata: item.Metadata,
	}, nil
}

// RenderLyricsButtons renders the lyrics provider buttons for a track
func (h *Handler) RenderLyricsButtons(c *fiber.Ctx) error {
	trackID := c.Params("trackId")
	if trackID == "" {
		return c.Status(fiber.StatusBadRequest).SendString("Track ID is required")
	}

	// Get track data for button context
	track, err := h.metadataService.GetTrackFileTags(c.Context(), trackID)
	if err != nil {
		slog.Error("Failed to get track for lyrics buttons", "error", err, "trackId", trackID)
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to load track data")
	}

	return c.Render("tag/lyrics_buttons", fiber.Map{
		"Track":           track,
		"LyricsProviders": h.service.GetLyricsProvidersInfo(),
	})
}

// GetLyricsText returns plain lyrics text for HTMX to set in textarea
func (h *Handler) GetLyricsText(c *fiber.Ctx) error {
	trackID := c.Params("trackId")
	providerName := c.Params("provider")

	if trackID == "" || providerName == "" {
		return c.Status(fiber.StatusBadRequest).SendString("Track ID and provider name are required")
	}

	// Fetch lyrics
	lyrics, err := h.service.SearchLyrics(c.Context(), trackID, providerName)
	if err != nil {
		slog.Error("Failed to fetch lyrics", "error", err, "trackId", trackID, "provider", providerName)
		return c.SendString("") // Return empty string on error for HTMX
	}

	slog.Info("Lyrics fetched successfully", "trackId", trackID, "provider", providerName, "lyricsLength", len(lyrics))

	// Return plain lyrics text for HTMX to set in textarea
	return c.SendString(lyrics)
}

// GetTrackLyrics returns the lyrics of a track in plain text.
func (h *Handler) GetTrackLyrics(c *fiber.Ctx) error {
	slog.Debug("GetTrackLyrics handler called", "id", c.Params("id"))
	track, err := h.service.libraryRepo.GetTrack(c.Context(), c.Params("id"))
	if err != nil {
		slog.Error("Error loading track", "error", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Error loading track")
	}
	if track == nil {
		return c.Status(fiber.StatusNotFound).SendString("Track not found")
	}
	return c.SendString(track.Metadata.Lyrics)
}

// GetQueueNewLyrics returns the new lyrics from a queue item's metadata.
func (h *Handler) GetQueueNewLyrics(c *fiber.Ctx) error {
	itemID := c.Params("id")
	queueItemsMap := h.service.GetLyricsQueueItems()
	item, ok := queueItemsMap[itemID]
	if !ok {
		return c.Status(fiber.StatusNotFound).SendString("Queue item not found")
	}
	if newLyrics, ok := item.Metadata["new_lyrics"]; ok && newLyrics != "" {
		return c.SendString(newLyrics)
	}
	return c.SendString("No new lyrics available")
}

// RenderLyricsQueueItems renders the lyrics queue content for HTMX
func (h *Handler) RenderLyricsQueueItems(c *fiber.Ctx) error {
	slog.Debug("RenderLyricsQueueItems handler called")
	queueItemsMap := h.service.GetLyricsQueueItems()
	slog.Info("Lyrics queue items", "count", len(queueItemsMap))

	// Collect all items into a slice of view models
	queueItems := make([]queueItemView, 0, len(queueItemsMap))
	for _, item := range queueItemsMap {
		view, err := convertQueueItem(item)
		if err != nil {
			slog.Error("Failed to convert queue item", "error", err, "itemID", item.ID)
			continue
		}
		queueItems = append(queueItems, view)
	}
	slog.Info("Successfully converted queue items", "converted", len(queueItems), "total", len(queueItemsMap))

	// Sort by timestamp (oldest first)
	sort.Slice(queueItems, func(i, j int) bool {
		return queueItems[i].Timestamp.Before(queueItems[j].Timestamp)
	})

	// Limit to 10 items for better performance and UX
	if len(queueItems) > 10 {
		queueItems = queueItems[:10]
	}
	if len(queueItems) > 0 {
		slog.Info("First queue item sample", "id", queueItems[0].ID, "type", queueItems[0].Type, "trackTitle", queueItems[0].Track.Title)
	}

	return c.Render("lyrics/queue_items", fiber.Map{
		"QueueItems": queueItems,
	})
}

// ProcessLyricsQueueItem handles actions for individual lyrics queue items
func (h *Handler) ProcessLyricsQueueItem(c *fiber.Ctx) error {
	itemID := c.Params("id")
	action := c.Params("action")
	slog.Info("Processing lyrics queue item", "itemID", itemID, "action", action)
	err := h.service.ProcessLyricsQueueItem(c.Context(), itemID, action)
	if err != nil {
		slog.Error("Failed to process lyrics queue item", "error", err, "itemID", itemID, "action", action)
		return c.Render("toast/toastErr", fiber.Map{
			"Msg": fmt.Sprintf("Failed to process lyrics queue item: %s", err.Error()),
		})
	}
	// Return success response that updates the UI
	actionMsg := "processed"
	switch action {
	case "override":
		actionMsg = "overridden"
	case "keep_old":
		actionMsg = "kept old"
	case "edit_manual":
		actionMsg = "marked for manual edit"
	case "no_lyrics":
		actionMsg = "marked as no lyrics"
	case "skip":
		actionMsg = "skipped"
	}
	c.Response().Header.Set("HX-Trigger", "lyricsQueueUpdated,refreshLyricsQueueBadge,updateLyricsQueueCount,activateIndividualGroupingLyrics")
	return c.Render("toast/toastOk", fiber.Map{
		"Msg": fmt.Sprintf("Track %s successfully", actionMsg),
	})
}

// LyricsQueueCount returns the current lyrics queue count formatted as "(X)" or empty if 0
func (h *Handler) LyricsQueueCount(c *fiber.Ctx) error {
	count := len(h.service.GetLyricsQueueItems())
	if count == 0 {
		return c.SendString("")
	}
	return c.SendString(fmt.Sprintf("(%d)", count))
}

// ClearLyricsQueue handles clearing all items from the lyrics queue
func (h *Handler) ClearLyricsQueue(c *fiber.Ctx) error {
	err := h.service.ClearLyricsQueue()
	if err != nil {
		slog.Error("Failed to clear lyrics queue", "error", err)
		return c.Render("toast/toastErr", fiber.Map{
			"Msg": "Failed to clear lyrics queue",
		})
	}
	c.Response().Header.Set("HX-Trigger", "lyricsQueueUpdated,refreshLyricsQueueBadge,updateLyricsQueueCount,activateIndividualGroupingLyrics")
	return c.Render("toast/toastOk", fiber.Map{
		"Msg": "Lyrics queue cleared successfully",
	})
}

// RenderGroupedLyricsQueueItems renders lyrics queue items grouped by artist or album
func (h *Handler) RenderGroupedLyricsQueueItems(c *fiber.Ctx) error {
	groupType := c.Query("type", "artist") // default to artist grouping

	var groups map[string][]music.QueueItem
	var templateName string

	if groupType == "album" {
		groups = h.service.GetLyricsGroupedByAlbum()
		templateName = "lyrics/queue_items_grouped_album"
	} else {
		groups = h.service.GetLyricsGroupedByArtist()
		templateName = "lyrics/queue_items_grouped_artist"
	}

	// Convert groups to view models
	viewGroups := make(map[string]groupView)
	for groupKey, items := range groups {
		viewItems := make([]queueItemView, 0, len(items))
		for _, item := range items {
			view, err := convertQueueItem(item)
			if err != nil {
				slog.Error("Failed to convert queue item", "error", err, "itemID", item.ID)
				continue
			}
			viewItems = append(viewItems, view)
		}
		// Sort items within each group by timestamp
		sort.Slice(viewItems, func(i, j int) bool {
			return viewItems[i].Timestamp.Before(viewItems[j].Timestamp)
		})
		viewGroups[groupKey] = groupView{
			Items: viewItems,
		}
	}

	return c.Render(templateName, fiber.Map{
		"Groups":    viewGroups,
		"GroupType": groupType,
	})
}

// ProcessLyricsQueueGroup processes all items in a group with the given action
func (h *Handler) ProcessLyricsQueueGroup(c *fiber.Ctx) error {
	groupKey := c.Params("groupKey")
	groupType := c.Params("groupType") // "artist" or "album"
	action := c.Params("action")

	// URL-decode the groupKey since it may contain encoded characters
	decodedGroupKey, err := url.QueryUnescape(groupKey)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid groupKey encoding",
		})
	}

	if groupType != "artist" && groupType != "album" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "groupType must be 'artist' or 'album'",
		})
	}

	err = h.service.ProcessLyricsQueueGroup(c.Context(), decodedGroupKey, groupType, action)
	if err != nil {
		slog.Error("Failed to process lyrics queue group", "error", err, "groupKey", decodedGroupKey, "groupType", groupType, "action", action)
		return c.Render("toast/toastErr", fiber.Map{
			"Msg": fmt.Sprintf("Failed to process group %s", decodedGroupKey),
		})
	}

	trigger := "lyricsQueueUpdated,refreshLyricsQueueBadge,updateLyricsQueueCount"
	if groupType == "artist" {
		trigger += ",activateArtistGroupingLyrics"
	} else if groupType == "album" {
		trigger += ",activateAlbumGroupingLyrics"
	}
	c.Response().Header.Set("HX-Trigger", trigger)
	return c.Render("toast/toastOk", fiber.Map{
		"Msg": fmt.Sprintf("Group '%s' processed successfully", decodedGroupKey),
	})
}

// RenderLyricsQueueHeader renders the lyrics queue header for HTMX
func (h *Handler) RenderLyricsQueueHeader(c *fiber.Ctx) error {
	slog.Debug("RenderLyricsQueueHeader handler called")
	count := len(h.service.GetLyricsQueueItems())
	return c.Render("lyrics/queue_header", fiber.Map{
		"QueueCount": count,
	})
}

// StartLyricsAnalysis handles starting the lyrics analysis job
func (h *Handler) StartLyricsAnalysis(c *fiber.Ctx) error {
	slog.Info("Starting lyrics analysis via HTTP request")

	// Get provider from form data
	provider := c.FormValue("provider")
	if provider == "" {
		return c.Render("toast/toastErr", fiber.Map{
			"Msg": "Please select a lyrics provider",
		})
	}

	jobID, err := h.service.StartLyricsAnalysis(c.Context(), provider)
	if err != nil {
		slog.Error("Failed to start lyrics analysis", "error", err)
		return c.Render("toast/toastErr", fiber.Map{
			"Msg": "Failed to start lyrics analysis: " + err.Error(),
		})
	}

	slog.Info("Lyrics analysis job started successfully", "jobID", jobID, "provider", provider)

	// Trigger HTMX to refresh the job list
	c.Set("HX-Trigger", "refreshJobList")

	if c.Get("HX-Request") == "true" {
		return c.Render("toast/toastOk", fiber.Map{
			"Msg": "Lyrics analysis started successfully",
		})
	}

	return c.Redirect("/ui/analyze/lyrics")
}

// RenderLyricsAnalysisSection renders the lyrics analysis section page
func (h *Handler) RenderLyricsAnalysisSection(c *fiber.Ctx) error {
	slog.Debug("Rendering lyrics analysis section")

	data := fiber.Map{
		"Title": "Lyrics Analysis",
	}

	// Get lyrics providers info for the UI
	data["LyricsProviders"] = h.service.GetLyricsProvidersInfo()

	if c.Get("HX-Request") != "true" {
		data["Section"] = "analyze_lyrics"
		return c.Render("main", data)
	}

	return c.Render("sections/analyze_lyrics", data)
}
