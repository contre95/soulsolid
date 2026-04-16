package duplicates

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/contre95/soulsolid/src/features/config"
	"github.com/contre95/soulsolid/src/infra/fingerprint"
	"github.com/contre95/soulsolid/src/music"
)

// TagWriter interface for writing tags.
type TagWriter interface {
	WriteFileTags(ctx context.Context, path string, track *music.Track) error
}

// Service provides duplicates analysis functionality.
type Service struct {
	tagWriter  TagWriter
	library    music.Library
	config     *config.Manager
	queue      music.Queue
	jobService music.JobService
	similarity *fingerprint.SimilarityService
}

// NewService creates a new duplicates service.
func NewService(tagWriter TagWriter, library music.Library, config *config.Manager, queue music.Queue, jobService music.JobService, similarity *fingerprint.SimilarityService) *Service {
	return &Service{
		tagWriter:  tagWriter,
		library:    library,
		config:     config,
		queue:      queue,
		jobService: jobService,
		similarity: similarity,
	}
}

// AddQueueItem adds a track to the duplicates queue.
func (s *Service) AddQueueItem(track *music.Track, qType music.QueueItemType, metadata map[string]string) error {
	if track.ID == "" {
		return fmt.Errorf("track ID cannot be empty")
	}
	return s.queue.Add(music.QueueItem{
		ID:        track.ID,
		Type:      qType,
		Track:     track,
		Timestamp: time.Now(),
		Metadata:  metadata,
	})
}

// GetQueueItems returns all items in the duplicates queue.
func (s *Service) GetQueueItems() map[string]music.QueueItem {
	return s.queue.GetAll()
}

// QueueCount returns the number of pending duplicate queue items.
func (s *Service) QueueCount() int {
	count := 0
	for _, item := range s.queue.GetAll() {
		if item.Type == DuplicateFPExact || item.Type == DuplicateFPFuzzy {
			count++
		}
	}
	return count
}

// ClearQueue removes all items from the duplicates queue.
func (s *Service) ClearQueue() error {
	return s.queue.Clear()
}

// GetGroupedByKey returns queue items grouped by group_key (the primary track's ID).
// Each group is enriched with display-ready fields for templates.
func (s *Service) GetGroupedByKey() []DuplicateGroup {
	items := s.queue.GetAll()
	raw := make(map[string][]music.QueueItem)
	for _, item := range items {
		if key, ok := item.Metadata["group_key"]; ok {
			raw[key] = append(raw[key], item)
		}
	}

	groups := make([]DuplicateGroup, 0, len(raw))
	for key, groupItems := range raw {
		// Best similarity across the group (max value stored in metadata).
		bestSim := ""
		matchType := ""
		for _, it := range groupItems {
			if s := it.Metadata["similarity"]; s != "" && (bestSim == "" || s > bestSim) {
				bestSim = s
			}
			if mt := it.Metadata["match_type"]; mt != "" {
				matchType = mt
			}
		}
		groups = append(groups, DuplicateGroup{
			Key:        key,
			Items:      groupItems,
			PrimaryID:  key,
			Similarity: bestSim,
			MatchType:  matchType,
		})
	}
	return groups
}

// ProcessQueueItem executes an action on a single queue item.
//
// Actions:
//   - keep_both      — remove from queue, keep both files untouched
//   - delete_this    — delete this (duplicate) track file + DB record, keep primary
//   - delete_primary — delete the primary track file + DB record, keep this one;
//     also removes all sibling queue items that referenced the same primary
func (s *Service) ProcessQueueItem(ctx context.Context, itemID string, action string) error {
	item, err := s.queue.GetByID(itemID)
	if err != nil {
		return fmt.Errorf("queue item not found: %w", err)
	}
	if item.Track == nil {
		return fmt.Errorf("queue item has no associated track")
	}

	switch action {
	case "keep_both":
		slog.Info("Keeping both tracks", "trackID", item.Track.ID)
		// No file operations — just remove from queue.

	case "delete_this":
		if err := os.Remove(item.Track.Path); err != nil {
			slog.Warn("Failed to remove duplicate file", "path", item.Track.Path, "error", err)
		}
		if err := s.library.DeleteTrack(ctx, item.Track.ID); err != nil {
			return fmt.Errorf("failed to delete track from library: %w", err)
		}
		slog.Info("Deleted duplicate track", "trackID", item.Track.ID)

	case "delete_primary":
		primaryID := item.Metadata["primary_id"]
		if primaryID == "" {
			return fmt.Errorf("queue item has no primary_id in metadata")
		}
		primary, err := s.library.GetTrack(ctx, primaryID)
		if err != nil {
			return fmt.Errorf("failed to get primary track: %w", err)
		}
		if err := os.Remove(primary.Path); err != nil {
			slog.Warn("Failed to remove primary file", "path", primary.Path, "error", err)
		}
		if err := s.library.DeleteTrack(ctx, primaryID); err != nil {
			return fmt.Errorf("failed to delete primary track from library: %w", err)
		}
		slog.Info("Deleted primary track, keeping duplicate", "primaryID", primaryID, "keptID", item.Track.ID)

		// Remove all sibling queue items that pointed to the same primary,
		// since the primary no longer exists.
		for id, sibling := range s.queue.GetAll() {
			if sibling.Metadata["primary_id"] == primaryID && id != itemID {
				if err := s.queue.Remove(id); err != nil {
					slog.Warn("Failed to remove sibling queue item", "id", id, "error", err)
				}
			}
		}

	default:
		return fmt.Errorf("unknown action %q", action)
	}

	return s.queue.Remove(itemID)
}

// ProcessQueueGroup executes a bulk action on all items in a fingerprint group.
//
// Actions:
//   - keep_all         — keep every file, clear the group from the queue
//   - delete_duplicates — delete all non-primary tracks, keep the primary
func (s *Service) ProcessQueueGroup(ctx context.Context, groupKey string, action string) error {
	var groupItems []music.QueueItem
	for _, item := range s.queue.GetAll() {
		if item.Metadata["group_key"] == groupKey {
			groupItems = append(groupItems, item)
		}
	}
	if len(groupItems) == 0 {
		return fmt.Errorf("no items found in group %q", groupKey)
	}

	// Map group actions to per-item actions.
	perItemAction := ""
	switch action {
	case "keep_all":
		perItemAction = "keep_both"
	case "delete_duplicates":
		perItemAction = "delete_this"
	default:
		return fmt.Errorf("unknown group action %q", action)
	}

	for _, item := range groupItems {
		if err := s.ProcessQueueItem(ctx, item.ID, perItemAction); err != nil {
			slog.Warn("Failed to process group item", "itemID", item.ID, "action", perItemAction, "error", err)
		}
	}
	return nil
}

// StartDuplicatesAnalysis kicks off the background duplicate-detection job.
func (s *Service) StartDuplicatesAnalysis(ctx context.Context, params map[string]any) (string, error) {
	if params == nil {
		params = map[string]any{}
	}
	if _, ok := params["fp_exact_thresh"]; !ok {
		params["fp_exact_thresh"] = 0.95
	}
	if _, ok := params["fp_fuzzy_thresh"]; !ok {
		params["fp_fuzzy_thresh"] = 0.75
	}
	jobID, err := s.jobService.StartJob("analyze_duplicates", "Analyze Library Duplicates (FP)", params)
	if err != nil {
		return "", fmt.Errorf("failed to start job: %w", err)
	}
	slog.Info("Duplicates analysis job started", "jobID", jobID, "params", params)
	return jobID, nil
}

// GetQueueItem returns a single queue item by ID.
func (s *Service) GetQueueItem(id string) (music.QueueItem, error) {
	return s.queue.GetByID(id)
}
