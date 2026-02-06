package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"os/signal"

	"github.com/contre95/soulsolid/src/features/analyze"
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
	"github.com/contre95/soulsolid/src/features/playback"
	"github.com/contre95/soulsolid/src/features/playlists"
	"github.com/contre95/soulsolid/src/features/syncdap"
	"github.com/contre95/soulsolid/src/infra/database"
	"github.com/contre95/soulsolid/src/infra/files"
	"github.com/contre95/soulsolid/src/infra/fingerprint"
	"github.com/contre95/soulsolid/src/infra/providers"
	"github.com/contre95/soulsolid/src/infra/queue"
	"github.com/contre95/soulsolid/src/infra/tag"
	"github.com/contre95/soulsolid/src/infra/watcher"
	"github.com/contre95/soulsolid/src/music"
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
	libraryService := library.NewService(db, cfgManager, fileOrganizer)

	// Initialize player providers
	var playerProviders []interface {
		IsEnabled() bool
		SyncPlaylist(ctx context.Context, playlist *music.Playlist) error
		DeletePlaylist(ctx context.Context, playlistID string) error
	}
	if cfg, ok := cfgManager.Get().Players["emby"]; ok && cfg.Enabled {
		url := ""
		if cfg.URL != nil {
			url = *cfg.URL
		}
		apiKey := ""
		if cfg.APIKey != nil {
			apiKey = *cfg.APIKey
		}
		playerProviders = append(playerProviders, providers.NewEmbyProvider(url, apiKey, cfg.Enabled))
	}
	if cfg, ok := cfgManager.Get().Players["plex"]; ok && cfg.Enabled {
		url := ""
		if cfg.URL != nil {
			url = *cfg.URL
		}
		token := ""
		if cfg.Token != nil {
			token = *cfg.Token
		}
		playerProviders = append(playerProviders, providers.NewPlexProvider(url, token, cfg.Enabled))
	}

	playlistsService := playlists.NewService(db, db, cfgManager, playerProviders)
	metricsService := metrics.NewService(db, cfgManager)
	jobService := jobs.NewService(&cfgManager.Get().Jobs)

	tagReader := tag.NewTagReader()
	fingerprintReader := fingerprint.NewFingerprintService(cfgManager)
	tagWriter := tag.NewTagWriter(cfgManager.Get().Downloaders.Artwork.Embedded)

	importQueue := queue.NewInMemoryQueue()
	dirWatcher, err := watcher.NewWatcher()
	if err != nil {
		log.Fatalf("failed to create watcher: %v", err)
	}
	importingService := importing.NewService(db, tagReader, fingerprintReader, fileOrganizer, cfgManager, jobService, importQueue, dirWatcher)

	directoryImportTask := importing.NewDirectoryImportTask(importingService)
	jobService.RegisterHandler("directory_import", jobs.NewBaseTaskHandler(directoryImportTask))

	metricsTask := metrics.NewMetricsCalculationTask(db)
	jobService.RegisterHandler("calculate_metrics", jobs.NewBaseTaskHandler(metricsTask))

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

	musicbrainzProvider := providers.NewMusicBrainzProvider(cfgManager.Get().Metadata.Providers["musicbrainz"].Enabled)
	discogsSecret := ""
	if cfgManager.Get().Metadata.Providers["discogs"].Secret != nil {
		discogsSecret = *cfgManager.Get().Metadata.Providers["discogs"].Secret
	}
	discogsProvider := providers.NewDiscogsProvider(cfgManager.Get().Metadata.Providers["discogs"].Enabled, discogsSecret)
	deezerProvider := providers.NewDeezerProvider(cfgManager.Get().Metadata.Providers["deezer"].Enabled)

	lrclibProvider := providers.NewLRCLibProvider(cfgManager.Get().Lyrics.Providers["lrclib"].Enabled)

	acoustIDService := providers.NewAcoustIDService(cfgManager)
	lyricsService := lyrics.NewService(tagWriter, tagReader, db, map[string]lyrics.LyricsProvider{
		"lrclib": lrclibProvider,
	}, cfgManager)
	tagService := metadata.NewService(tagWriter, tagReader, db, map[string]metadata.MetadataProvider{
		"musicbrainz": musicbrainzProvider,
		"discogs":     discogsProvider,
		"deezer":      deezerProvider,
	}, acoustIDService, cfgManager)

	analyzeService := analyze.NewService(tagService, lyricsService, db, jobService, cfgManager, fileOrganizer) // Now using interfaces
	downloadingService := downloading.NewService(cfgManager, jobService, pluginManager, tagWriter)

	downloadTask := downloading.NewDownloadJobTask(downloadingService)
	jobService.RegisterHandler("download_track", jobs.NewBaseTaskHandler(downloadTask))
	jobService.RegisterHandler("download_album", jobs.NewBaseTaskHandler(downloadTask))
	jobService.RegisterHandler("download_artist", jobs.NewBaseTaskHandler(downloadTask))
	jobService.RegisterHandler("download_tracks", jobs.NewBaseTaskHandler(downloadTask))
	jobService.RegisterHandler("download_playlist", jobs.NewBaseTaskHandler(downloadTask))

	acoustIDTask := analyze.NewAcoustIDJobTask(analyzeService)
	jobService.RegisterHandler("analyze_acoustid", jobs.NewBaseTaskHandler(acoustIDTask))

	lyricsTask := analyze.NewLyricsJobTask(analyzeService)
	jobService.RegisterHandler("analyze_lyrics", jobs.NewBaseTaskHandler(lyricsTask))

	reorganizeTask := analyze.NewReorganizeJobTask(analyzeService)
	jobService.RegisterHandler("analyze_reorganize", jobs.NewBaseTaskHandler(reorganizeTask))

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

	playbackService := playback.NewService(db)
	server := hosting.NewServer(cfgManager, importingService, libraryService, playlistsService, syncService, downloadingService, jobService, tagService, lyricsService, metricsService, analyzeService, playbackService)
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
