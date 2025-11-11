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
	"github.com/contre95/soulsolid/src/infra/files"
	"github.com/contre95/soulsolid/src/infra/metadata"
	"github.com/contre95/soulsolid/src/infra/queue"
	"github.com/contre95/soulsolid/src/infra/tag"
	"github.com/contre95/soulsolid/src/infra/watcher"
)

func main() {
	cfgManager, err := config.Load("config.yaml")
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	logger := logging.SetupLogger(cfgManager)
	slog.SetDefault(logger)

	pathParser := files.NewTemplatePathParser(cfgManager)
	fileOrganizer := files.NewFileOrganizer(cfgManager.Get().LibraryPath, pathParser)

	db, err := database.NewSqliteLibrary(cfgManager.Get().Database.Path)
	if err != nil {
		log.Fatalf("failed to create library: %v", err)
	}
	libraryService := library.NewService(db, cfgManager)
	jobService := jobs.NewService(&cfgManager.Get().Jobs)

	tagReader := tag.NewTagReader()
	fingerprintReader := chroma.NewFingerprintService()

	importQueue := queue.NewInMemoryQueue()
	eventChan := make(chan importing.FileEvent, 10)
	watcherInstance, err := watcher.NewWatcher(eventChan)
	if err != nil {
		log.Fatalf("failed to create watcher: %v", err)
	}
	importingService := importing.NewService(db, tagReader, fingerprintReader, fileOrganizer, cfgManager, jobService, importQueue, watcherInstance)

	directoryImportTask := importing.NewDirectoryImportTask(importingService)
	jobService.RegisterHandler("directory_import", jobs.NewBaseTaskHandler(directoryImportTask))

	syncService := syncdap.NewService(cfgManager, jobService)
	if cfgManager.Get().Sync.Enabled {
		syncService.Start()
		defer syncService.Stop()
	}

	syncTask := syncdap.NewSyncDapTask(syncService)
	jobService.RegisterHandler("dap_sync", jobs.NewBaseTaskHandler(syncTask))

	pluginManager := downloading.NewPluginManager()
	err = pluginManager.LoadPlugins(cfgManager.Get())
	if err != nil {
		slog.Error("Failed to load plugins", "error", err)
		panic("Failed to load plugins")
	}

	tagWriter := tag.NewTagWriter(cfgManager.Get().Downloaders.Artwork.Embedded)

	musicbrainzProvider := metadata.NewMusicBrainzProvider(cfgManager.Get().Metadata.Providers["musicbrainz"].Enabled)
	discogsAPIKey := ""
	if cfgManager.Get().Metadata.Providers["discogs"].APIKey != nil {
		discogsAPIKey = *cfgManager.Get().Metadata.Providers["discogs"].APIKey
	}
	discogsProvider := metadata.NewDiscogsProvider(cfgManager.Get().Metadata.Providers["discogs"].Enabled, discogsAPIKey)
	deezerProvider := metadata.NewDeezerProvider(cfgManager.Get().Metadata.Providers["deezer"].Enabled)

	tagService := tagging.NewService(tagWriter, tagReader, db, []tagging.MetadataProvider{musicbrainzProvider, discogsProvider, deezerProvider}, fingerprintReader, cfgManager)
	downloadingService := downloading.NewService(cfgManager, jobService, pluginManager, tagWriter)

	downloadTask := downloading.NewDownloadJobTask(downloadingService)
	jobService.RegisterHandler("download_track", jobs.NewBaseTaskHandler(downloadTask))
	jobService.RegisterHandler("download_album", jobs.NewBaseTaskHandler(downloadTask))
	jobService.RegisterHandler("download_tracks", jobs.NewBaseTaskHandler(downloadTask))

	var telegramBot *hosting.TelegramBot
	if cfgManager.Get().Telegram.Enabled {
		var err error
		telegramBot, err = hosting.NewTelegramBot(cfgManager, libraryService, jobService, syncService, importingService)
		if err != nil {
			slog.Error("Failed to initialize Telegram bot", "error", err)
		} else {
			go telegramBot.Start()
			slog.Info("Telegram bot started")
		}
	}

	server := hosting.NewServer(cfgManager, importingService, libraryService, syncService, downloadingService, jobService, tagService)
	slog.Info("Starting server", "port", cfgManager.Get().Server.Port)
	if err := server.Start(); err != nil {
		slog.Error("server stopped: %v", "error", err)
	}
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit
	slog.Info("Shutting down server...")

	if telegramBot != nil {
		telegramBot.Stop()
		slog.Info("Telegram bot stopped")
	}

	if err := server.Shutdown(); err != nil {
		log.Fatalf("failed to shutdown server: %v", err)
	}
	slog.Info("Server gracefully shut down.")
}
