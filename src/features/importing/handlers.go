package importing

import (
	"fmt"
	"log/slog"
	"net/url"

	"github.com/gofiber/fiber/v2"
)

// Handler is the handler for the organizing feature.
type Handler struct {
	service *Service
}

// NewHandler creates a new handler for the organizing feature.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// ImportDirectory is the handler for importing a directory.
func (h *Handler) ImportDirectory(c *fiber.Ctx) error {
	type ImportPathRequest struct {
		DirectoryPath string `json:"directoryPath"`
	}
	var req ImportPathRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "cannot parse request body",
		})
	}
	jobID, err := h.service.ImportDirectory(c.Context(), req.DirectoryPath)
	if err != nil {
		slog.Error("Error importing directory", "error", err)
		return c.Render("toast/toastErr", fiber.Map{
			"Msg": "Failed to start sync job",
		})
	}
	slog.Info("ImportDirectory: directory import started", "jobID", jobID)
	c.Response().Header.Set("HX-Trigger", "jobStarted")
	c.Response().Header.Set("HX-Trigger", "queueUpdated")
	c.Response().Header.Set("HX-Trigger", "refreshImportQueueBadge")
	return c.Render("toast/toastInfo", fiber.Map{
		"Msg": "Directory import started!",
	})
}

// ProcessQueueItem handles import/cancel actions for individual queue items
func (h *Handler) ProcessQueueItem(c *fiber.Ctx) error {
	itemID := c.Params("id")
	action := c.Params("action") // "import" or "cancel"
	err := h.service.ProcessQueueItem(c.Context(), itemID, action)
	if err != nil {
		slog.Error("Failed to process queue item", "error", err, "itemID", itemID, "action", action)
		return c.Render("toast/toastErr", fiber.Map{
			"Msg": "Failed to process queue item",
		})
	}
	// Return success response that updates the UI
	actionMsg := "skipped"
	switch action {
	case "import":
		actionMsg = "imported"
	case "replace":
		actionMsg = "replaced"
	case "delete":
		actionMsg = "deleted"
	}
	c.Response().Header.Set("HX-Trigger", "queueUpdated")
	c.Response().Header.Set("HX-Trigger", "refreshImportQueueBadge")
	return c.Render("toast/toastOk", fiber.Map{
		"Msg": fmt.Sprintf("Track %s successfully", actionMsg),
	})
}

// QueueCount returns the current queue count formatted as "(X)" or empty if 0
func (h *Handler) QueueCount(c *fiber.Ctx) error {
	count := len(h.service.GetQueuedItems())
	if count == 0 {
		return c.SendString("")
	}
	return c.SendString(fmt.Sprintf("(%d)", count))
}

// ClearQueue handles clearing all items from the import queue
func (h *Handler) ClearQueue(c *fiber.Ctx) error {
	err := h.service.ClearQueue()
	if err != nil {
		slog.Error("Failed to clear queue", "error", err)
		return c.Render("toast/toastErr", fiber.Map{
			"Msg": "Failed to clear queue",
		})
	}
	c.Response().Header.Set("HX-Trigger", "queueUpdated")
	c.Response().Header.Set("HX-Trigger", "refreshImportQueueBadge")
	return c.Render("toast/toastOk", fiber.Map{
		"Msg": "Queue cleared successfully",
	})
}

// PruneDownloadPath handles pruning the download path and clearing the queue
func (h *Handler) PruneDownloadPath(c *fiber.Ctx) error {
	err := h.service.PruneDownloadPath(c.Context())
	if err != nil {
		slog.Error("Failed to prune download path", "error", err)
		return c.Render("toast/toastErr", fiber.Map{
			"Msg": "Failed to prune download path",
		})
	}
	c.Response().Header.Set("HX-Trigger", "queueUpdated")
	c.Response().Header.Set("HX-Trigger", "refreshImportQueueBadge")
	return c.Render("toast/toastOk", fiber.Map{
		"Msg": "Download path pruned and queue cleared successfully",
	})
}

// ToggleWatcher toggles the file system watcher on/off
func (h *Handler) ToggleWatcher(c *fiber.Ctx) error {
	action := c.FormValue("action")
	if action == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "action parameter required",
		})
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
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid action",
		})
	}

	if err != nil {
		slog.Error("Failed to toggle watcher", "action", action, "error", err)
		return c.Render("toast/toastErr", fiber.Map{
			"Msg": "Failed to " + action + " file watcher",
		})
	}

	c.Response().Header.Set("HX-Trigger", "watcherStatusChanged")
	return c.Render("toast/toastOk", fiber.Map{
		"Msg": msg,
	})
}

// GetWatcherStatus returns the current status of the watcher
func (h *Handler) GetWatcherStatus(c *fiber.Ctx) error {
	running := h.service.GetWatcherStatus()
	return c.Render("components/status_badge", fiber.Map{
		"Running": running,
	})
}

// GetWatcherToggleState returns the toggle input element with correct checked state
func (h *Handler) GetWatcherToggleState(c *fiber.Ctx) error {
	running := h.service.GetWatcherStatus()
	return c.Render("components/toggle", fiber.Map{
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

	return c.Render("importing/directory_form", fiber.Map{
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

	// Limit to 10 items for better performance and UX
	queueItems := make([]QueueItem, 0, 10)
	count := 0
	for _, item := range queueItemsMap {
		if count >= 10 {
			break
		}
		queueItems = append(queueItems, item)
		count++
	}

	return c.Render("importing/queue_items", fiber.Map{
		"QueueItems": queueItems,
	})
}

// GetQueueHeader renders the queue header for HTMX
func (h *Handler) GetQueueHeader(c *fiber.Ctx) error {
	slog.Debug("GetQueueHeader handler called")

	// Get queue count for display
	queueItems := h.service.GetQueuedItems()
	queueCount := len(queueItems)

	return c.Render("importing/queue_header", fiber.Map{
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
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "groupType must be 'artist' or 'album'",
		})
	}

	if action != "import" && action != "cancel" && action != "delete" && action != "replace" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "action must be 'import', 'cancel', 'delete', or 'replace'",
		})
	}

	err = h.service.ProcessQueueGroup(c.Context(), decodedGroupKey, groupType, action)
	if err != nil {
		slog.Error("Failed to process group", "error", err, "groupKey", decodedGroupKey, "groupType", groupType, "action", action)
		return c.Render("toast/toastErr", fiber.Map{
			"Msg": fmt.Sprintf("Failed to process group %s", decodedGroupKey),
		})
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

	c.Response().Header.Set("HX-Trigger", "queueUpdated")
	c.Response().Header.Set("HX-Trigger", "refreshImportQueueBadge")
	return c.Render("toast/toastOk", fiber.Map{
		"Msg": fmt.Sprintf("Group '%s' %s successfully", decodedGroupKey, actionMsg),
	})
}

// RenderGroupedQueueItems renders queue items grouped by artist or album
func (h *Handler) RenderGroupedQueueItems(c *fiber.Ctx) error {
	groupType := c.Query("type", "artist") // default to artist grouping

	var groups map[string][]QueueItem
	var templateName string

	if groupType == "album" {
		groups = h.service.GetGroupedByAlbum()
		templateName = "importing/queue_items_grouped_album"
	} else {
		groups = h.service.GetGroupedByArtist()
		templateName = "importing/queue_items_grouped_artist"
	}

	return c.Render(templateName, fiber.Map{
		"Groups":    groups,
		"GroupType": groupType,
	})
}
