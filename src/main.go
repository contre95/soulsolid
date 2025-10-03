package main

import (
	"log"
	"log/slog"
	"os"
	"os/signal"

	"github.com/contre95/soulsolid/src/features/config"
	"github.com/contre95/soulsolid/src/features/downloading"
	"github.com/contre95/soulsolid/src/features/hosting"
	"github.com/contre95/soulsolid/src/features/importing"
	"github.com/contre95/soulsolid/src/features/jobs"
	"github.com/contre95/soulsolid/src/features/library"
	"github.com/contre95/soulsolid/src/features/logging"
	"github.com/contre95/soulsolid/src/features/syncdap"
	"github.com/contre95/soulsolid/src/features/tagging"
	"github.com/contre95/soulsolid/src/infra/chroma"
	"github.com/contre95/soulsolid/src/infra/database"
	"github.com/contre95/soulsolid/src/infra/download/dummy"
	"github.com/contre95/soulsolid/src/infra/files"
	"github.com/contre95/soulsolid/src/infra/queue"
	"github.com/contre95/soulsolid/src/infra/tag"
)

func main() {
	// Load configuration
	cfgManager, err := config.Load("config.yaml")
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// Setup default logger with slog
	logger := logging.SetupLogger(cfgManager)
	slog.SetDefault(logger)

	pathParser := files.NewTemplatePathParser(cfgManager)
	fileOrganizer := files.NewFileOrganizer(cfgManager.Get().LibraryPath, pathParser)

	// Create the database library
	db, err := database.NewSqliteLibrary(cfgManager.Get().Database.Path)
	if err != nil {
		log.Fatalf("failed to create library: %v", err)
	}
	libraryService := library.NewService(db, cfgManager)

	// Create the job service
	jobService := jobs.NewService(&cfgManager.Get().Jobs)

	// Create the importing service
	tagReader := tag.NewTagReader()
	fingerprintReader := chroma.NewFingerprintService()

	importQueue := queue.NewInMemoryQueue()
	importingService := importing.NewService(db, tagReader, fingerprintReader, fileOrganizer, cfgManager, jobService, importQueue)

	directoryImportTask := importing.NewDirectoryImportTask(importingService)
	jobService.RegisterHandler("directory_import", jobs.NewBaseTaskHandler(directoryImportTask))

	// Create the syncdap service
	syncService := syncdap.NewService(cfgManager, jobService)
	if cfgManager.Get().Sync.Enabled {
		syncService.Start()
		defer syncService.Stop()
	}

	// Register syncdap Task
	syncTask := syncdap.NewSyncDapTask(syncService)
	jobService.RegisterHandler("dap_sync", jobs.NewBaseTaskHandler(syncTask))

	// Create the plugin manager and load plugins
	pluginManager := downloading.NewPluginManager()
	if cfgManager.Get().Demo {
		dummyDownloader := dummy.NewDummyDownloader()
		pluginManager.AddDownloader("dummy", dummyDownloader)
		slog.Info("Loaded built-in dummy downloader (demo mode)", "name", dummyDownloader.Name())
	}
	err = pluginManager.LoadPlugins(cfgManager.Get())
	if err != nil {
		slog.Error("Failed to load plugins", "error", err)
		panic("Failed to load plugins")
	}

	// Create the tag writer
	tagWriter := tag.NewTagWriter()

	// Create metadata providers
	musicbrainzProvider := tag.NewMusicBrainzProvider(cfgManager.Get().Tag.Providers["musicbrainz"].Enabled)
	discogsProvider := tag.NewDiscogsProvider(cfgManager.Get().Tag.Providers["discogs"].Enabled)

	// Create the tag service
	tagService := tagging.NewService(tagWriter, tagReader, db, []tagging.MetadataProvider{musicbrainzProvider, discogsProvider}, fingerprintReader, cfgManager)

	// Create the downloading service
	downloadingService := downloading.NewService(cfgManager, jobService, pluginManager, tagWriter)

	// Register download Tasks
	downloadTask := downloading.NewDownloadJobTask(downloadingService)
	jobService.RegisterHandler("download_track", jobs.NewBaseTaskHandler(downloadTask))
	jobService.RegisterHandler("download_album", jobs.NewBaseTaskHandler(downloadTask))

	// Create and start the Telegram bot if enabled
	var telegramBot *hosting.TelegramBot
	if cfgManager.Get().Telegram.Enabled {
		var err error
		telegramBot, err = hosting.NewTelegramBot(cfgManager, libraryService, jobService, syncService, downloadingService, importingService)
		if err != nil {
			slog.Error("Failed to initialize Telegram bot", "error", err)
		} else {
			go telegramBot.Start()
			slog.Info("Telegram bot started")
		}
	}

	// Create and start the HTTP server
	server := hosting.NewServer(cfgManager, importingService, libraryService, syncService, downloadingService, jobService, tagService)
	if err := server.Start(); err != nil {
		slog.Error("server stopped: %v", "error", err)
	}
	slog.Info("Server started. Press Ctrl+C to shut down.", "port", cfgManager.Get().Server.Port)
	// Wait for a shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit
	slog.Info("Shutting down server...")

	// Shutdown the Telegram bot
	if telegramBot != nil {
		telegramBot.Stop()
		slog.Info("Telegram bot stopped")
	}

	// Shutdown the server
	if err := server.Shutdown(); err != nil {
		log.Fatalf("failed to shutdown server: %v", err)
	}
	slog.Info("Server gracefully shut down.")
}
