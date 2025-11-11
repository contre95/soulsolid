package importing

import (
	"fmt"
	"log/slog"

	"github.com/contre95/soulsolid/src/features/jobs"
	"github.com/gofiber/fiber/v2"
)

// Handler is the handler for the organizing feature.
type Handler struct {
	service    *Service
	jobService *jobs.Service
}

// NewHandler creates a new handler for the organizing feature.
func NewHandler(service *Service, jobService *jobs.Service) *Handler {
	return &Handler{service: service, jobService: jobService}
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

// QueueCount returns the current queue count formatted as "(X)"
func (h *Handler) QueueCount(c *fiber.Ctx) error {
	return c.SendString(fmt.Sprintf("(%d)", len(h.service.GetQueuedItems())))
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

	if action == "start" {
		err = h.service.StartWatcher()
		msg = "File watcher started successfully"
	} else if action == "stop" {
		err = h.service.StopWatcher()
		msg = "File watcher stopped successfully"
	} else {
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
	if running {
		return c.SendString(`<span class="text-xs text-green-600 dark:text-green-400 font-medium">Active</span>`)
	}
	return c.SendString(`<span class="text-xs text-gray-500 dark:text-gray-400">Inactive</span>`)
}

// GetWatcherToggleState returns the toggle input element with correct checked state
func (h *Handler) GetWatcherToggleState(c *fiber.Ctx) error {
	running := h.service.GetWatcherStatus()
	checked := ""
	if running {
		checked = "checked"
	}

	html := fmt.Sprintf(`<input id="watcher-toggle" type="checkbox" class="peer appearance-none w-11 h-5 bg-slate-100 rounded-full checked:bg-slate-800 cursor-pointer transition-colors duration-300" %s
           hx-post="/import/watcher/toggle" hx-target="#toast-container" hx-swap="innerHTML" hx-trigger="change"
           hx-vals="js:{action: event.target.checked ? 'start' : 'stop'}">
<label for="watcher-toggle" class="absolute top-0 left-0 w-5 h-5 bg-white rounded-full border border-slate-300 shadow-sm transition-transform duration-300 peer-checked:translate-x-6 peer-checked:border-slate-800 cursor-pointer">
</label>`, checked)

	return c.SendString(html)
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
