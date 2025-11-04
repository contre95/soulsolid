package jobs

import (
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	service *Service
}

// JobResponse is a wrapper for the Job struct to include API links
type JobResponse struct {
	*Job
	Links map[string]string `json:"_links"`
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) HandleStartJob(c *fiber.Ctx) error {
	jobType := c.Params("type")
	name := c.Query("name", jobType)

	jobID, err := h.service.StartJob(jobType, name, nil)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"job_id": jobID})
}

func (h *Handler) HandleJobStatus(c *fiber.Ctx) error {
	jobID := c.Params("id")
	job, exists := h.service.GetJob(jobID)
	if !exists {
		return c.Status(404).SendString("Job not found")
	}

	baseURL := c.BaseURL()
	response := &JobResponse{
		Job: job,
		Links: map[string]string{
			"self":     fmt.Sprintf("%s/jobs/%s", baseURL, job.ID),
			"progress": fmt.Sprintf("%s/jobs/%s/progress", baseURL, job.ID),
			"logs":     fmt.Sprintf("%s/jobs/%s/logs", baseURL, job.ID),
		},
	}

	return c.JSON(response)
}

func (h *Handler) HandleJobLogs(c *fiber.Ctx) error {
	jobID := c.Params("id")
	job, exists := h.service.GetJob(jobID)
	if !exists {
		return c.Status(404).SendString("Job not found")
	}

	if job.LogPath == "" {
		return c.SendString("No logs for this job.")
	}

	logContent, err := os.ReadFile(job.LogPath)
	if err != nil {
		return c.Status(500).SendString("Failed to read log file.")
	}

	// Check if color parameter is set
	color := c.Query("color") == "true"

	if color {
		// Check if this is a direct request (not HTMX)
		if c.Get("HX-Request") != "true" {
			// Return full HTML page for direct access (full screen)
			return c.Render("jobs/job_logs_fullscreen", fiber.Map{
				"Job": job,
			})
		} else {
			// Return just the colored content for HTMX requests
			coloredContent := ParseAndColorLogContent(string(logContent))
			c.Set("Content-Type", "text/html")
			return c.SendString(coloredContent)
		}
	} else {
		// Return raw text
		c.Set("Content-Type", "text/plain")
		return c.SendString(string(logContent))
	}
}

func (h *Handler) HandleJobProgress(c *fiber.Ctx) error {
	jobID := c.Params("id")
	job, exists := h.service.GetJob(jobID)
	if !exists {
		return c.Status(404).SendString("Job not found.")
	}

	if job.Status == JobStatusCompleted || job.Status == JobStatusFailed || job.Status == JobStatusCancelled {
		c.Set("HX-Trigger", "done")
	}

	return c.Render("jobs/job_card_progress_bar", fiber.Map{
		"ID":        job.ID,
		"Name":      job.Name,
		"Type":      job.Type,
		"Status":    job.Status,
		"Progress":  job.Progress,
		"Message":   job.Message,
		"Error":     job.Error,
		"CreatedAt": job.CreatedAt,
		"LogPath":   job.LogPath,
	})
}

func (h *Handler) HandleJobList(c *fiber.Ctx) error {
	jobs := h.service.GetJobs()
	baseURL := c.BaseURL()
	responses := make([]*JobResponse, len(jobs))
	for i, job := range jobs {
		responses[i] = &JobResponse{
			Job: job,
			Links: map[string]string{
				"self":     fmt.Sprintf("%s/jobs/%s", baseURL, job.ID),
				"progress": fmt.Sprintf("%s/jobs/%s/progress", baseURL, job.ID),
				"logs":     fmt.Sprintf("%s/jobs/%s/logs", baseURL, job.ID),
			},
		}
	}
	return c.JSON(responses)
}

func (h *Handler) HandleCancelJob(c *fiber.Ctx) error {
	jobID := c.Params("id")

	err := h.service.CancelJob(jobID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	// Get the updated job to render the card
	job, exists := h.service.GetJob(jobID)
	if !exists {
		return c.Status(404).SendString("Job not found")
	}

	return c.Render("jobs/job_card", fiber.Map{
		"ID":        job.ID,
		"Name":      job.Name,
		"Type":      job.Type,
		"Status":    job.Status,
		"Progress":  job.Progress,
		"Message":   job.Message,
		"Error":     job.Error,
		"CreatedAt": job.CreatedAt,
		"LogPath":   job.LogPath,
	})
}

func (h *Handler) HandleCleanupJobs(c *fiber.Ctx) error {
	h.service.CleanupOldJobs(24 * time.Hour)
	return c.JSON(fiber.Map{"status": "cleanup completed"})
}

func (h *Handler) HandleClearFinishedJobs(c *fiber.Ctx) error {
	err := h.service.ClearFinishedJobs()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	if c.Get("HX-Request") == "true" {
		c.Set("HX-Trigger", "refreshJobList")
		return c.Render("toast/toastOk", fiber.Map{
			"Msg": "Finished jobs cleared",
		})
	}

	return c.JSON(fiber.Map{"status": "finished jobs cleared"})
}

func (h *Handler) HandleActiveJob(c *fiber.Ctx) error {
	jobs := h.service.GetJobs()

	// Filter out completed/failed/cancelled jobs to show only active ones
	activeJobs := make([]*Job, 0)
	for _, job := range jobs {
		if job.Status == JobStatusRunning || job.Status == JobStatusPending {
			activeJobs = append(activeJobs, job)
		}
	}

	return c.Render("jobs/active_list", fiber.Map{
		"Jobs": activeJobs,
	})
}

func (h *Handler) HandleAllJobsList(c *fiber.Ctx) error {
	jobs := h.service.GetJobs()

	// Filter to show only completed and failed jobs
	finishedJobs := make([]*Job, 0)
	for _, job := range jobs {
		if job.Status == JobStatusCompleted || job.Status == JobStatusFailed {
			finishedJobs = append(finishedJobs, job)
		}
	}

	return c.Render("jobs/job_list", fiber.Map{
		"Jobs": finishedJobs,
	})
}

func (h *Handler) HandleLatestJobs(c *fiber.Ctx) error {
	jobs := h.service.GetJobs()
	sort.Slice(jobs, func(i, j int) bool {
		return jobs[i].CreatedAt.After(jobs[j].CreatedAt)
	})
	if len(jobs) > 5 {
		jobs = jobs[:5]
	}
	return c.Render("cards/latest_jobs", fiber.Map{
		"Jobs": jobs,
	})
}
