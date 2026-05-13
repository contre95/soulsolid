package playlists

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/contre95/soulsolid/src/features/config"
	"github.com/contre95/soulsolid/src/music"
)

// Service is the domain service for the playlists feature.
type Service struct {
	playlistRepo  music.PlaylistRepository
	library       music.Library
	configManager *config.Manager
	providers     map[string]music.PlaylistProvider
	jobService    music.JobService
}

// NewService creates a new playlists service.
func NewService(playlistRepo music.PlaylistRepository, lib music.Library, cfgManager *config.Manager, providers map[string]music.PlaylistProvider, jobService music.JobService) *Service {
	if providers == nil {
		providers = map[string]music.PlaylistProvider{}
	}
	return &Service{
		playlistRepo:  playlistRepo,
		library:       lib,
		configManager: cfgManager,
		providers:     providers,
		jobService:    jobService,
	}
}

// StartPushJob enqueues a job that pushes a local playlist to a remote provider.
func (s *Service) StartPushJob(playlistID, providerName string) (string, error) {
	return s.jobService.StartJob("playlist_push", fmt.Sprintf("Push playlist to %s", providerName), map[string]any{
		"operation":   "push",
		"provider":    providerName,
		"playlist_id": playlistID,
	})
}

// StartPullJob enqueues a job that pulls all playlists from a remote provider.
func (s *Service) StartPullJob(providerName string) (string, error) {
	return s.jobService.StartJob("playlist_pull", fmt.Sprintf("Pull playlists from %s", providerName), map[string]any{
		"operation": "pull",
		"provider":  providerName,
	})
}

// StartSyncJob enqueues a job that two-way syncs a local playlist with a remote provider.
func (s *Service) StartSyncJob(playlistID, providerName string) (string, error) {
	return s.jobService.StartJob("playlist_sync", fmt.Sprintf("Sync playlist with %s", providerName), map[string]any{
		"operation":   "sync",
		"provider":    providerName,
		"playlist_id": playlistID,
	})
}

// CreatePlaylist creates a new playlist.
func (s *Service) CreatePlaylist(ctx context.Context, name, description string) (*music.Playlist, error) {
	slog.Debug("CreatePlaylist service called", "name", name)

	playlist := &music.Playlist{
		ID:           music.GeneratePlaylistID(),
		Name:         name,
		Description:  description,
		Tracks:       []*music.Track{},
		CreatedDate:  time.Now(),
		ModifiedDate: time.Now(),
	}

	if err := playlist.Validate(); err != nil {
		slog.Error("CreatePlaylist validation failed", "error", err)
		return nil, err
	}

	err := s.playlistRepo.Create(ctx, playlist)
	if err != nil {
		slog.Error("CreatePlaylist failed", "name", name, "error", err)
		return nil, err
	}

	slog.Debug("CreatePlaylist completed", "id", playlist.ID, "name", name)
	return playlist, nil
}

// GetPlaylist gets a playlist by ID.
func (s *Service) GetPlaylist(ctx context.Context, id string) (*music.Playlist, error) {
	slog.Debug("GetPlaylist service called", "id", id)

	playlist, err := s.playlistRepo.GetByID(ctx, id)
	if err != nil {
		slog.Error("GetPlaylist failed", "id", id, "error", err)
		return nil, err
	}

	slog.Debug("GetPlaylist completed", "id", id)
	return playlist, nil
}

// GetAllPlaylists gets all playlists.
func (s *Service) GetAllPlaylists(ctx context.Context) ([]*music.Playlist, error) {
	slog.Debug("GetAllPlaylists service called")

	playlists, err := s.playlistRepo.GetAll(ctx)
	if err != nil {
		slog.Error("GetAllPlaylists failed", "error", err)
		return nil, err
	}

	slog.Debug("GetAllPlaylists completed", "count", len(playlists))
	return playlists, nil
}

// UpdatePlaylist updates a playlist.
func (s *Service) UpdatePlaylist(ctx context.Context, playlist *music.Playlist) error {
	slog.Debug("UpdatePlaylist service called", "id", playlist.ID, "name", playlist.Name)

	playlist.ModifiedDate = time.Now()

	if err := playlist.Validate(); err != nil {
		slog.Error("UpdatePlaylist validation failed", "error", err)
		return err
	}

	err := s.playlistRepo.Update(ctx, playlist)
	if err != nil {
		slog.Error("UpdatePlaylist failed", "id", playlist.ID, "error", err)
		return err
	}

	slog.Debug("UpdatePlaylist completed", "id", playlist.ID)
	return nil
}

// DeletePlaylist deletes a playlist.
func (s *Service) DeletePlaylist(ctx context.Context, id string) error {
	slog.Debug("DeletePlaylist service called", "id", id)

	err := s.playlistRepo.Delete(ctx, id)
	if err != nil {
		slog.Error("DeletePlaylist failed", "id", id, "error", err)
		return err
	}

	slog.Debug("DeletePlaylist completed", "id", id)
	return nil
}

// AddItemToPlaylist adds tracks, artists, or albums to a playlist.
func (s *Service) AddItemToPlaylist(ctx context.Context, playlistID, itemType, itemID string) error {
	slog.Debug("AddItemToPlaylist service called", "playlistID", playlistID, "itemType", itemType, "itemID", itemID)

	// Verify playlist exists
	playlist, err := s.playlistRepo.GetByID(ctx, playlistID)
	if err != nil {
		slog.Error("AddItemToPlaylist: failed to get playlist", "playlistID", playlistID, "error", err)
		return fmt.Errorf("failed to get playlist %s: %w", playlistID, err)
	}
	if playlist == nil {
		slog.Error("AddItemToPlaylist: playlist not found", "playlistID", playlistID)
		return fmt.Errorf("playlist not found: %s", playlistID)
	}

	// Get track IDs to add based on item type
	var trackIDs []string
	switch itemType {
	case "track":
		// Verify track exists
		track, err := s.library.GetTrack(ctx, itemID)
		if err != nil {
			slog.Error("AddItemToPlaylist: failed to get track", "trackID", itemID, "error", err)
			return fmt.Errorf("failed to get track %s: %w", itemID, err)
		}
		if track == nil {
			slog.Error("AddItemToPlaylist: track not found in database", "trackID", itemID)
			return fmt.Errorf("track not found: %s", itemID)
		}
		trackIDs = []string{itemID}

	case "artist":
		// Get all tracks by this artist
		tracks, err := s.library.GetTracksFilteredPaginated(ctx, 10000, 0, &music.TrackFilter{ArtistIDs: []string{itemID}})
		if err != nil {
			slog.Error("AddItemToPlaylist: failed to get artist tracks", "artistID", itemID, "error", err)
			return fmt.Errorf("failed to get artist tracks: %w", err)
		}
		for _, track := range tracks {
			trackIDs = append(trackIDs, track.ID)
		}
		if len(trackIDs) == 0 {
			return fmt.Errorf("no tracks found for artist: %s", itemID)
		}

	case "album":
		// Get all tracks from this album
		tracks, err := s.library.GetTracksFilteredPaginated(ctx, 10000, 0, &music.TrackFilter{AlbumIDs: []string{itemID}})
		if err != nil {
			slog.Error("AddItemToPlaylist: failed to get album tracks", "albumID", itemID, "error", err)
			return fmt.Errorf("failed to get album tracks: %w", err)
		}
		for _, track := range tracks {
			trackIDs = append(trackIDs, track.ID)
		}
		if len(trackIDs) == 0 {
			return fmt.Errorf("no tracks found for album: %s", itemID)
		}

	default:
		return fmt.Errorf("unsupported item type: %s", itemType)
	}

	// Add tracks to playlist (skip duplicates)
	addedCount := 0
	for _, trackID := range trackIDs {
		err = s.playlistRepo.AddTrackToPlaylist(ctx, playlistID, trackID)
		if err != nil {
			// If track already exists, that's ok - just skip it
			if !strings.Contains(err.Error(), "already exists") {
				slog.Error("AddItemToPlaylist: failed to add track", "playlistID", playlistID, "trackID", trackID, "error", err)
				return fmt.Errorf("failed to add track %s: %w", trackID, err)
			}
		} else {
			addedCount++
		}
	}

	slog.Info("AddItemToPlaylist completed successfully", "playlistID", playlistID, "itemType", itemType, "itemID", itemID, "tracksAdded", addedCount)
	return nil
}

// RemoveTrackFromPlaylist removes a track from a playlist.
func (s *Service) RemoveTrackFromPlaylist(ctx context.Context, playlistID, trackID string) error {
	slog.Debug("RemoveTrackFromPlaylist service called", "playlistID", playlistID, "trackID", trackID)

	err := s.playlistRepo.RemoveTrackFromPlaylist(ctx, playlistID, trackID)
	if err != nil {
		slog.Error("RemoveTrackFromPlaylist failed", "playlistID", playlistID, "trackID", trackID, "error", err)
		return err
	}

	slog.Debug("RemoveTrackFromPlaylist completed", "playlistID", playlistID, "trackID", trackID)
	return nil
}

// GetPlaylistTracks gets all tracks for a playlist.
func (s *Service) GetPlaylistTracks(ctx context.Context, playlistID string) ([]*music.Track, error) {
	slog.Debug("GetPlaylistTracks service called", "playlistID", playlistID)

	tracks, err := s.playlistRepo.GetTracksForPlaylist(ctx, playlistID)
	if err != nil {
		slog.Error("GetPlaylistTracks failed", "playlistID", playlistID, "error", err)
		return nil, err
	}

	slog.Debug("GetPlaylistTracks completed", "playlistID", playlistID, "count", len(tracks))
	return tracks, nil
}

// ExportM3U exports a playlist to an M3U file.
func (s *Service) ExportM3U(ctx context.Context, playlistID, filePath string) error {
	slog.Debug("ExportM3U service called", "playlistID", playlistID, "filePath", filePath)

	// Get playlist tracks
	tracks, err := s.playlistRepo.GetTracksForPlaylist(ctx, playlistID)
	if err != nil {
		slog.Error("ExportM3U: failed to get playlist tracks", "playlistID", playlistID, "error", err)
		return err
	}

	// Create output file
	file, err := os.Create(filePath)
	if err != nil {
		slog.Error("ExportM3U: failed to create file", "filePath", filePath, "error", err)
		return fmt.Errorf("failed to create M3U file: %w", err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)

	// Write M3U header
	_, err = writer.WriteString("#EXTM3U\n")
	if err != nil {
		return err
	}

	// Write each track
	for _, track := range tracks {
		// Write extended M3U info
		duration := track.Metadata.Duration
		artists := make([]string, len(track.Artists))
		for i, ar := range track.Artists {
			if ar.Artist != nil {
				artists[i] = ar.Artist.Name
			}
		}
		artistStr := strings.Join(artists, ", ")

		_, err = writer.WriteString(fmt.Sprintf("#EXTINF:%d,%s - %s\n", duration, artistStr, track.Title))
		if err != nil {
			return err
		}

		// Write file path
		_, err = writer.WriteString(track.Path + "\n")
		if err != nil {
			return err
		}
	}

	err = writer.Flush()
	if err != nil {
		slog.Error("ExportM3U: failed to flush writer", "error", err)
		return err
	}

	slog.Debug("ExportM3U completed", "playlistID", playlistID, "filePath", filePath, "tracksExported", len(tracks))
	return nil
}

// ListProviders returns info about all configured playlist providers.
func (s *Service) ListProviders() []ProviderInfo {
	result := make([]ProviderInfo, 0, len(s.providers))
	for key, p := range s.providers {
		result = append(result, ProviderInfo{
			Name:        key,
			Type:        p.Name(),
			DisplayName: p.DisplayName(),
			Enabled:     p.IsEnabled(),
		})
	}
	return result
}

// PullFromProvider fetches all playlists from the named provider and creates or
// updates the matching local playlists. Track matching uses file path first,
// then title+artist as a fallback.
func (s *Service) PullFromProvider(ctx context.Context, providerName string) ([]*music.Playlist, error) {
	slog.Debug("PullFromProvider service called", "provider", providerName)

	provider, ok := s.providers[providerName]
	if !ok {
		return nil, fmt.Errorf("playlist provider %q not found", providerName)
	}

	remotePlaylists, err := provider.ListPlaylists(ctx)
	if err != nil {
		return nil, fmt.Errorf("list playlists from %s: %w", providerName, err)
	}

	// Load all local playlists once to avoid repeated GetAll calls.
	allLocal, err := s.playlistRepo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("get local playlists: %w", err)
	}

	var pulled []*music.Playlist
	for _, rp := range remotePlaylists {
		full, err := provider.GetPlaylist(ctx, rp.RemoteID)
		if err != nil {
			slog.Warn("PullFromProvider: failed to get remote playlist tracks", "playlist", rp.Name, "error", err)
			continue
		}

		// Find local playlist: check cached link first, then fall back to name match.
		var local *music.Playlist
		for _, lp := range allLocal {
			links, err := s.playlistRepo.GetProviderLinks(ctx, lp.ID)
			if err == nil {
				for _, link := range links {
					if link.ProviderName == providerName && link.RemoteID == rp.RemoteID {
						local, _ = s.playlistRepo.GetByID(ctx, lp.ID)
						break
					}
				}
			}
			if local != nil {
				break
			}
		}
		if local == nil {
			for _, lp := range allLocal {
				if lp.Name == rp.Name {
					local, _ = s.playlistRepo.GetByID(ctx, lp.ID)
					break
				}
			}
		}
		if local == nil {
			local, err = s.CreatePlaylist(ctx, rp.Name, rp.Description)
			if err != nil {
				slog.Warn("PullFromProvider: failed to create local playlist", "name", rp.Name, "error", err)
				continue
			}
		}

		// Persist the remote association.
		_ = s.playlistRepo.SetProviderLink(ctx, local.ID, providerName, provider.Name(), rp.RemoteID)

		added := 0
		for _, rt := range full.Tracks {
			localTrack := s.resolveRemoteTrack(ctx, provider, rt)
			if localTrack == nil {
				continue
			}
			if local.ContainsTrack(localTrack.ID) {
				continue
			}
			if err := s.playlistRepo.AddTrackToPlaylist(ctx, local.ID, localTrack.ID); err != nil {
				slog.Warn("PullFromProvider: failed to add track to local playlist", "trackID", localTrack.ID, "error", err)
				continue
			}
			local.Tracks = append(local.Tracks, localTrack)
			added++
		}

		slog.Info("PullFromProvider: pulled playlist", "name", rp.Name, "tracksAdded", added)
		pulled = append(pulled, local)
	}

	slog.Debug("PullFromProvider completed", "provider", providerName, "playlists", len(pulled))
	return pulled, nil
}

// PushToProvider pushes a local playlist to the named provider, creating the
// remote playlist if it does not already exist (matched by name).
// Returns (pushed, unmatched, error).
func (s *Service) PushToProvider(ctx context.Context, playlistID, providerName string) (int, int, error) {
	slog.Debug("PushToProvider service called", "playlistID", playlistID, "provider", providerName)

	provider, ok := s.providers[providerName]
	if !ok {
		return 0, 0, fmt.Errorf("playlist provider %q not found", providerName)
	}

	local, err := s.playlistRepo.GetByID(ctx, playlistID)
	if err != nil {
		return 0, 0, fmt.Errorf("get local playlist: %w", err)
	}
	if local == nil {
		return 0, 0, fmt.Errorf("local playlist %s not found", playlistID)
	}
	local.Tracks, err = s.playlistRepo.GetTracksForPlaylist(ctx, playlistID)
	if err != nil {
		return 0, 0, fmt.Errorf("get playlist tracks: %w", err)
	}

	// Find or create the remote playlist by name.
	remoteID, err := s.findOrCreateRemotePlaylist(ctx, provider, providerName, local.ID, local.Name, local.Description)
	if err != nil {
		return 0, 0, fmt.Errorf("find or create remote playlist: %w", err)
	}

	// Get current remote tracks to skip duplicates.
	remoteFull, err := provider.GetPlaylist(ctx, remoteID)
	if err != nil {
		return 0, 0, fmt.Errorf("get remote playlist tracks: %w", err)
	}
	remoteTrackIDs := map[string]struct{}{}
	for _, rt := range remoteFull.Tracks {
		remoteTrackIDs[rt.RemoteID] = struct{}{}
	}

	var toAdd []string
	unmatched := 0
	for _, lt := range local.Tracks {
		rt, err := s.resolveLocalTrack(ctx, provider, lt)
		if err != nil || rt == nil {
			slog.Warn("PushToProvider: could not resolve local track to remote", "track", lt.Title, "error", err)
			unmatched++
			continue
		}
		if _, exists := remoteTrackIDs[rt.RemoteID]; exists {
			continue
		}
		toAdd = append(toAdd, rt.RemoteID)
	}

	if len(toAdd) > 0 {
		if err := provider.AddTracksToPlaylist(ctx, remoteID, toAdd); err != nil {
			return 0, unmatched, fmt.Errorf("add tracks to remote playlist: %w", err)
		}
	}

	slog.Info("PushToProvider completed", "playlist", local.Name, "provider", providerName, "tracksPushed", len(toAdd), "unmatched", unmatched)
	return len(toAdd), unmatched, nil
}

// SyncWithProvider performs a two-way sync between a local playlist and its
// counterpart on the named provider. Remote-only tracks are added locally;
// local-only tracks are pushed to the remote.
func (s *Service) SyncWithProvider(ctx context.Context, playlistID, providerName string) (*SyncResult, error) {
	slog.Debug("SyncWithProvider service called", "playlistID", playlistID, "provider", providerName)

	provider, ok := s.providers[providerName]
	if !ok {
		return nil, fmt.Errorf("playlist provider %q not found", providerName)
	}

	local, err := s.playlistRepo.GetByID(ctx, playlistID)
	if err != nil {
		return nil, fmt.Errorf("get local playlist: %w", err)
	}
	if local == nil {
		return nil, fmt.Errorf("local playlist %s not found", playlistID)
	}
	local.Tracks, err = s.playlistRepo.GetTracksForPlaylist(ctx, playlistID)
	if err != nil {
		return nil, fmt.Errorf("get playlist tracks: %w", err)
	}

	result := &SyncResult{PlaylistName: local.Name}

	// Find or create the remote playlist.
	remoteID, err := s.findOrCreateRemotePlaylist(ctx, provider, providerName, local.ID, local.Name, local.Description)
	if err != nil {
		return nil, fmt.Errorf("find or create remote playlist: %w", err)
	}

	remoteFull, err := provider.GetPlaylist(ctx, remoteID)
	if err != nil {
		return nil, fmt.Errorf("get remote playlist: %w", err)
	}

	// Index local tracks by path for fast lookup.
	localByPath := map[string]*music.Track{}
	for _, lt := range local.Tracks {
		if lt.Path != "" {
			localByPath[lt.Path] = lt
		}
	}

	// Build set of remote track IDs for deduplication in the push direction.
	remoteIDSet := map[string]struct{}{}
	for _, rt := range remoteFull.Tracks {
		remoteIDSet[rt.RemoteID] = struct{}{}
	}

	// Pull: add remote-only tracks to local playlist.
	for _, rt := range remoteFull.Tracks {
		localTrack := s.resolveRemoteTrack(ctx, provider, rt)
		if localTrack == nil {
			result.TracksUnmatched++
			continue
		}
		if local.ContainsTrack(localTrack.ID) {
			continue
		}
		if err := s.playlistRepo.AddTrackToPlaylist(ctx, local.ID, localTrack.ID); err != nil {
			slog.Warn("SyncWithProvider: failed to add track locally", "track", localTrack.Title, "error", err)
			continue
		}
		local.Tracks = append(local.Tracks, localTrack)
		result.TracksAdded++
	}

	// Push: add local-only tracks to remote playlist.
	// Resolve each local track to its remote ID and skip those already present —
	// path-based comparison is unreliable when mount points differ.
	var toAdd []string
	for _, lt := range local.Tracks {
		rt, err := s.resolveLocalTrack(ctx, provider, lt)
		if err != nil || rt == nil {
			continue
		}
		if _, exists := remoteIDSet[rt.RemoteID]; exists {
			continue
		}
		toAdd = append(toAdd, rt.RemoteID)
	}
	if len(toAdd) > 0 {
		if err := provider.AddTracksToPlaylist(ctx, remoteID, toAdd); err != nil {
			slog.Warn("SyncWithProvider: failed to push tracks to remote", "error", err)
		} else {
			result.TracksPushed = len(toAdd)
		}
	}

	slog.Info("SyncWithProvider completed", "playlist", local.Name, "provider", providerName,
		"added", result.TracksAdded, "pushed", result.TracksPushed, "unmatched", result.TracksUnmatched)
	return result, nil
}

// resolveRemoteTrack maps a RemoteTrack to a local music.Track using path then metadata.
func (s *Service) resolveRemoteTrack(ctx context.Context, _ music.PlaylistProvider, rt music.RemoteTrack) *music.Track {
	if rt.Path != "" {
		if t, err := s.library.FindTrackByPath(ctx, rt.Path); err == nil && t != nil {
			return t
		}
	}
	if rt.Title != "" && rt.Artist != "" {
		if t, err := s.library.FindTrackByMetadata(ctx, rt.Title, rt.Artist, rt.Album); err == nil && t != nil {
			return t
		}
	}
	return nil
}

// resolveLocalTrack maps a local music.Track to a RemoteTrack using path then metadata.
func (s *Service) resolveLocalTrack(ctx context.Context, provider music.PlaylistProvider, lt *music.Track) (*music.RemoteTrack, error) {
	if lt.Path != "" {
		rt, err := provider.FindTrackByPath(ctx, lt.Path)
		if err != nil {
			slog.Debug("resolveLocalTrack: path lookup error", "track", lt.Title, "path", lt.Path, "error", err)
		} else if rt != nil {
			return rt, nil
		} else {
			slog.Debug("resolveLocalTrack: path lookup returned no match, trying metadata", "track", lt.Title, "path", lt.Path)
		}
	}
	artistName := ""
	if len(lt.Artists) > 0 && lt.Artists[0].Artist != nil {
		artistName = lt.Artists[0].Artist.Name
	}
	if lt.Title != "" && artistName != "" {
		rt, err := provider.FindTrackByMetadata(ctx, lt.Title, artistName)
		if err != nil {
			slog.Debug("resolveLocalTrack: metadata lookup error", "track", lt.Title, "artist", artistName, "error", err)
			return nil, err
		}
		if rt == nil {
			slog.Warn("resolveLocalTrack: track not found on provider", "track", lt.Title, "artist", artistName, "path", lt.Path)
		}
		return rt, nil
	}
	slog.Warn("resolveLocalTrack: insufficient metadata to search", "track", lt.Title, "path", lt.Path)
	return nil, nil
}

// findOrCreateRemotePlaylist returns the remote playlist ID for the given local playlist,
// using the cached link if available to skip the ListPlaylists round-trip.
func (s *Service) findOrCreateRemotePlaylist(ctx context.Context, provider music.PlaylistProvider, providerName, playlistID, name, description string) (string, error) {
	// Check cached link first.
	if links, err := s.playlistRepo.GetProviderLinks(ctx, playlistID); err == nil {
		for _, link := range links {
			if link.ProviderName == providerName {
				return link.RemoteID, nil
			}
		}
	}
	// Fall back to name search on the remote.
	remotes, err := provider.ListPlaylists(ctx)
	if err != nil {
		return "", err
	}
	for _, rp := range remotes {
		if rp.Name == name {
			_ = s.playlistRepo.SetProviderLink(ctx, playlistID, providerName, provider.Name(), rp.RemoteID)
			return rp.RemoteID, nil
		}
	}
	remoteID, err := provider.CreatePlaylist(ctx, name, description)
	if err != nil {
		return "", fmt.Errorf("create playlist: %w", err)
	}
	_ = s.playlistRepo.SetProviderLink(ctx, playlistID, providerName, provider.Name(), remoteID)
	return remoteID, nil
}

// GetPlaylistsContainingTrack gets all playlists that contain a specific track.
func (s *Service) GetPlaylistsContainingTrack(ctx context.Context, trackID string) ([]*music.Playlist, error) {
	slog.Debug("GetPlaylistsContainingTrack service called", "trackID", trackID)

	allPlaylists, err := s.playlistRepo.GetAll(ctx)
	if err != nil {
		slog.Error("GetPlaylistsContainingTrack: failed to get all playlists", "error", err)
		return nil, err
	}

	var containingPlaylists []*music.Playlist
	for _, playlist := range allPlaylists {
		fullPlaylist, err := s.playlistRepo.GetByID(ctx, playlist.ID)
		if err != nil {
			slog.Warn("GetPlaylistsContainingTrack: failed to get full playlist", "playlistID", playlist.ID, "error", err)
			continue
		}
		if fullPlaylist.ContainsTrack(trackID) {
			containingPlaylists = append(containingPlaylists, fullPlaylist)
		}
	}

	slog.Debug("GetPlaylistsContainingTrack completed", "trackID", trackID, "count", len(containingPlaylists))
	return containingPlaylists, nil
}
