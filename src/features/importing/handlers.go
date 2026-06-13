package importing

import (
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"sort"
	"time"

	"github.com/contre95/soulsolid/src/features/hosting/respond"
	"github.com/contre95/soulsolid/src/music"
	"github.com/gofiber/fiber/v2"
)

// Handler is the handler for the organizing feature.
type Handler struct {
	service *Service
}

// queueItemView is a view model for queue items that includes the track. A single item can
// carry several statuses at once, so the booleans below are not mutually exclusive.
type queueItemView struct {
	ID           string
	Timestamp    time.Time
	JobID        string
	Track        *music.Track
	ItemMetadata map[string]string

	// Status flags (derived from the item's types; may be true in combination)
	IsDuplicate       bool
	IsMissingMetadata bool
	IsFailedImport    bool
	IsManualReview    bool

	// Action availability, implementing the "block until metadata is fixed" rule
	ShowReplace    bool
	ReplaceEnabled bool
	ShowImport     bool
	ImportEnabled  bool
	CancelLabel    string // "Skip" for duplicates/failed imports, otherwise "Cancel"
	BlockReason    string // tooltip shown on disabled import/replace buttons
}

// groupView is a view model for grouped queue items
type groupView struct {
	Items         []queueItemView
	HasImportable bool // true if at least one non-duplicate importable item exists
	HasDuplicates bool // true if at least one duplicate item exists
}

// convertQueueItem converts a music.QueueItem to queueItemView
func convertQueueItem(item music.QueueItem) (queueItemView, error) {
	if item.Track == nil {
		return queueItemView{}, errors.New("queue item has no track")
	}
	isDup := item.HasType(music.Duplicate)
	isMissing := item.HasType(music.MissingMetadata)
	isFailed := item.HasType(music.FailedImport)
	isManual := item.HasType(music.ManualReview)

	view := queueItemView{
		ID:                item.ID,
		Timestamp:         item.Timestamp,
		JobID:             item.JobID,
		Track:             item.Track,
		ItemMetadata:      item.Metadata,
		IsDuplicate:       isDup,
		IsMissingMetadata: isMissing,
		IsFailedImport:    isFailed,
		IsManualReview:    isManual,
	}

	// Replace is a duplicate-only action; it is blocked while metadata is missing or the
	// import failed, so the library is never overwritten with an incomplete track.
	view.ShowReplace = isDup
	view.ReplaceEnabled = isDup && !isMissing && !isFailed

	// Import applies to manual-review / missing-metadata items (never duplicates or failed
	// imports). It stays disabled until the missing required metadata is fixed.
	view.ShowImport = !isDup && !isFailed && (isManual || isMissing)
	view.ImportEnabled = view.ShowImport && !isMissing

	view.CancelLabel = "Cancel"
	if isDup || isFailed {
		view.CancelLabel = "Skip"
	}
	if isMissing {
		view.BlockReason = "Fix the missing metadata before importing"
	}
	return view, nil
}

// NewHandler creates a new handler for the organizing feature.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RenderImportSection renders the import page.
func (h *Handler) RenderImportSection(c *fiber.Ctx) error {
	slog.Debug("RenderImport handler called")
	return respond.Section(c, "import", fiber.Map{"Title": "Import"})
}

// ImportDirectory is the handler for importing a directory.
func (h *Handler) ImportDirectory(c *fiber.Ctx) error {
	type ImportPathRequest struct {
		DirectoryPath string `json:"directoryPath"`
	}
	var req ImportPathRequest
	if err := c.BodyParser(&req); err != nil {
		return respond.ToastErr(c, fiber.StatusBadRequest, "Cannot parse request body")
	}
	jobID, err := h.service.ImportDirectory(c.Context(), req.DirectoryPath)
	if err != nil {
		slog.Error("Error importing directory", "error", err)
		return respond.ToastErr(c, fiber.StatusInternalServerError, "Failed to start sync job")
	}
	slog.Info("ImportDirectory: directory import started", "jobID", jobID)
	c.Response().Header.Set("HX-Trigger", "jobStarted,queueUpdated,refreshImportQueueBadge")
	return respond.ToastJob(c, jobID, "Directory import started!")
}

// ProcessQueueItem handles import/cancel actions for individual queue items
func (h *Handler) ProcessQueueItem(c *fiber.Ctx) error {
	itemID := c.Params("id")
	action := c.Params("action") // "import" or "cancel"
	err := h.service.ProcessQueueItem(c.Context(), itemID, action)
	if err != nil {
		slog.Error("Failed to process queue item", "error", err, "itemID", itemID, "action", action)
		return respond.ToastErr(c, fiber.StatusInternalServerError, "Failed to process queue item")
	}
	actionMsg := "skipped"
	switch action {
	case "import":
		actionMsg = "imported"
	case "replace":
		actionMsg = "replaced"
	case "delete":
		actionMsg = "deleted"
	}
	c.Response().Header.Set("HX-Trigger", "queueUpdated,refreshImportQueueBadge,activateIndividualGrouping")
	return respond.ToastOk(c, fmt.Sprintf("Track %s successfully", actionMsg))
}

// QueueCount returns the current queue count formatted as "(X)" or empty if 0
func (h *Handler) QueueCount(c *fiber.Ctx) error {
	count := len(h.service.GetQueuedItems())
	formatted := ""
	if count > 0 {
		formatted = fmt.Sprintf("(%d)", count)
	}
	return respond.Text(c, "queue_count", count, formatted)
}

// ClearQueue handles clearing all items from the import queue
func (h *Handler) ClearQueue(c *fiber.Ctx) error {
	err := h.service.ClearQueue()
	if err != nil {
		slog.Error("Failed to clear queue", "error", err)
		return respond.ToastErr(c, fiber.StatusInternalServerError, "Failed to clear queue")
	}
	c.Response().Header.Set("HX-Trigger", "queueUpdated,refreshImportQueueBadge,activateIndividualGrouping")
	return respond.ToastOk(c, "Queue cleared successfully")
}

// PruneDownloadPath handles pruning the download path and clearing the queue
func (h *Handler) PruneDownloadPath(c *fiber.Ctx) error {
	err := h.service.PruneDownloadPath(c.Context())
	if err != nil {
		slog.Error("Failed to prune download path", "error", err)
		return respond.ToastErr(c, fiber.StatusInternalServerError, "Failed to prune download path")
	}
	c.Response().Header.Set("HX-Trigger", "queueUpdated,refreshImportQueueBadge,activateIndividualGrouping")
	return respond.ToastOk(c, "Download path pruned and queue cleared successfully")
}

// ToggleWatcher toggles the file system watcher on/off
func (h *Handler) ToggleWatcher(c *fiber.Ctx) error {
	action := c.FormValue("action")
	if action == "" {
		return respond.ToastErr(c, fiber.StatusBadRequest, "action parameter required")
	}

	var err error
	var msg string

	switch action {
	case "start":
		err = h.service.StartWatcher()
		msg = "File watcher started successfully"
	case "stop":
		err = h.service.StopWatcher()
		msg = "File watcher stopped successfully"
	default:
		return respond.ToastErr(c, fiber.StatusBadRequest, "invalid action")
	}

	if err != nil {
		slog.Error("Failed to toggle watcher", "action", action, "error", err)
		return respond.ToastErr(c, fiber.StatusInternalServerError, "Failed to "+action+" file watcher")
	}

	c.Response().Header.Set("HX-Trigger", "watcherStatusChanged")
	return respond.ToastOk(c, msg)
}

// GetWatcherStatus returns the current status of the watcher
func (h *Handler) GetWatcherStatus(c *fiber.Ctx) error {
	running := h.service.GetWatcherStatus()
	return respond.Partial(c, "components/status_badge", fiber.Map{
		"Running": running,
	})
}

// GetWatcherToggleState returns the toggle input element with correct checked state
func (h *Handler) GetWatcherToggleState(c *fiber.Ctx) error {
	running := h.service.GetWatcherStatus()
	return respond.Partial(c, "components/toggle", fiber.Map{
		"ID":      "watcher-toggle",
		"Checked": running,
		"PostURL": "/import/watcher/toggle",
		"Vals":    "js:{action: event.target.checked ? 'start' : 'stop'}",
	})
}

// UI Hanlders
// GetDirectoryForm renders the directory import form
func (h *Handler) GetDirectoryForm(c *fiber.Ctx) error {
	slog.Debug("GetDirectoryForm handler called")

	// Get default download path from service
	defaultDownloadPath := h.service.config.Get().DownloadPath

	return respond.Partial(c, "importing/directory_form", fiber.Map{
		"DefaultDownloadPath": defaultDownloadPath,
		"Config":              h.service.config.Get(),
	})
}

// RenderQueueItems renders the queue content for HTMX
func (h *Handler) RenderQueueItems(c *fiber.Ctx) error {
	slog.Debug("RenderImportQueue handler called")
	// Get queue data from importing service
	c.Response().Header.Set("HX-Trigger", "updateQueueCount")
	queueItemsMap := h.service.GetQueuedItems()

	// Collect all items into a slice of view models
	queueItems := make([]queueItemView, 0, len(queueItemsMap))
	for _, item := range queueItemsMap {
		view, err := convertQueueItem(item)
		if err != nil {
			slog.Error("Failed to convert queue item", "error", err, "itemID", item.ID)
			continue // skip items with invalid payload
		}
		queueItems = append(queueItems, view)
	}

	// Sort by timestamp (oldest first)
	sort.Slice(queueItems, func(i, j int) bool {
		return queueItems[i].Timestamp.Before(queueItems[j].Timestamp)
	})

	// Limit to 10 items for better performance and UX
	if len(queueItems) > 10 {
		queueItems = queueItems[:10]
	}

	return respond.Partial(c, "importing/queue_items", fiber.Map{
		"QueueItems": queueItems,
	})
}

// GetQueueHeader renders the queue header for HTMX
func (h *Handler) GetQueueHeader(c *fiber.Ctx) error {
	slog.Debug("GetQueueHeader handler called")

	// Get queue count for display
	queueItems := h.service.GetQueuedItems()
	queueCount := len(queueItems)

	return respond.Partial(c, "importing/queue_header", fiber.Map{
		"QueueCount": queueCount,
	})
}

// ProcessQueueGroup handles bulk actions for queue item groups
func (h *Handler) ProcessQueueGroup(c *fiber.Ctx) error {
	groupKey := c.Params("groupKey")
	groupType := c.Params("groupType") // "artist" or "album"
	action := c.Params("action")       // "import", "cancel", "delete", "replace"

	// URL-decode the groupKey since it may contain encoded characters
	decodedGroupKey, err := url.QueryUnescape(groupKey)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid groupKey encoding",
		})
	}

	if groupType != "artist" && groupType != "album" {
		return respond.ToastErr(c, fiber.StatusBadRequest, "groupType must be 'artist' or 'album'")
	}

	if action != "import" && action != "cancel" && action != "delete" && action != "replace" {
		return respond.ToastErr(c, fiber.StatusBadRequest, "action must be one of: import, cancel, delete, replace")
	}

	err = h.service.ProcessQueueGroup(c.Context(), decodedGroupKey, groupType, action)
	if err != nil {
		slog.Error("Failed to process group", "error", err, "groupKey", decodedGroupKey, "groupType", groupType, "action", action)
		return respond.ToastErr(c, fiber.StatusInternalServerError, fmt.Sprintf("Failed to process group %s", decodedGroupKey))
	}

	actionMsg := "processed"
	switch action {
	case "import":
		actionMsg = "imported"
	case "replace":
		actionMsg = "replaced"
	case "cancel":
		actionMsg = "skipped"
	case "delete":
		actionMsg = "deleted"
	}

	trigger := "queueUpdated,refreshImportQueueBadge"
	if groupType == "artist" {
		trigger += ",activateArtistGrouping"
	} else {
		trigger += ",activateAlbumGrouping"
	}
	c.Response().Header.Set("HX-Trigger", trigger)
	return respond.ToastOk(c, fmt.Sprintf("Group '%s' %s successfully", decodedGroupKey, actionMsg))
}

// RenderGroupedQueueItems renders queue items grouped by artist or album
func (h *Handler) RenderGroupedQueueItems(c *fiber.Ctx) error {
	groupType := c.Query("type", "artist") // default to artist grouping

	var groups map[string][]music.QueueItem
	var templateName string

	if groupType == "album" {
		groups = h.service.GetGroupedByAlbum()
		templateName = "importing/queue_items_grouped_album"
	} else {
		groups = h.service.GetGroupedByArtist()
		templateName = "importing/queue_items_grouped_artist"
	}

	// Convert groups to view models
	viewGroups := make(map[string]groupView)
	for groupKey, items := range groups {
		viewItems := make([]queueItemView, 0, len(items))
		hasImportable := false
		hasDuplicates := false
		for _, item := range items {
			view, err := convertQueueItem(item)
			if err != nil {
				slog.Error("Failed to convert queue item", "error", err, "itemID", item.ID)
				continue
			}
			viewItems = append(viewItems, view)
			// Only surface the bulk replace button when at least one item can
			// actually be replaced; a duplicate that is missing metadata or failed
			// import has IsDuplicate==true but ReplaceEnabled==false.
			if view.ReplaceEnabled {
				hasDuplicates = true
			}
			if view.ImportEnabled {
				hasImportable = true
			}
		}
		// Sort items within each group by timestamp
		sort.Slice(viewItems, func(i, j int) bool {
			return viewItems[i].Timestamp.Before(viewItems[j].Timestamp)
		})
		viewGroups[groupKey] = groupView{
			Items:         viewItems,
			HasImportable: hasImportable,
			HasDuplicates: hasDuplicates,
		}
	}

	return respond.Partial(c, templateName, fiber.Map{
		"Groups":    viewGroups,
		"GroupType": groupType,
	})
}

// ServeQueueItemArtwork serves the embedded album art for a queue item's track file.
func (h *Handler) ServeQueueItemArtwork(c *fiber.Ctx) error {
	id := c.Params("id")
	data, mimeType, err := h.service.GetPendingTrackArtwork(id)
	if err != nil || len(data) == 0 {
		return c.Status(fiber.StatusNotFound).SendString("No artwork found")
	}
	c.Set("Content-Type", mimeType)
	c.Set("Cache-Control", "public, max-age=3600")
	return c.Send(data)
}
