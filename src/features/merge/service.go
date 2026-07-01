package merge

import (
	"context"
	"fmt"
	"log/slog"
	"sort"

	"github.com/contre95/soulsolid/src/music"
)

// Library is the subset of the library repository the merge feature needs. It is satisfied by
// *database.SqliteLibrary; defined here so the feature depends only on what it uses.
type Library interface {
	GetArtists(ctx context.Context) ([]*music.Artist, error)
	GetAlbums(ctx context.Context) ([]*music.Album, error)
	GetGenres(ctx context.Context) ([]string, error)
	GetTrack(ctx context.Context, id string) (*music.Track, error)
	GetTracksFilteredPaginated(ctx context.Context, limit, offset int, filter *music.TrackFilter) ([]*music.Track, error)
	MergeArtists(ctx context.Context, canonicalID string, mergedIDs []string) error
	MergeAlbums(ctx context.Context, canonicalID string, mergedIDs []string) error
	StandardizeGenre(ctx context.Context, canonical string, variants []string) ([]string, error)
}

// TagReader reads tags from a music file (subset of the shared infra implementation).
type TagReader interface {
	ReadFileTags(ctx context.Context, filePath string) (*music.Track, error)
}

// TagWriter writes tags into a music file (subset of the shared infra implementation).
type TagWriter interface {
	WriteFileTags(ctx context.Context, filePath string, track *music.Track) error
}

// Service detects and applies metadata merges (artists, albums, genres).
type Service struct {
	library    Library
	tagWriter  TagWriter
	tagReader  TagReader
	jobService music.JobService
}

// NewService creates a new merge service.
func NewService(lib Library, tagWriter TagWriter, tagReader TagReader, jobService music.JobService) *Service {
	return &Service{
		library:    lib,
		tagWriter:  tagWriter,
		tagReader:  tagReader,
		jobService: jobService,
	}
}

// FindArtistGroups returns groups of artists whose names normalize to the same key.
func (s *Service) FindArtistGroups(ctx context.Context) ([]Group, error) {
	artists, err := s.library.GetArtists(ctx)
	if err != nil {
		return nil, err
	}
	buckets := map[string][]Variant{}
	for _, a := range artists {
		if a == nil || a.Name == music.VariousArtistsName {
			continue
		}
		key := normalizeKey(a.Name)
		if key == "" {
			continue
		}
		buckets[key] = append(buckets[key], Variant{ID: a.ID, Value: a.Name})
	}
	return buildGroups(buckets), nil
}

// FindAlbumGroups returns groups of albums that share a normalized title AND primary artist.
func (s *Service) FindAlbumGroups(ctx context.Context) ([]Group, error) {
	albums, err := s.library.GetAlbums(ctx)
	if err != nil {
		return nil, err
	}
	buckets := map[string][]Variant{}
	for _, al := range albums {
		if al == nil {
			continue
		}
		titleKey := normalizeKey(al.Title)
		if titleKey == "" {
			continue
		}
		artistName := ""
		if len(al.Artists) > 0 && al.Artists[0].Artist != nil {
			artistName = al.Artists[0].Artist.Name
		}
		key := titleKey + "\x00" + normalizeKey(artistName)
		buckets[key] = append(buckets[key], Variant{ID: al.ID, Value: al.Title, Sub: artistName})
	}
	return buildGroups(buckets), nil
}

// FindGenreGroups returns groups of genre strings that normalize to the same key.
func (s *Service) FindGenreGroups(ctx context.Context) ([]Group, error) {
	genres, err := s.library.GetGenres(ctx)
	if err != nil {
		return nil, err
	}
	buckets := map[string][]Variant{}
	for _, g := range genres {
		key := normalizeKey(g)
		if key == "" {
			continue
		}
		buckets[key] = append(buckets[key], Variant{ID: g, Value: g})
	}
	return buildGroups(buckets), nil
}

// buildGroups keeps only buckets with two or more variants and attaches a smart canonical default.
func buildGroups(buckets map[string][]Variant) []Group {
	groups := make([]Group, 0)
	for key, variants := range buckets {
		if len(variants) < 2 {
			continue
		}
		sort.Slice(variants, func(i, j int) bool { return variants[i].Value < variants[j].Value })
		values := make([]string, len(variants))
		for i, v := range variants {
			values[i] = v.Value
		}
		groups = append(groups, Group{Key: key, Canonical: smartCanonical(values), Variants: variants})
	}
	sort.Slice(groups, func(i, j int) bool { return groups[i].Key < groups[j].Key })
	return groups
}

// StartMerge validates the selection and launches a background job that performs the merge in the
// database and rewrites the affected files' tags. For artists/albums canonical and members are
// entity IDs; for genres they are the raw genre strings.
func (s *Service) StartMerge(ctx context.Context, kind Kind, canonical string, members []string) (string, error) {
	switch kind {
	case KindArtist, KindAlbum, KindGenre:
	default:
		return "", fmt.Errorf("unknown merge kind %q", kind)
	}
	if canonical == "" {
		return "", fmt.Errorf("no canonical value selected")
	}
	if len(members) < 2 {
		return "", fmt.Errorf("a merge needs at least two members")
	}

	// Everything except the canonical one gets merged into it. The canonical must be selected.
	merged := make([]string, 0, len(members)-1)
	canonicalSelected := false
	for _, m := range members {
		if m == canonical {
			canonicalSelected = true
			continue
		}
		merged = append(merged, m)
	}
	if !canonicalSelected {
		return "", fmt.Errorf("the canonical value must be one of the selected items")
	}
	if len(merged) == 0 {
		return "", fmt.Errorf("nothing to merge into the canonical value")
	}

	jobID, err := s.jobService.StartJob("analyze_merge", fmt.Sprintf("Merge %ss", kind), map[string]any{
		"kind":      string(kind),
		"canonical": canonical,
		"merged":    merged,
	})
	if err != nil {
		return "", fmt.Errorf("failed to start merge job: %w", err)
	}
	slog.Info("merge job started", "kind", kind, "canonical", canonical, "merged", merged, "jobID", jobID)
	return jobID, nil
}

// applyMerge performs the database merge and rewrites file tags. It is invoked from the job task.
func (s *Service) applyMerge(ctx context.Context, job *music.Job, kind Kind, canonical string, merged []string, progress func(int, string)) (map[string]any, error) {
	var affectedIDs []string
	switch kind {
	case KindArtist:
		if err := s.library.MergeArtists(ctx, canonical, merged); err != nil {
			return nil, fmt.Errorf("failed to merge artists: %w", err)
		}
		ids, err := s.affectedTrackIDs(ctx, &music.TrackFilter{ArtistIDs: []string{canonical}})
		if err != nil {
			return nil, err
		}
		affectedIDs = ids
	case KindAlbum:
		if err := s.library.MergeAlbums(ctx, canonical, merged); err != nil {
			return nil, fmt.Errorf("failed to merge albums: %w", err)
		}
		ids, err := s.affectedTrackIDs(ctx, &music.TrackFilter{AlbumIDs: []string{canonical}})
		if err != nil {
			return nil, err
		}
		affectedIDs = ids
	case KindGenre:
		ids, err := s.library.StandardizeGenre(ctx, canonical, merged)
		if err != nil {
			return nil, fmt.Errorf("failed to standardize genre: %w", err)
		}
		affectedIDs = ids
	default:
		return nil, fmt.Errorf("unknown merge kind %q", kind)
	}

	updated, failed := 0, 0
	total := len(affectedIDs)
	for i, id := range affectedIDs {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		if total > 0 {
			progress((i*100)/total, fmt.Sprintf("Rewriting file tags %d/%d", i+1, total))
		}
		if err := s.rewriteTrackTags(ctx, kind, id); err != nil {
			job.Logger.Warn("failed to rewrite file tags", "trackID", id, "error", err, "color", "orange")
			failed++
			continue
		}
		updated++
	}

	msg := fmt.Sprintf("Merged %d %s(s) into 1; rewrote %d file(s), %d failed.", len(merged), kind, updated, failed)
	job.Logger.Info("merge completed", "kind", kind, "merged", len(merged), "filesUpdated", updated, "filesFailed", failed, "color", "green")
	progress(100, msg)
	return map[string]any{
		"kind":         string(kind),
		"mergedCount":  len(merged),
		"filesUpdated": updated,
		"filesFailed":  failed,
		"msg":          msg,
	}, nil
}

// affectedTrackIDs returns the IDs of every track matching the filter, paginating to bound memory.
func (s *Service) affectedTrackIDs(ctx context.Context, filter *music.TrackFilter) ([]string, error) {
	var ids []string
	const batch = 200
	for offset := 0; ; offset += batch {
		tracks, err := s.library.GetTracksFilteredPaginated(ctx, batch, offset, filter)
		if err != nil {
			return nil, err
		}
		for _, t := range tracks {
			ids = append(ids, t.ID)
		}
		if len(tracks) < batch {
			break
		}
	}
	return ids, nil
}

// rewriteTrackTags reads the track's current file tags, overrides only the merged dimension with
// the (now canonical) database values, and writes the tags back — preserving all other file data.
func (s *Service) rewriteTrackTags(ctx context.Context, kind Kind, trackID string) error {
	dbTrack, err := s.library.GetTrack(ctx, trackID)
	if err != nil {
		return err
	}
	if dbTrack == nil {
		return fmt.Errorf("track not found: %s", trackID)
	}
	fileTrack, err := s.tagReader.ReadFileTags(ctx, dbTrack.Path)
	if err != nil {
		return fmt.Errorf("read tags: %w", err)
	}
	fileTrack.ID = dbTrack.ID
	fileTrack.Path = dbTrack.Path
	switch kind {
	case KindArtist:
		fileTrack.Artists = dbTrack.Artists
		if dbTrack.Album != nil {
			fileTrack.Album = dbTrack.Album
		}
	case KindAlbum:
		fileTrack.Album = dbTrack.Album
	case KindGenre:
		fileTrack.Metadata.Genre = dbTrack.Metadata.Genre
	}
	return s.tagWriter.WriteFileTags(ctx, fileTrack.Path, fileTrack)
}
