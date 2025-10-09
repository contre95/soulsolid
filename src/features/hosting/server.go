package hosting

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/contre95/soulsolid/src/features/config"
	"github.com/contre95/soulsolid/src/features/downloading"
	"github.com/contre95/soulsolid/src/features/importing"
	"github.com/contre95/soulsolid/src/features/jobs"
	"github.com/contre95/soulsolid/src/features/library"
	"github.com/contre95/soulsolid/src/features/syncdap"
	"github.com/contre95/soulsolid/src/features/tagging"
	"github.com/contre95/soulsolid/src/features/ui"
	"github.com/contre95/soulsolid/src/music"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/html/v2"
)

// Server is the HTTP server for the application.
type Server struct {
	app  *fiber.App
	port uint32
}

// NewServer creates a new HTTP server.
func NewServer(cfg *config.Manager, importingService *importing.Service, libraryService *library.Service, syncService *syncdap.Service, downloadingService *downloading.Service, jobService *jobs.Service, tagService *tagging.Service) *Server {
	engine := html.New(cfg.Get().Server.ViewsPath, ".html")
	engine.Debug(cfg.Get().Logger.Level == "debug")
	// Add custom template functions
	engine.AddFunc("isDebug", func() bool {
		return cfg.Get().Logger.HTMXDebug
	})
	engine.AddFunc("add", func(a, b int) int {
		return a + b
	})

	engine.AddFunc("duration", func(seconds int) string {
		if seconds == 0 {
			return "0:00"
		}
		minutes := seconds / 60
		remainingSeconds := seconds % 60
		return fmt.Sprintf("%d:%02d", minutes, remainingSeconds)
	})
	engine.AddFunc("formatDuration", func(seconds int) string {
		if seconds == 0 {
			return "0 min"
		}
		hours := seconds / 3600
		minutes := (seconds % 3600) / 60
		if hours > 0 {
			return fmt.Sprintf("%d hr %d min", hours, minutes)
		}
		return fmt.Sprintf("%d min", minutes)
	})
	engine.AddFunc("totalDuration", func(tracks []music.Track) string {
		totalSeconds := 0
		for _, track := range tracks {
			totalSeconds += track.Metadata.Duration
		}
		if totalSeconds == 0 {
			return "0 min"
		}
		hours := totalSeconds / 3600
		minutes := (totalSeconds % 3600) / 60
		if hours > 0 {
			return fmt.Sprintf("%d hr %d min", hours, minutes)
		}
		return fmt.Sprintf("%d min", minutes)
	})

	engine.AddFunc("capitalize", func(s string) string {
		return strings.Title(strings.ToLower(s))
	})

	app := fiber.New(fiber.Config{
		Views: engine,
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			slog.Error("Internal Server Error", "error", err)
			return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
		},
		AppName:               "Soulsolid",
		DisableStartupMessage: true,
		EnablePrintRoutes:     cfg.Get().Server.PrintRoutes,
		BodyLimit:             1000 * 1024 * 1024, // 100MB limit for file uploads
	})

	// Add middleware
	app.Use(HTMXMiddleware())
	app.Use(LogAllRequestsMiddleware())

	app.Static("/", "./public")
	app.Static("/node_modules", "./node_modules")
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	uiHandler := ui.NewHandler(cfg)

	importing.RegisterRoutes(app, importingService)
	library.RegisterRoutes(app, libraryService)
	ui.RegisterRoutes(app, uiHandler)
	config.RegisterRoutes(app, cfg)
	jobs.RegisterRoutes(app, jobService)
	if cfg.Get().Sync.Enabled {
		syncdap.RegisterRoutes(app, syncService, jobService)
	}
	downloading.RegisterRoutes(app, downloadingService, jobService)
	tagging.RegisterRoutes(app, tagService)

	return &Server{app: app, port: cfg.Get().Server.Port}
}

// Start starts the HTTP server.
func (s *Server) Start() error {
	return s.app.Listen(":" + fmt.Sprint(s.port))
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown() error {
	return s.app.Shutdown()
}
