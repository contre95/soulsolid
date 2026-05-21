package jobs

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/contre95/soulsolid/src/features/hosting/respond"
	"github.com/contre95/soulsolid/src/music"
	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	service *Service
}

// JobResponse is a wrapper for the Job struct to include API links
type JobResponse struct {
	*music.Job
	Links map[string]string `json:"_links"`
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RenderJobsSection renders the jobs page.
func (h *Handler) RenderJobsSection(c *fiber.Ctx) error {
	return respond.Section(c, "jobs", fiber.Map{"Title": "Jobs"})
}

func (h *Handler) HandleStartJob(c *fiber.Ctx) error {
	jobType := strings.Clone(c.Params("type"))
	name := c.Query("name", jobType)

	jobID, err := h.service.StartJob(jobType, name, nil)
	if err != nil {
		return respond.ToastErr(c, 500, fmt.Sprintf("Failed to start job: %s", err.Error()))
	}

	c.Set("HX-Trigger", "refreshActiveJobsBadge")
	return respond.ToastJob(c, jobID, fmt.Sprintf("Started %s job", jobType))
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
			if job.Status == music.JobStatusCompleted || job.Status == music.JobStatusFailed || job.Status == music.JobStatusCancelled {
				c.Set("HX-Trigger", "logsComplete")
			}
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

	if job.Status == music.JobStatusCompleted || job.Status == music.JobStatusFailed || job.Status == music.JobStatusCancelled {
		c.Set("HX-Trigger", "done")
	}

	return respond.Partial(c, "jobs/job_card_progress_bar", fiber.Map{
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

	return respond.Partial(c, "jobs/job_card", fiber.Map{
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
	return respond.ToastOk(c, "cleanup completed")
}

func (h *Handler) HandleClearFinishedJobs(c *fiber.Ctx) error {
	if err := h.service.ClearFinishedJobs(); err != nil {
		return respond.ToastErr(c, 500, err.Error())
	}
	c.Set("HX-Trigger", "refreshJobList")
	return respond.ToastOk(c, "Finished jobs cleared")
}

func (h *Handler) HandleActiveJob(c *fiber.Ctx) error {
	jobs := h.service.GetJobs()

	// Filter out completed/failed/cancelled jobs to show only active ones
	activeJobs := make([]*music.Job, 0)
	for _, job := range jobs {
		if job.Status == music.JobStatusRunning || job.Status == music.JobStatusPending {
			activeJobs = append(activeJobs, job)
		}
	}

	// Sort active jobs by date (newest first)
	sort.Slice(activeJobs, func(i, j int) bool {
		return activeJobs[i].CreatedAt.After(activeJobs[j].CreatedAt)
	})

	return respond.Partial(c, "jobs/active_list", fiber.Map{
		"Jobs": activeJobs,
	})
}

func (h *Handler) HandleFilteredJobsList(c *fiber.Ctx) error {
	jobs := h.service.GetJobs()

	// Filter jobs by type if specified
	jobTypeFilter := c.Query("prefix")
	if jobTypeFilter != "" {
		filteredJobs := make([]*music.Job, 0)
		for _, job := range jobs {
			if strings.HasPrefix(job.Type, jobTypeFilter) {
				filteredJobs = append(filteredJobs, job)
			}
		}
		jobs = filteredJobs
	}

	// Sort jobs by date (newest first)
	sort.Slice(jobs, func(i, j int) bool {
		return jobs[i].CreatedAt.After(jobs[j].CreatedAt)
	})

	return respond.Partial(c, "jobs/job_list", fiber.Map{
		"Jobs": jobs,
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
	return respond.Partial(c, "cards/latest_jobs", fiber.Map{
		"Jobs": jobs,
	})
}

func (h *Handler) HandleJobsCount(c *fiber.Ctx) error {
	jobs := h.service.GetJobs()
	filter := c.Query("filter", "active")
	count := 0

	for _, job := range jobs {
		if filter == "active" && (job.Status == music.JobStatusRunning || job.Status == music.JobStatusPending) {
			count++
		} else if filter == "all" {
			count++
		}
	}

	formatted := ""
	if count > 0 {
		formatted = fmt.Sprintf("(%d)", count)
	}
	return respond.Text(c, "jobs_count", count, formatted)
}
