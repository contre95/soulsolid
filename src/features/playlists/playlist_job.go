package playlists

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/contre95/soulsolid/src/music"
)

// PlaylistJobTask handles push/pull/sync job execution.
// The operation is selected via job.Metadata["operation"]: "push", "pull", or "sync".
type PlaylistJobTask struct {
	service *Service
}

func NewPlaylistJobTask(service *Service) *PlaylistJobTask {
	return &PlaylistJobTask{service: service}
}

func (t *PlaylistJobTask) MetadataKeys() []string {
	return []string{"operation", "provider"}
}

func (t *PlaylistJobTask) Execute(ctx context.Context, job *music.Job, progressUpdater func(int, string)) (map[string]any, error) {
	operation, _ := job.Metadata["operation"].(string)
	providerName, _ := job.Metadata["provider"].(string)
	playlistID, _ := job.Metadata["playlist_id"].(string)

	job.Logger.Info("Starting playlist job", "operation", operation, "provider", providerName)
	progressUpdater(0, fmt.Sprintf("Starting %s with %s…", operation, providerName))

	switch operation {
	case "push":
		if playlistID == "" {
			return nil, fmt.Errorf("playlist_id is required for push")
		}
		pushed, unmatched, err := t.service.PushToProvider(ctx, playlistID, providerName, job.Logger)
		if err != nil {
			return nil, err
		}
		msg := fmt.Sprintf("Pushed %d tracks to %s", pushed, providerName)
		if unmatched > 0 {
			msg += fmt.Sprintf(" (%d could not be matched)", unmatched)
		}
		progressUpdater(100, msg)
		return map[string]any{"pushed": pushed, "unmatched": unmatched, "msg": msg}, nil

	case "pull":
		pulled, err := t.service.PullFromProvider(ctx, providerName, job.Logger)
		if err != nil {
			return nil, err
		}
		msg := fmt.Sprintf("Pulled %d playlists from %s", len(pulled), providerName)
		progressUpdater(100, msg)
		return map[string]any{"playlists": len(pulled), "msg": msg}, nil

	case "sync":
		if playlistID == "" {
			return nil, fmt.Errorf("playlist_id is required for sync")
		}
		result, err := t.service.SyncWithProvider(ctx, playlistID, providerName, job.Logger)
		if err != nil {
			return nil, err
		}
		msg := fmt.Sprintf("Sync with %s: +%d local, +%d remote", providerName, result.TracksAdded, result.TracksPushed)
		if result.TracksRemovedFromLocal+result.TracksRemovedFromRemote > 0 {
			msg += fmt.Sprintf(", -%d local, -%d remote", result.TracksRemovedFromLocal, result.TracksRemovedFromRemote)
		}
		if result.TracksUnmatched > 0 {
			msg += fmt.Sprintf(", %d unmatched", result.TracksUnmatched)
		}
		progressUpdater(100, msg)
		return map[string]any{
			"tracksAdded":           result.TracksAdded,
			"tracksPushed":          result.TracksPushed,
			"tracksUnmatched":       result.TracksUnmatched,
			"tracksRemovedFromLocal":  result.TracksRemovedFromLocal,
			"tracksRemovedFromRemote": result.TracksRemovedFromRemote,
			"msg":                   msg,
		}, nil

	case "sync_all":
		if err := t.service.AutoSyncAll(ctx, job.Logger); err != nil {
			return nil, err
		}
		msg := "All playlists synced"
		progressUpdater(100, msg)
		return map[string]any{"msg": msg}, nil

	default:
		return nil, fmt.Errorf("unknown playlist job operation: %s", operation)
	}
}

func (t *PlaylistJobTask) Cleanup(job *music.Job) error {
	slog.Debug("Cleaning up playlist job", "jobID", job.ID)
	return nil
}
