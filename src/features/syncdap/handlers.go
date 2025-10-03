package syncdap

import (
	"log/slog"

	"soulsolid/src/features/jobs"
	"github.com/gofiber/fiber/v2"
)

// Handler handles HTTP requests for device syncing
type Handler struct {
	service    *Service
	jobService jobs.JobService
}

// NewHandler creates a new sync handler
func NewHandler(service *Service, jobService jobs.JobService) *Handler {
	return &Handler{service: service, jobService: jobService}
}

// GetDeviceStatusCard returns a card with the current sync status of all devices
func (h *Handler) GetDeviceStatusCard(c *fiber.Ctx) error {
	slog.Debug("GetDeviceStatusCard handler called")

	// Check if sync is enabled in config
	if !h.service.configManager.Get().Sync.Enabled {
		return c.SendString("") // Return empty if disabled
	}

	status := h.service.GetStatus()
	return c.Render("cards/device_status_card", fiber.Map{
		"Status": status,
	})
}

// GetSyncStatus returns the current sync status of all devices
func (h *Handler) GetSyncStatus(c *fiber.Ctx) error {
	slog.Debug("GetSyncStatus handler called")
	status := h.service.GetStatus()
	return c.Render("sync/sync_status", fiber.Map{
		"Status": status,
	})
}

// GetDeviceStatus returns the status of a specific device
func (h *Handler) GetDeviceStatus(c *fiber.Ctx) error {
	uuid := c.Params("uuid")
	slog.Debug("GetDeviceStatus handler called", "uuid", uuid)
	status, exists := h.service.GetDeviceStatus(uuid)
	if !exists {
		return c.Status(fiber.StatusNotFound).SendString("Device not found")
	}
	return c.JSON(status)
}

// TriggerSync manually triggers a sync for a device
func (h *Handler) TriggerSync(c *fiber.Ctx) error {
	uuid := c.Params("uuid")
	slog.Debug("TriggerSync handler called", "uuid", uuid)
	jobID, err := h.service.StartSync(uuid)
	if err != nil {
		slog.Error("Failed to start sync job", "error", err)
		return c.Render("toast/toastErr", fiber.Map{
			"Msg": "Failed to start sync job",
		})
	}
	slog.Info("TriggerSync: sync job started", "jobID", jobID)
	c.Response().Header.Set("HX-Trigger", "jobStarted")
	return c.Render("toast/toastInfo", fiber.Map{
		"Msg": "Sync job started!",
	})
}

// CancelSync cancels an ongoing sync operation
func (h *Handler) CancelSync(c *fiber.Ctx) error {
	uuid := c.Params("uuid")
	slog.Debug("CancelSync handler called", "uuid", uuid)
	err := h.service.CancelSync(uuid)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	return c.JSON(fiber.Map{
		"message": "Sync cancelled",
		"uuid":    uuid,
	})
}
