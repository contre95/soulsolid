package reorganize

import (
	"log/slog"

	"github.com/contre95/soulsolid/src/features/config"
	"github.com/contre95/soulsolid/src/features/hosting/respond"
	"github.com/gofiber/fiber/v2"
)

// Handler handles HTTP requests for the reorganize feature.
type Handler struct {
	service *Service
	config  *config.Manager
}

// NewHandler creates a new reorganize handler.
func NewHandler(service *Service, config *config.Manager) *Handler {
	return &Handler{
		service: service,
		config:  config,
	}
}

// StartReorganizeAnalysis handles starting the file reorganization job
func (h *Handler) StartReorganizeAnalysis(c *fiber.Ctx) error {
	slog.Info("Starting file reorganization job from web request")

	fat32Safe := c.FormValue("fat32_safe") == "true"
	jobID, err := h.service.StartReorganizeAnalysis(c.Context(), fat32Safe)
	if err != nil {
		slog.Error("Failed to start file reorganization job", "error", err)
		return respond.Err(c, fiber.StatusInternalServerError, "Failed to start file reorganization job: "+err.Error())
	}

	slog.Info("File reorganization job started", "jobID", jobID)

	c.Set("HX-Trigger", "refreshJobList")
	return respond.Job(c, jobID, "File reorganization started successfully")
}

// RenderFilesReorganizationSection renders the file paths section page
func (h *Handler) RenderFilesReorganizationSection(c *fiber.Ctx) error {
	slog.Debug("Rendering file paths section")
	return respond.Section(c, "analyze_files", fiber.Map{
		"Title":  "File Paths",
		"Config": h.config.Get(),
	})
}
