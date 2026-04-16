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
	data := fiber.Map{"Section": "analyze_duplicates"}
	if c.Get("HX-Request") != "true" {
		return c.Render("main", data)
	}
	return c.Render("sections/analyze_duplicates", data)
}

func (h *Handler) StartDuplicatesAnalysis(c *fiber.Ctx) error {
	params := map[string]any{}
	if v, err := strconv.ParseFloat(c.FormValue("fp_exact_thresh"), 64); err == nil {
		params["fp_exact_thresh"] = v
	}
	if v, err := strconv.ParseFloat(c.FormValue("fp_fuzzy_thresh"), 64); err == nil {
		params["fp_fuzzy_thresh"] = v
	}
	if c.FormValue("use_acoustid") == "true" {
		params["use_acoustid"] = true
	}

	jobID, err := h.service.StartDuplicatesAnalysis(c.Context(), params)
	if err != nil {
		if c.Get("HX-Request") == "true" {
			return c.Render("toast/toastErr", fiber.Map{"Msg": fmt.Sprintf("Failed to start: %v", err)})
		}
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	if c.Get("HX-Request") == "true" {
		c.Response().Header.Set("HX-Trigger", "refreshJobList")
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
	var items []music.QueueItem
	for _, item := range all {
		if item.Type == DuplicateFPExact || item.Type == DuplicateFPFuzzy {
			items = append(items, item)
		}
	}
	return c.Render("duplicates/queue_items", fiber.Map{"QueueItems": items})
}

func (h *Handler) RenderGroupedQueueItems(c *fiber.Ctx) error {
	return c.Render("duplicates/queue_items_grouped_fp", fiber.Map{
		"Groups": h.service.GetGroupedByKey(),
	})
}

func (h *Handler) QueueCount(c *fiber.Ctx) error {
	count := h.service.QueueCount()
	if count == 0 {
		return c.SendString("")
	}
	return c.SendString(fmt.Sprintf(" (%d)", count))
}

func (h *Handler) ClearQueue(c *fiber.Ctx) error {
	if err := h.service.ClearQueue(); err != nil {
		slog.Error("failed to clear duplicates queue", "error", err)
		if c.Get("HX-Request") == "true" {
			return c.Render("toast/toastErr", fiber.Map{"Msg": "Failed to clear queue"})
		}
		return c.Status(500).JSON(fiber.Map{"error": "failed to clear"})
	}
	c.Response().Header.Set("HX-Trigger", "duplicatesQueueUpdated,refreshDuplicatesQueueBadge")
	if c.Get("HX-Request") == "true" {
		return c.Render("toast/toastOk", fiber.Map{"Msg": "Queue cleared"})
	}
	return c.JSON(fiber.Map{"status": "ok"})
}

func (h *Handler) ProcessQueueItem(c *fiber.Ctx) error {
	itemID := c.Params("id")
	action := c.Params("action")
	if err := h.service.ProcessQueueItem(c.Context(), itemID, action); err != nil {
		slog.Error("failed to process queue item", "id", itemID, "action", action, "error", err)
		if c.Get("HX-Request") == "true" {
			return c.Render("toast/toastErr", fiber.Map{"Msg": fmt.Sprintf("Failed: %s", err)})
		}
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	c.Response().Header.Set("HX-Trigger", "duplicatesQueueUpdated,refreshDuplicatesQueueBadge")
	if c.Get("HX-Request") == "true" {
		return c.Render("toast/toastOk", fiber.Map{"Msg": "Done"})
	}
	return c.JSON(fiber.Map{"status": "ok"})
}

func (h *Handler) ProcessQueueGroup(c *fiber.Ctx) error {
	groupKey := c.Params("groupKey")
	action := c.Params("action")
	if err := h.service.ProcessQueueGroup(c.Context(), groupKey, action); err != nil {
		slog.Error("failed to process queue group", "groupKey", groupKey, "action", action, "error", err)
		if c.Get("HX-Request") == "true" {
			return c.Render("toast/toastErr", fiber.Map{"Msg": fmt.Sprintf("Failed: %s", err)})
		}
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	c.Response().Header.Set("HX-Trigger", "duplicatesQueueUpdated,refreshDuplicatesQueueBadge")
	if c.Get("HX-Request") == "true" {
		return c.Render("toast/toastOk", fiber.Map{"Msg": "Group processed"})
	}
	return c.JSON(fiber.Map{"status": "ok"})
}

func (h *Handler) RenderCompare(c *fiber.Ctx) error {
	item, err := h.service.GetQueueItem(c.Params("id"))
	if err != nil {
		return c.Render("toast/toastErr", fiber.Map{"Msg": "Item not found"})
	}
	return c.Render("duplicates/compare_modal", fiber.Map{"Item": item})
}
