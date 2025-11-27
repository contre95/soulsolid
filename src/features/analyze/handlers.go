package analyze

import (
	"log/slog"

	"github.com/gofiber/fiber/v2"
)

// Handler handles HTTP requests for analysis operations
type Handler struct {
	service *Service
}

// NewHandler creates a new analyze handler
func NewHandler(service *Service) *Handler {
	return &Handler{
		service: service,
	}
}

// StartAcoustIDAnalysis handles starting the AcoustID analysis job
func (h *Handler) StartAcoustIDAnalysis(c *fiber.Ctx) error {
	slog.Info("Starting AcoustID analysis via HTTP request")

	jobID, err := h.service.StartAcoustIDAnalysis(c.Context())
	if err != nil {
		slog.Error("Failed to start AcoustID analysis", "error", err)
		return c.Render("toast/toastErr", fiber.Map{
			"Msg": "Failed to start AcoustID analysis: " + err.Error(),
		})
	}

	slog.Info("AcoustID analysis job started successfully", "jobID", jobID)

	// // Trigger HTMX to refresh the job list
	c.Set("HX-Trigger", "refreshJobList")

	if c.Get("HX-Request") == "true" {
		return c.Render("toast/toastOk", fiber.Map{
			"Msg": "AcoustID analysis started successfully",
		})
	}

	return c.Redirect("/ui/analyze")
}

// StartLyricsAnalysis handles starting the lyrics analysis job
func (h *Handler) StartLyricsAnalysis(c *fiber.Ctx) error {
	slog.Info("Starting lyrics analysis via HTTP request")

	jobID, err := h.service.StartLyricsAnalysis(c.Context())
	if err != nil {
		slog.Error("Failed to start lyrics analysis", "error", err)
		return c.Render("toast/toastErr", fiber.Map{
			"Msg": "Failed to start lyrics analysis: " + err.Error(),
		})
	}

	slog.Info("Lyrics analysis job started successfully", "jobID", jobID)

	// Trigger HTMX to refresh the job list
	c.Set("HX-Trigger", "refreshJobList")

	if c.Get("HX-Request") == "true" {
		return c.Render("toast/toastOk", fiber.Map{
			"Msg": "Lyrics analysis started successfully",
		})
	}

	return c.Redirect("/ui/analyze")
}

// RenderAnalyzeSection renders the analyze section page
func (h *Handler) RenderAnalyzeSection(c *fiber.Ctx) error {
	slog.Debug("Rendering analyze section")

	data := fiber.Map{
		"Title": "Analyze",
	}

	if c.Get("HX-Request") != "true" {
		data["Section"] = "analyze"
		return c.Render("main", data)
	}

	return c.Render("sections/analyze", data)
}
