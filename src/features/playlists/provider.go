package playlists

// SyncResult summarises what changed during a sync operation.
type SyncResult struct {
	PlaylistName          string
	TracksAdded           int // tracks added to the local playlist
	TracksPushed          int // tracks added to the remote playlist
	TracksUnmatched       int // remote tracks that could not be matched to a local track
	TracksRemovedFromLocal  int // tracks removed from local (deleted on remote since last sync)
	TracksRemovedFromRemote int // tracks removed from remote (deleted locally since last sync)
}

// ProviderInfo holds display information about a configured playlist provider.
type ProviderInfo struct {
	Name        string // config name (map key)
	Type        string // provider type: "emby" or "jellyfin"
	DisplayName string
	Enabled     bool
}
