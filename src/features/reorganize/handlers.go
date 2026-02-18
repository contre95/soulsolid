package reorganize

import (
	"log/slog"

	"github.com/contre95/soulsolid/src/features/config"
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

	jobID, err := h.service.StartReorganizeAnalysis(c.Context())
	if err != nil {
		slog.Error("Failed to start file reorganization job", "error", err)
		return c.Status(fiber.StatusInternalServerError).Render("toast/toastError", fiber.Map{
			"Msg": "Failed to start file reorganization job: " + err.Error(),
		})
	}

	slog.Info("File reorganization job started", "jobID", jobID)

	if c.Get("HX-Request") == "true" {
		return c.Render("toast/toastOk", fiber.Map{
			"Msg": "File reorganization started successfully",
		})
	}

	return c.Redirect("/ui/analyze/files")
}

// RenderFilesReorganizationSection renders the file paths section page
func (h *Handler) RenderFilesReorganizationSection(c *fiber.Ctx) error {
	slog.Debug("Rendering file paths section")

	data := fiber.Map{
		"Title":  "File Paths",
		"Config": h.config.Get(),
	}

	if c.Get("HX-Request") != "true" {
		data["Section"] = "analyze_files"
		return c.Render("main", data)
	}

	return c.Render("sections/analyze_files", data)
}
