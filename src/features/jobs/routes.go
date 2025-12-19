package jobs

import "github.com/gofiber/fiber/v2"

func RegisterRoutes(app *fiber.App, service *Service) {
	handler := NewHandler(service)
	jobs := app.Group("/jobs")
	ui := app.Group("/ui")
	ui.Get("/jobs", handler.RenderJobsSection)
	uiJobs := app.Group("/ui/jobs")
	uiJobs.Get("/active", handler.HandleActiveJob)
	uiJobs.Get("/list", handler.HandleFilteredJobsList)
	uiJobs.Get("/latest", handler.HandleLatestJobs)
	uiJobs.Post("/clear-finished", handler.HandleClearFinishedJobs)
	uiJobs.Get("/count", handler.HandleJobsCount)

	// ui.Post("/cleanup", handler.HandleCleanupJobs)
	jobs.Get("/", handler.HandleJobList)
	jobs.Post("/start/:type", handler.HandleStartJob)
	jobs.Get("/:id", handler.HandleJobStatus)
	jobs.Get("/:id/progress", handler.HandleJobProgress)
	jobs.Get("/:id/logs", handler.HandleJobLogs)
	jobs.Post("/:id/cancel", handler.HandleCancelJob)
}
