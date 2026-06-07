package jobs

import "github.com/gofiber/fiber/v2"

func RegisterRoutes(app *fiber.App, service *Service) {
	handler := NewHandler(service)

	app.Get("/jobs", handler.RenderJobsSection)

	jobs := app.Group("/jobs")
	jobs.Get("/active", handler.HandleActiveJob)
	jobs.Get("/list", handler.HandleFilteredJobsList)
	jobs.Get("/latest", handler.HandleLatestJobs)
	jobs.Post("/clear-finished", handler.HandleClearFinishedJobs)
	jobs.Get("/count", handler.HandleJobsCount)
	jobs.Get("/all", handler.HandleJobList)
	jobs.Post("/start/:type", handler.HandleStartJob)
	jobs.Get("/:id", handler.HandleJobStatus)
	jobs.Get("/:id/progress", handler.HandleJobProgress)
	jobs.Get("/:id/logs", handler.HandleJobLogs)
	jobs.Post("/:id/cancel", handler.HandleCancelJob)
}
