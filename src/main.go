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
	"github.com/contre95/soulsolid/src/features/lyrics"
	"github.com/contre95/soulsolid/src/features/metadata"
	"github.com/contre95/soulsolid/src/features/metrics"
	"github.com/contre95/soulsolid/src/features/playlists"
	"github.com/contre95/soulsolid/src/features/reorganize"
	"github.com/contre95/soulsolid/src/infra/database"
	"github.com/contre95/soulsolid/src/infra/files"
	"github.com/contre95/soulsolid/src/infra/fingerprint"
	"github.com/contre95/soulsolid/src/infra/providers"
	"github.com/contre95/soulsolid/src/infra/queue"
	"github.com/contre95/soulsolid/src/infra/tag"
	"github.com/contre95/soulsolid/src/infra/watcher"
)

func main() {
	configPath := "/config/config.yaml"
	if envPath := os.Getenv("SOULSOLID_CONFIG_PATH"); envPath != "" {
		configPath = envPath
	}
	cfgManager, err := config.NewManager(configPath)
	if err != nil {
		log.Fatalf("failed to load config manager: %v", err)
	}

	logger := logging.SetupLogger(cfgManager)
	slog.SetDefault(logger)

	pathParser := files.NewTemplatePathParser(cfgManager)
	fileOrganizer := files.NewFileOrganizer(cfgManager.Get().LibraryPath, pathParser, func() bool {
		return cfgManager.Get().Import.PathOptions.Fat32Safe
	})

	db, err := database.NewSqliteLibrary(cfgManager.Get().Database.Path)
	if err != nil {
		log.Fatalf("failed to create library: %v", err)
	}
	libraryService := library.NewService(db, cfgManager, fileOrganizer)
	playlistsService := playlists.NewService(db, db, cfgManager)
	metricsService := metrics.NewService(db, cfgManager)
	jobService := jobs.NewService(&cfgManager.Get().Jobs)

	tagReader := tag.NewTagReader()
	fingerprintReader := fingerprint.NewFingerprintService(cfgManager)
	tagWriter := tag.NewTagWriter(cfgManager.Get().Downloaders.Artwork.Embedded)

	importQueue := queue.NewInMemoryQueue()
	lyricsQueue := queue.NewInMemoryQueue()
	dirWatcher, err := watcher.NewWatcher()
	if err != nil {
		log.Fatalf("failed to create watcher: %v", err)
	}
	importingService := importing.NewService(db, tagReader, fingerprintReader, fileOrganizer, cfgManager, jobService, importQueue, dirWatcher)

	reorganizeService := reorganize.NewService(db, fileOrganizer, cfgManager, jobService)

	directoryImportTask := importing.NewDirectoryImportTask(importingService)
	jobService.RegisterHandler("directory_import", jobs.NewBaseTaskHandler(directoryImportTask))

	metricsTask := metrics.NewMetricsCalculationTask(db)
	jobService.RegisterHandler("calculate_metrics", jobs.NewBaseTaskHandler(metricsTask))

	pluginManager := downloading.NewPluginManager()
	err = pluginManager.LoadPlugins(cfgManager.Get())
	if err != nil {
		slog.Error("Failed to load plugins", "error", err)
		panic("Failed to load plugins")
	}

	musicbrainzProvider := providers.NewMusicBrainzProvider(cfgManager.Get().Metadata.Providers["musicbrainz"].Enabled)
	discogsSecret := ""
	if cfgManager.Get().Metadata.Providers["discogs"].Secret != nil {
		discogsSecret = *cfgManager.Get().Metadata.Providers["discogs"].Secret
	}
	discogsProvider := providers.NewDiscogsProvider(cfgManager.Get().Metadata.Providers["discogs"].Enabled, discogsSecret)
	deezerProvider := providers.NewDeezerProvider(cfgManager.Get().Metadata.Providers["deezer"].Enabled)

	lrclibProvider := providers.NewLRCLibProvider(cfgManager.Get().Lyrics.Providers["lrclib"].Enabled, cfgManager.Get().Lyrics.Providers["lrclib"].PreferSynced)

	acoustIDService := providers.NewAcoustIDService(cfgManager)
	lyricsService := lyrics.NewService(tagWriter, tagReader, db, map[string]lyrics.LyricsProvider{
		"lrclib": lrclibProvider,
	}, cfgManager, lyricsQueue, jobService)
	tagService := metadata.NewService(tagWriter, tagReader, db, map[string]metadata.MetadataProvider{
		"musicbrainz": musicbrainzProvider,
		"discogs":     discogsProvider,
		"deezer":      deezerProvider,
	}, acoustIDService, cfgManager, jobService)

	downloadingService := downloading.NewService(cfgManager, jobService, pluginManager, tagWriter)

	downloadTask := downloading.NewDownloadJobTask(downloadingService)
	jobService.RegisterHandler("download_track", jobs.NewBaseTaskHandler(downloadTask))
	jobService.RegisterHandler("download_album", jobs.NewBaseTaskHandler(downloadTask))
	jobService.RegisterHandler("download_artist", jobs.NewBaseTaskHandler(downloadTask))
	jobService.RegisterHandler("download_tracks", jobs.NewBaseTaskHandler(downloadTask))
	jobService.RegisterHandler("download_playlist", jobs.NewBaseTaskHandler(downloadTask))

	acoustIDTask := metadata.NewAcoustIDJobTask(tagService)
	jobService.RegisterHandler("analyze_acoustid", jobs.NewBaseTaskHandler(acoustIDTask))

	lyricsTask := lyrics.NewLyricsJobTask(lyricsService)
	jobService.RegisterHandler("analyze_lyrics", jobs.NewBaseTaskHandler(lyricsTask))

	reorganizeTask := reorganize.NewReorganizeJobTask(reorganizeService)
	jobService.RegisterHandler("analyze_reorganize", jobs.NewBaseTaskHandler(reorganizeTask))

	var telegramBot *hosting.TelegramBot
	if cfgManager.Get().Telegram.Enabled {
		var err error
		telegramBot, err = hosting.NewTelegramBot(cfgManager, libraryService, jobService, importingService)
		if err != nil {
			slog.Error("Failed to initialize Telegram bot", "error", err)
		} else {
			go telegramBot.Start()
			slog.Info("Telegram bot started")
		}
	}

	server := hosting.NewServer(cfgManager, importingService, libraryService, playlistsService, downloadingService, jobService, tagService, lyricsService, metricsService, reorganizeService)
	slog.Info("Starting server", "port", cfgManager.Get().Server.Port)
	if err := server.Start(); err != nil {
		slog.Error("server stopped: %v", "error", err)
	}
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	serverErr := make(chan error, 1)
	go func() {
		slog.Info("Starting server", "port", cfgManager.Get().Server.Port)
		if err := server.Start(); err != nil {
			serverErr <- err
		}
	}()

	select {
	case <-quit:
		slog.Info("Shutting down server...")
	case err := <-serverErr:
		slog.Error("server stopped", "error", err)
	}

	if telegramBot != nil {
		telegramBot.Stop()
		slog.Info("Telegram bot stopped")
	}

	importingService.StopWatcher()
	if err := server.Shutdown(); err != nil {
		log.Fatalf("failed to shutdown server: %v", err)
	}
	slog.Info("Server gracefully shut down.")
}
