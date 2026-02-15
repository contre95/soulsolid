package config

// Config holds the application configuration.
type Config struct {
	LibraryPath  string      `yaml:"libraryPath" validate:"required"`
	DownloadPath string      `yaml:"downloadPath" validate:"required"`
	Telegram     Telegram    `yaml:"telegram"`
	Logger       Logger      `yaml:"logger"`
	Downloaders  Downloaders `yaml:"downloaders"`
	Server       Server      `yaml:"server"`
	Database     Database    `yaml:"database"`
	Import       Import      `yaml:"import"`
	Metadata     Metadata    `yaml:"metadata"`
	Lyrics       Lyrics      `yaml:"lyrics"`
	Sync         Sync        `yaml:"sync"`
	Jobs         Jobs        `yaml:"jobs"`
}
type Jobs struct {
	Log      bool          `yaml:"log"`
	LogPath  string        `yaml:"log_path"`
	Webhooks WebhookConfig `yaml:"webhooks"`
}

type WebhookConfig struct {
	Enabled  bool     `yaml:"enabled"`
	JobTypes []string `yaml:"job_types"`
	Command  string   `yaml:"command"`
}

type Import struct {
	Move                 bool   `yaml:"move"` // If not copies
	AlwaysQueue          bool   `yaml:"always_queue"`
	Duplicates           string `yaml:"duplicates"` // "replace", "skip", "queue"
	PathOptions          Paths  `yaml:"paths"`
	AutoStartWatcher     bool   `yaml:"auto_start_watcher"`
	AllowMissingMetadata bool   `yaml:"allow_missing_metadata"`
}

type Paths struct {
	Compilations    string `yaml:"compilations"`
	AlbumSoundtrack string `yaml:"album:soundtrack"`
	AlbumSingle     string `yaml:"album:single"`
	AlbumEP         string `yaml:"album:ep"`
	DefaultPath     string `yaml:"default_path"`
}

// Database holds the configuration for the database
type Database struct {
	Path string `yaml:"path" validate:"required"`
}

// Server hold the configuration for the Fiber server Config
type Server struct {
	PrintRoutes bool   `yaml:"show_routes"`
	Port        uint32 `yaml:"port"`
}

// Logger holds the configuration for the app logging
type Logger struct {
	Enabled   bool   `yaml:"enabled"`
	Level     string `yaml:"level"`
	Format    string `yaml:"format"`
	HTMXDebug bool   `yaml:"htmx_debug"`
}

type Telegram struct {
	Enabled      bool     `yaml:"enabled"`
	Token        string   `yaml:"token"`
	AllowedUsers []string `yaml:"allowedUsers"`
	BotHandle    string   `yaml:"bot_handle"`
}

// Downloaders holds the configuration for the various downloaders.
type Downloaders struct {
	Plugins []PluginConfig `yaml:"plugins"`
	Artwork Artwork        `yaml:"artwork"`
}

// Metadata holds the configuration for metadata tagging providers
type Metadata struct {
	Providers map[string]Provider `yaml:"providers"`
}

// Provider holds configuration for individual tagging providers
type Provider struct {
	Enabled bool    `yaml:"enabled"`
	Secret  *string `yaml:"secret,omitempty"`
}

// Lyrics holds the configuration for lyrics providers
type Lyrics struct {
	Providers map[string]LyricsProvider `yaml:"providers"`
}

// Provider holds configuration for individual lyric providers
type LyricsProvider struct {
	Enabled      bool `yaml:"enabled"`
	PreferSynced bool `yaml:"prefer_synced,omitempty"`
}

// Sync holds configuration for device synchronization
type Sync struct {
	Enabled bool     `yaml:"enabled"`
	Devices []Device `yaml:"devices"`
}

// Device holds configuration for individual sync devices
type Device struct {
	UUID     string `yaml:"uuid"`
	Name     string `yaml:"name"`
	SyncPath string `yaml:"sync_path"`
}

// Artwork holds configuration for artwork handling
type Artwork struct {
	Embedded EmbeddedArtwork `yaml:"embedded"`
}

// EmbeddedArtwork holds configuration for embedded artwork
type EmbeddedArtwork struct {
	Enabled bool `yaml:"enabled"`
	Size    int  `yaml:"size"`
	Quality int  `yaml:"quality"`
}

// PluginConfig holds configuration for a plugin downloader
type PluginConfig struct {
	Name   string         `yaml:"name"`
	Path   string         `yaml:"path"`
	URL    string         `yaml:"url,omitempty"`
	Icon   string         `yaml:"icon,omitempty"`
	Config map[string]any `yaml:"config"`
}
