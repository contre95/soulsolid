package duplicates

import (
	"fmt"
	"strconv"

	"log/slog"

	"github.com/contre95/soulsolid/src/music"
	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RenderAnalyzeDuplicatesSection(c *fiber.Ctx) error {
	data := fiber.Map{
		"Section": "analyze_duplicates",
	}
	if c.Get("HX-Request") != "true" {
		// Full page render for direct navigation/F5 (includes sidebar/navbar via main template)
		return c.Render("main", data)
	}
	return c.Render("sections/analyze_duplicates", data)
}

func (h *Handler) StartDuplicatesAnalysis(c *fiber.Ctx) error {
	params := map[string]any{}
	if exactStr := c.FormValue("fp_exact_thresh"); exactStr != "" {
		if v, err := strconv.ParseFloat(exactStr, 64); err == nil {
			params["fp_exact_thresh"] = v
		}
	}
	if fuzzyStr := c.FormValue("fp_fuzzy_thresh"); fuzzyStr != "" {
		if v, err := strconv.ParseFloat(fuzzyStr, 64); err == nil {
			params["fp_fuzzy_thresh"] = v
		}
	}
	if action := c.FormValue("default_action"); action != "" {
		params["default_action"] = action
	}
	jobID, err := h.service.StartDuplicatesAnalysis(c.Context(), params)
	if err != nil {
		if c.Get("HX-Request") == "true" {
			return c.Render("toast/toastErr", fiber.Map{"Msg": fmt.Sprintf("Failed to start: %v", err)})
		}
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	if c.Get("HX-Request") == "true" {
		return c.Render("toast/toastInfo", fiber.Map{"Msg": fmt.Sprintf("Duplicates analysis started (job %s)", jobID)})
	}
	return c.JSON(fiber.Map{"job_id": jobID})
}

func (h *Handler) RenderQueueHeader(c *fiber.Ctx) error {
	return c.Render("duplicates/queue_header", fiber.Map{
		"Count": h.service.QueueCount(),
	})
}

func (h *Handler) RenderQueueItems(c *fiber.Ctx) error {
	all := h.service.GetQueueItems()
	var queueItems []music.QueueItem
	for _, item := range all {
		if item.Type == "duplicate_fp_exact" || item.Type == "duplicate_fp_fuzzy" {
			queueItems = append(queueItems, item)
		}
	}
	return c.Render("duplicates/queue_items", fiber.Map{"QueueItems": queueItems})
}

func (h *Handler) RenderGroupedQueueItems(c *fiber.Ctx) error {
	groups := h.service.GetGroupedByFP()
	return c.Render("duplicates/queue_items_grouped_fp", fiber.Map{"Groups": groups})
}

func (h *Handler) QueueCount(c *fiber.Ctx) error {
	count := h.service.QueueCount()
	if count == 0 {
		return c.SendString("")
	}
	return c.SendString(fmt.Sprintf(" (%d)", count))
}

func (h *Handler) ClearQueue(c *fiber.Ctx) error {
	err := h.service.ClearQueue()
	if err != nil {
		slog.Error("failed to clear queue", "error", err)
		if c.Get("HX-Request") == "true" {
			return c.Render("toast/toastErr", fiber.Map{"Msg": "Failed to clear queue"})
		}
		return c.Status(500).JSON(fiber.Map{"error": "failed to clear"})
	}
	if c.Get("HX-Request") == "true" {
		return c.Render("toast/toastOk", fiber.Map{"Msg": "Queue cleared!"})
	}
	return c.JSON(fiber.Map{"status": "ok"})
}

func (h *Handler) ProcessQueueItem(c *fiber.Ctx) error {
	itemID := c.Params("id")
	action := c.Params("action")
	err := h.service.ProcessQueueItem(c.Context(), itemID, action)
	if err != nil {
		slog.Error("failed process item", "id", itemID, "action", action, "error", err)
		if c.Get("HX-Request") == "true" {
			return c.Render("toast/toastErr", fiber.Map{"Msg": fmt.Sprintf("Failed: %s", err)})
		}
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	if c.Get("HX-Request") == "true" {
		return c.Render("toast/toastOk", fiber.Map{"Msg": "Item processed!"})
	}
	return c.JSON(fiber.Map{"status": "ok"})
}

func (h *Handler) ProcessQueueGroup(c *fiber.Ctx) error {
	groupFP := c.Params("groupKey")
	action := c.Params("action")
	err := h.service.ProcessQueueGroup(c.Context(), groupFP, action)
	if err != nil {
		slog.Error("failed process group", "fp", groupFP, "action", action, "error", err)
		if c.Get("HX-Request") == "true" {
			return c.Render("toast/toastErr", fiber.Map{"Msg": fmt.Sprintf("Failed: %s", err)})
		}
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	if c.Get("HX-Request") == "true" {
		return c.Render("toast/toastOk", fiber.Map{"Msg": "Group processed!"})
	}
	return c.JSON(fiber.Map{"status": "ok"})
}

func (h *Handler) RenderCompare(c *fiber.Ctx) error {
	id := c.Params("id")
	item, err := h.service.GetQueueItem(id)
	if err != nil {
		return c.Render("toast/toastErr", fiber.Map{"Msg": "Item not found"})
	}
	data := fiber.Map{
		"Item":    item,
		"Section": "compare",
	}
	return c.Render("duplicates/compare_modal", data)
}
