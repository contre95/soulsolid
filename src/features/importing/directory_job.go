package importing

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/contre95/soulsolid/src/features/config"
	"github.com/contre95/soulsolid/src/music"
	"github.com/google/uuid"
)

// ImportAction represents the action to take for a track during import
type ImportAction int

const (
	SkipTrack ImportAction = iota
	ReplaceTrack
	QueueTrack
	ImportTrack
)

// generateTrackIDFromPath generates a deterministic UUID for a track from its file path
func generateTrackIDFromPath(path string) string {
	return uuid.NewSHA1(uuid.NameSpaceDNS, []byte(path)).String()
}

// DirectoryImportTask implements jobs.Task for directory imports.
type DirectoryImportTask struct {
	service *Service
}

// NewDirectoryImportTask creates a new DirectoryImportTask.
func NewDirectoryImportTask(service *Service) *DirectoryImportTask {
	return &DirectoryImportTask{service: service}
}

// MetadataKeys returns the required metadata keys for a directory import job.
func (e *DirectoryImportTask) MetadataKeys() []string {
	return []string{"path"}
}

// Execute runs the directory import logic.
func (e *DirectoryImportTask) Execute(ctx context.Context, job *music.Job, progressUpdater func(int, string)) (map[string]any, error) {
	path := job.Metadata["path"].(string)

	stats, err := e.runDirectoryImport(ctx, path, progressUpdater, job.Logger, job)
	if err != nil {
		return nil, fmt.Errorf("failed to import directory: %w", err)
	}

	// Check if context was cancelled
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	totalProcessed := stats.TracksImported + stats.Skipped + stats.Queued + stats.Errors
	finalMessage := fmt.Sprintf("Directory import finished. Processed %d tracks (%d imported, %d queued, %d skipped, %d errors).",
		totalProcessed, stats.TracksImported, stats.Queued, stats.Skipped, stats.Errors)
	job.Logger.Info(finalMessage)

	// Determine job status - consider skips and queued as successful
	if stats.TracksImported == 0 && stats.Skipped == 0 && stats.Queued == 0 && stats.Errors > 0 {
		// Complete failure - no tracks processed successfully
		slog.Warn("No tracks were successfully processed", "stats", stats)
		return map[string]any{"stats": stats, "msg": finalMessage}, errors.New("No tracks were successfully processed")
	} else if stats.Errors > 0 {
		// Partial success - some failures occurred
		slog.Warn("Some tracks failed to process", "stats", stats)
		return map[string]any{"stats": stats, "msg": finalMessage}, errors.New("Some tracks failed to process")
	}

	// Full success - all tracks processed without errors (including skips)
	return map[string]any{"stats": stats, "msg": finalMessage}, nil
}

// Cleanup does nothing for directory imports.
func (e *DirectoryImportTask) Cleanup(job *music.Job) error {
	return nil
}

// countSupportedFiles counts the number of supported audio files in a directory
func countSupportedFiles(pathToImport string) int {
	totalFiles := 0
	filepath.Walk(pathToImport, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			ext := strings.ToLower(filepath.Ext(path))
			if supportedExtensions[ext] {
				totalFiles++
			}
		}
		return nil
	})
	return totalFiles
}

// determineAction determines what action to take for a track based on config and duplicate
// tracks. A queued track may carry more than one type at once (e.g. Duplicate + MissingMetadata).
func determineAction(track *music.Track, duplicateTrack *music.Track, config config.Import, logger *slog.Logger) (ImportAction, []music.QueueItemType, map[string]string) {
	metadata := map[string]string{}
	var types []music.QueueItemType

	// Permitted missing fields were already defaulted before duplicate detection; any field
	// still missing here is not permitted by config, so flag the track as missing metadata.
	amm := config.AllowMissingMetadata
	missingErr := track.ValidateRequiredMetadata(amm.Artist, amm.Album, amm.Title, amm.Year, amm.Genre)
	if missingErr != nil {
		types = append(types, MissingMetadata)
		metadata["error"] = missingErr.Error()
	}

	if duplicateTrack != nil {
		// Fast paths apply only when force-queuing is off. "skip" discards the duplicate
		// regardless of metadata (nothing enters the library). "replace" only auto-replaces a
		// complete track; an incomplete one is queued instead so it can't silently overwrite
		// the library with missing metadata.
		if !config.AlwaysQueue {
			switch config.Duplicates {
			case "skip":
				logger.Info("Service.runDirectoryImport: Decided to skip duplicate track", "reason", "skip enabled for duplicates", "duplicate_path", duplicateTrack.Path, "title", track.Title)
				return SkipTrack, nil, nil
			case "replace":
				if missingErr == nil {
					logger.Info("Service.runDirectoryImport: Decided to replace duplicate track", "reason", "replace enabled for duplicates", "duplicate_path", duplicateTrack.Path, "new_path", track.Path, "title", track.Title)
					return ReplaceTrack, nil, nil
				}
				logger.Info("Service.runDirectoryImport: queuing duplicate instead of replacing", "reason", "missing required metadata", "duplicate_path", duplicateTrack.Path, "title", track.Title)
			}
		}
		types = append(types, Duplicate)
		metadata["duplicate_path"] = duplicateTrack.Path
		logger.Info("Service.runDirectoryImport: Decided to queue as duplicate", "duplicate_path", duplicateTrack.Path, "title", track.Title, "types", types)
		return QueueTrack, types, metadata
	}

	// No duplicate.
	if missingErr != nil {
		logger.Info("Service.runDirectoryImport: track queued as missing metadata", "title", track.Title, "error", missingErr.Error())
		return QueueTrack, types, metadata
	}
	if config.AlwaysQueue {
		logger.Info("Service.runDirectoryImport: track queued for manual review", "reason", "always_queue enabled", "title", track.Title)
		return QueueTrack, []music.QueueItemType{ManualReview}, nil
	}
	logger.Info("Service.runDirectoryImport: Decided to import track", "reason", "track didn't exists in the library", "title", track.Title, "artist", track.Artists)
	return ImportTrack, nil, nil
}

// addTrackToQueue adds a track to the queue
func (e *DirectoryImportTask) addTrackToQueue(track *music.Track, queueTypes []music.QueueItemType, jobID string, duplicateTrack *music.Track, logger *slog.Logger, metadata map[string]string) error {
	if track == nil {
		return fmt.Errorf("track cannot be nil")
	}
	if duplicateTrack != nil {
		track.ID = duplicateTrack.ID
	}
	if track.ID == "" {
		return fmt.Errorf("track ID cannot be empty")
	}

	item := music.QueueItem{
		ID:        track.ID,
		Types:     queueTypes,
		Track:     track,
		Timestamp: time.Now(),
		JobID:     jobID,
		Metadata:  metadata,
	}
	err := e.service.queue.Add(item)
	if err != nil {
		if errors.Is(err, music.ErrTrackInTheQueueAlready) {
			logger.Warn("Service.runDirectoryImport: track already exists in queue", "error", err, "trackID", track.ID)
		} else {
			logger.Error("Service.runDirectoryImport: failed to add track to queue", "error", err, "trackID", track.ID)
		}
	}
	return nil
}

// importFile moves or copies a track file to the library location and returns the new path
func (e *DirectoryImportTask) importFile(ctx context.Context, track *music.Track, moveFiles bool, logger *slog.Logger) (string, error) {
	var newPath string
	var err error
	if moveFiles {
		newPath, err = e.service.fileManager.MoveTrackToLibrary(ctx, track)
	} else {
		newPath, err = e.service.fileManager.CopyTrackToLibrary(ctx, track)
	}
	if err != nil {
		logger.Error("Service.runDirectoryImport: could not organize track", "error", err)
		return "", err
	}
	return newPath, nil
}

func (e *DirectoryImportTask) findDuplicateTrack(ctx context.Context, trackToImport *music.Track, fingerprint string, logger *slog.Logger) (*music.Track, error) {
	trackID := music.GenerateTrackID(fingerprint)
	duplicateTrack, err := e.service.library.GetTrack(ctx, trackID)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			duplicateTrack = nil // Not found, not an error
		} else {
			logger.Error("Service.runDirectoryImport: error checking if track exists by ID", "error", err, "trackID", trackToImport.ID)
			return nil, err
		}
	}

	// Also check for duplicate track by library path to catch duplicates that have already been imported
	if duplicateTrack == nil {
		// Generate the library path that this track would get
		libraryPath, err := e.service.fileManager.GetLibraryPath(ctx, trackToImport)
		if err != nil {
			logger.Warn("Service.runDirectoryImport: failed to generate library path for duplicate check", "error", err, "title", trackToImport.Title)
			// Don't fail the import, just skip this check
		} else {
			// Check if a track with this library path already exists
			duplicateTrack, err = e.service.library.FindTrackByPath(ctx, libraryPath)
			if err != nil {
				if err.Error() == "sql: no rows in result set" {
					duplicateTrack = nil // Not found, not an error
				} else {
					logger.Error("Service.runDirectoryImport: error checking if track exists by library path", "error", err, "path", libraryPath)
					return nil, err
				}
			}
		}
	}
	return duplicateTrack, nil
}

func (e *DirectoryImportTask) runDirectoryImport(ctx context.Context, pathToImport string, progressUpdater func(int, string), logger *slog.Logger, job *music.Job) (ImportStats, error) {
	logger.Info("Service.runDirectoryImport: starting import", "path", pathToImport)
	var stats ImportStats
	moveFiles := e.service.config.Get().Import.Move
	config := e.service.config.Get().Import

	// Count total files first for progress tracking and logging purposes
	totalFiles := countSupportedFiles(pathToImport)
	logger.Info("Service.runDirectoryImport: found files to process", "total", totalFiles)

	processedFiles := 0
	err := filepath.Walk(pathToImport, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			logger.Error("Service.runDirectoryImport: could not walk root dir", "error", err)
			return err
		}
		if !info.IsDir() {
			ext := strings.ToLower(filepath.Ext(path))
			if !supportedExtensions[ext] {
				logger.Debug("Service.runDirectoryImport: skipping unsupported file", "path", path, "extension", ext)
				return nil // Skip unsupported files
			}

			logger.Info("Service.runDirectoryImport: processing file", "trackToImport", path)

			trackToImport, err := e.service.metadataReader.ReadFileTags(ctx, path)
			if err != nil || info.Size() == 0 {
				logger.Warn("Service.runDirectoryImport: could not read metadata from file", "path", path, "error", err)
				stats.Errors++
				// Create minimal track and add to queue.
				nullTrackForQueue := music.Track{}
				nullTrackForQueue.Title = path
				nullTrackForQueue.Path = path
				nullTrackForQueue.EnsureMetadataDefaults(true, true, true, true, true)
				nullTrackForQueue.ID = generateTrackIDFromPath(path) // ID generate for queue duplicates.
				if err := e.addTrackToQueue(&nullTrackForQueue, []music.QueueItemType{FailedImport}, job.ID, nil, logger, map[string]string{"error": err.Error()}); err != nil {
					logger.Error("Service.runDirectoryImport: failed to add metadata-failed track to queue", "error", err)
				}
				processedFiles++
				return nil
			}
			slog.Info("Read metadata from file", "path", path, "track", trackToImport)

			// Apply default metadata if configured to allow missing metadata
			amm := config.AllowMissingMetadata
			trackToImport.EnsureMetadataDefaults(amm.Artist, amm.Album, amm.Title, amm.Year, amm.Genre)

			// Set source data for local file
			trackToImport.MetadataSource = music.MetadataSource{
				Source:            "LocalFile",
				MetadataSourceURL: path,
			}

			fingerprint, err := e.service.fingerprintReader.GenerateFingerprint(ctx, path)
			if err != nil {
				logger.Warn("Service.runDirectoryImport: failed to generate fingerprint, falling back to metadata", "error", err, "trackToImport", path)
				stats.Errors++
				// Set track ID from path and add to queue for manual review
				trackToImport.ID = generateTrackIDFromPath(path)
				if err := e.addTrackToQueue(trackToImport, []music.QueueItemType{FailedImport}, job.ID, nil, logger, map[string]string{"error": err.Error()}); err != nil {
					logger.Error("Service.runDirectoryImport: failed to add fingerprint-failed track to queue", "error", err)
				}
				processedFiles++
				return nil
			}
			slog.Info("Generated fingerprint for track", "path", path, "track", trackToImport, "fingerprint", fingerprint[:15])
			slog.Debug("Generated fingerprint for track", "path", path, "track", trackToImport, "fingerprint", fingerprint)
			trackToImport.ChromaprintFingerprint = fingerprint
			trackToImport.ID = music.GenerateTrackID(fingerprint)
			slog.Info("Generated track id", "id", trackToImport.ID)
			duplicateTrack, err := e.findDuplicateTrack(ctx, trackToImport, fingerprint, logger)
			if err != nil {
				logger.Error("Service.runDirectoryImport: failed to find duplicate track", "error", err)
				stats.Errors++
				if err := e.addTrackToQueue(trackToImport, []music.QueueItemType{FailedImport}, job.ID, nil, logger, map[string]string{"error": err.Error()}); err != nil {
					logger.Error("Service.runDirectoryImport: failed to add database-error track to queue", "error", err)
				}
				processedFiles++
				return nil
			}
			var action ImportAction
			var queueTypes []music.QueueItemType
			var itemMetadata map[string]string
			action, queueTypes, itemMetadata = determineAction(trackToImport, duplicateTrack, config, logger)

			switch action {
			case SkipTrack:
				stats.Skipped++
				logger.Info("Service.runDirectoryImport: Skipping duplicate track", "reason", "track already exists", "duplicate_path", path, "title", trackToImport.Title, "color", "blue")
			case QueueTrack:
				if err := e.addTrackToQueue(trackToImport, queueTypes, job.ID, duplicateTrack, logger, itemMetadata); err != nil {
					stats.Errors++
				} else {
					stats.Queued++
					logger.Info("Service.runDirectoryImport: track queued as duplicate", "reason", "duplicate track found", "duplicate_path", path, "title", trackToImport.Title, "color", "violet")
				}
			case ReplaceTrack:
				if err := e.service.replaceTrack(ctx, trackToImport, duplicateTrack, moveFiles, logger); err != nil {
					logger.Error("Service.runDirectoryImport: failed to replace track", "error", err)
					stats.Errors++
					// Add failed track to queue for manual review
					if err := e.addTrackToQueue(trackToImport, []music.QueueItemType{FailedImport}, job.ID, duplicateTrack, logger, map[string]string{"error": err.Error()}); err != nil {
						logger.Error("Service.runDirectoryImport: failed to add failed replace track to queue", "error", err)
					}
				} else {
					stats.TracksImported++
					logger.Info("Service.runDirectoryImport: duplicate track replaced", "title", trackToImport.Title, "color", "orange")
				}
			case ImportTrack:
				// determineAction already validated required metadata and applied any
				// permitted fallback defaults, so the track is ready to import here.
				if err := e.service.importTrack(ctx, trackToImport, moveFiles, logger); err != nil {
					logger.Error("Service.runDirectoryImport: failed to import track", "error", err, "title", trackToImport.Title, "path", trackToImport.Path)
					stats.Errors++
					// Add failed track to queue for manual review
					if err := e.addTrackToQueue(trackToImport, []music.QueueItemType{FailedImport}, job.ID, nil, logger, map[string]string{"error": err.Error()}); err != nil {
						logger.Error("Service.runDirectoryImport: failed to add failed import track to queue", "error", err)
					}
				} else {
					stats.TracksImported++
					logger.Info("Service.runDirectoryImport: Track Imported", "title", trackToImport.Title, "color", "green")
				}
			}

			// Update progress after processing each file
			processedFiles++
			if progressUpdater != nil && totalFiles > 0 {
				progress := min((processedFiles*100)/totalFiles, 100)
				progressUpdater(progress, fmt.Sprintf("Processed: %s", filepath.Base(path)))
			}
		}
		return nil
	})

	if progressUpdater != nil && err == nil && totalFiles > 0 {
		progressUpdater(100, "Import completed")
	}

	return stats, err
}
