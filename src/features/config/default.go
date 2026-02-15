package config

var defaultConfig = Config{
	LibraryPath:  "./music",
	DownloadPath: "./downloads",
	Telegram: Telegram{
		Enabled:      false,
		Token:        "",                                   // Can be obtained with https://t.me/BotFather
		AllowedUsers: []string{"<your_telegram_username>"}, // No @
		BotHandle:    "@<YourTelegramUserBot>",             // With @
	},
	Logger: Logger{
		Enabled:   true,
		Level:     "info",
		Format:    "text",
		HTMXDebug: false,
	},
	Downloaders: Downloaders{
		Plugins: []PluginConfig{},
		Artwork: Artwork{
			Embedded: EmbeddedArtwork{
				Enabled: true,
				Size:    1000,
				Quality: 85,
			},
		},
	},
	Server: Server{
		PrintRoutes: false,
		Port:        3535,
	},
	Database: Database{
		Path: "./library.db",
	},
	Import: Import{
		Move:             false,
		AlwaysQueue:      false,
		Duplicates:       "queue",
		AutoStartWatcher: false,
		PathOptions: Paths{
			Compilations:    "%asciify{$genre}/%asciify{$format}/%asciify{$albumartist}/%asciify{$album} (%if{$original_year,$original_year,$year})/%asciify{$track $title}",
			AlbumSoundtrack: "%asciify{$genre}/%asciify{$format}/%asciify{$albumartist}/%asciify{$album} [OST] (%if{$original_year,$original_year,$year})/%asciify{$track $title}",
			AlbumSingle:     "%asciify{$genre}/%asciify{$format}/%asciify{$albumartist}/%asciify{$album} [Single] (%if{$original_year,$original_year,$year})/%asciify{$track $title}",
			AlbumEP:         "%asciify{$genre}/%asciify{$format}/%asciify{$albumartist}/%asciify{$album} [EP] (%if{$original_year,$original_year,$year})/%asciify{$track $title}",
			DefaultPath:     "%asciify{$genre}/%asciify{$format}/%asciify{$albumartist}/%asciify{$album} (%if{$original_year,$original_year,$year})/%asciify{$track $title}",
		},
	},
	Metadata: Metadata{
		Providers: map[string]Provider{
			"deezer": {
				Enabled: true,
			},
			"discogs": {
				Enabled: false,
				Secret:  nil,
			},
			"musicbrainz": {
				Enabled: true,
			},
			"acoustid": {
				Enabled: false,
				Secret:  nil,
			},
		},
	},
	Lyrics: Lyrics{
		Providers: map[string]LyricsProvider{
			"lrclib": {
				Enabled:      true,
				PreferSynced: false,
			},
		},
	},
	Sync: Sync{
		Enabled: false,
		Devices: []Device{},
	},
	Jobs: Jobs{
		Log:     true,
		LogPath: "./logs/jobs",
		Webhooks: WebhookConfig{
			Enabled:  false,
			JobTypes: []string{},
			Command:  "",
		},
	},
}
