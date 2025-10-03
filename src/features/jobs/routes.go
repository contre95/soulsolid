package jobs

import "github.com/gofiber/fiber/v2"

func RegisterRoutes(app *fiber.App, service *Service) {
	handler := NewHandler(service)
	jobs := app.Group("/jobs")
	ui := app.Group("/ui/jobs")
	ui.Get("/active", handler.HandleActiveJob)
	ui.Get("/list", handler.HandleAllJobsList)
	ui.Get("/latest", handler.HandleLatestJobs)

	// ui.Post("/cleanup", handler.HandleCleanupJobs)
	jobs.Get("/", handler.HandleJobList)
	jobs.Post("/start/:type", handler.HandleStartJob)
	jobs.Get("/:id", handler.HandleJobStatus)
	jobs.Get("/:id/progress", handler.HandleJobProgress)
	jobs.Get("/:id/logs", handler.HandleJobLogs)
	jobs.Post("/:id/cancel", handler.HandleCancelJob)
}
