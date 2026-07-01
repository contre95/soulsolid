package merge

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/contre95/soulsolid/src/music"
)

// MergeJobTask applies a metadata merge (DB + file tags) in the background.
type MergeJobTask struct {
	service *Service
}

// NewMergeJobTask creates a new merge job task.
func NewMergeJobTask(service *Service) *MergeJobTask {
	return &MergeJobTask{service: service}
}

// MetadataKeys lists the job metadata keys required for a merge job.
func (t *MergeJobTask) MetadataKeys() []string {
	return []string{"kind", "canonical", "merged"}
}

// Execute applies the merge described by the job metadata.
func (t *MergeJobTask) Execute(ctx context.Context, job *music.Job, progressUpdater func(int, string)) (map[string]any, error) {
	kindStr, _ := job.Metadata["kind"].(string)
	canonical, _ := job.Metadata["canonical"].(string)
	merged, err := toStringSlice(job.Metadata["merged"])
	if err != nil {
		return nil, fmt.Errorf("invalid merged members: %w", err)
	}
	if kindStr == "" || canonical == "" || len(merged) == 0 {
		return nil, fmt.Errorf("merge job missing kind/canonical/merged")
	}

	job.Logger.Info("starting merge", "kind", kindStr, "canonical", canonical, "merged", merged, "color", "blue")
	progressUpdater(0, "Starting merge")
	return t.service.applyMerge(ctx, job, Kind(kindStr), canonical, merged, progressUpdater)
}

// Cleanup is a no-op for merge jobs.
func (t *MergeJobTask) Cleanup(job *music.Job) error {
	slog.Debug("cleaning up merge job", "jobID", job.ID)
	return nil
}

// toStringSlice tolerates both []string (in-process job metadata) and []any (defensive).
func toStringSlice(v any) ([]string, error) {
	switch t := v.(type) {
	case nil:
		return nil, nil
	case []string:
		return t, nil
	case []any:
		out := make([]string, 0, len(t))
		for _, e := range t {
			s, ok := e.(string)
			if !ok {
				return nil, fmt.Errorf("element is not a string: %v", e)
			}
			out = append(out, s)
		}
		return out, nil
	default:
		return nil, fmt.Errorf("unexpected type %T", v)
	}
}
