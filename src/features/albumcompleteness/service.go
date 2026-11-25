package albumcompleteness

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/contre95/soulsolid/src/features/metadata"
	"github.com/contre95/soulsolid/src/music"
)

// defaultThrottle spaces out provider lookups during a job; MusicBrainz allows one request per second.
const defaultThrottle = 1100 * time.Millisecond

// Service checks albums against the official track count of the metadata providers.
type Service struct {
	repo       LibraryRepo
	metadata   MetadataService
	jobService music.JobService
	throttle   time.Duration
}

// NewService creates a new album completeness service.
func NewService(repo LibraryRepo, metadata MetadataService, jobService music.JobService) *Service {
	return &Service{
		repo:       repo,
		metadata:   metadata,
		jobService: jobService,
		throttle:   defaultThrottle,
	}
}

// GetAlbumCompleteness returns the stored completeness of an album without contacting any provider.
func (s *Service) GetAlbumCompleteness(ctx context.Context, albumID string) (*AlbumCompleteness, error) {
	album, err := s.repo.GetAlbum(ctx, albumID)
	if err != nil {
		return nil, fmt.Errorf("failed to get album: %w", err)
	}
	if album == nil {
		return nil, ErrAlbumNotFound
	}
	local, err := s.localCount(ctx, albumID)
	if err != nil {
		return nil, err
	}
	return fromAttributes(albumID, album.Attributes, local), nil
}

// CheckAlbum asks the providers for the album's official track count and stores it. A manually
// entered count is replaced, since this is an explicit request. It returns ErrNoMatch when no
// provider knows the album.
func (s *Service) CheckAlbum(ctx context.Context, albumID string) (*AlbumCompleteness, error) {
	album, err := s.repo.GetAlbum(ctx, albumID)
	if err != nil {
		return nil, fmt.Errorf("failed to get album: %w", err)
	}
	if album == nil {
		return nil, ErrAlbumNotFound
	}
	local, err := s.localCount(ctx, albumID)
	if err != nil {
		return nil, err
	}
	return s.checkAlbum(ctx, album, local)
}

// checkAlbum does the provider lookup for an already loaded album.
func (s *Service) checkAlbum(ctx context.Context, album *music.Album, local int) (*AlbumCompleteness, error) {
	info, err := s.metadata.FetchAlbumInfo(ctx, searchParams(album))
	if err != nil {
		return nil, err
	}
	checkedAt := time.Now().UTC().Format(time.RFC3339)
	if info == nil {
		// Remember the attempt so the job does not retry the album on every run.
		if err := s.repo.SetAlbumAttributes(ctx, album.ID, map[string]string{
			music.AlbumAttrCompletenessCheckedAt: checkedAt,
		}); err != nil {
			return nil, fmt.Errorf("failed to store check time: %w", err)
		}
		return nil, ErrNoMatch
	}

	attrs := map[string]string{
		music.AlbumAttrExpectedTracks:        strconv.Itoa(info.TotalTracks),
		music.AlbumAttrCompletenessSource:    info.Source,
		music.AlbumAttrCompletenessSourceURL: info.SourceURL,
		music.AlbumAttrCompletenessCheckedAt: checkedAt,
	}
	if err := s.repo.SetAlbumAttributes(ctx, album.ID, attrs); err != nil {
		return nil, fmt.Errorf("failed to store completeness: %w", err)
	}
	slog.Info("Album completeness checked", "albumID", album.ID, "title", album.Title,
		"local", local, "expected", info.TotalTracks, "source", info.Source)
	return fromAttributes(album.ID, attrs, local), nil
}

// SetExpectedTracks stores a manually entered track count, for editions the providers get wrong.
func (s *Service) SetExpectedTracks(ctx context.Context, albumID string, expected int) (*AlbumCompleteness, error) {
	if expected < 1 || expected > 999 {
		return nil, fmt.Errorf("expected track count must be between 1 and 999")
	}
	album, err := s.repo.GetAlbum(ctx, albumID)
	if err != nil {
		return nil, fmt.Errorf("failed to get album: %w", err)
	}
	if album == nil {
		return nil, ErrAlbumNotFound
	}
	local, err := s.localCount(ctx, albumID)
	if err != nil {
		return nil, err
	}
	attrs := map[string]string{
		music.AlbumAttrExpectedTracks:        strconv.Itoa(expected),
		music.AlbumAttrCompletenessSource:    music.CompletenessSourceManual,
		music.AlbumAttrCompletenessSourceURL: "",
		music.AlbumAttrCompletenessCheckedAt: time.Now().UTC().Format(time.RFC3339),
	}
	if err := s.repo.SetAlbumAttributes(ctx, albumID, attrs); err != nil {
		return nil, fmt.Errorf("failed to store expected track count: %w", err)
	}
	return fromAttributes(albumID, attrs, local), nil
}

// GetIncompleteAlbums returns one page of albums missing tracks and the total number of them.
func (s *Service) GetIncompleteAlbums(ctx context.Context, limit, offset int) ([]*music.AlbumCompletenessRow, int, error) {
	count, err := s.repo.GetIncompleteAlbumsCount(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count incomplete albums: %w", err)
	}
	rows, err := s.repo.GetIncompleteAlbums(ctx, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list incomplete albums: %w", err)
	}
	return rows, count, nil
}

// StartAnalysis starts the background job that checks every album. Unless force is set, albums
// that were already checked and albums with a manual count are skipped.
func (s *Service) StartAnalysis(ctx context.Context, force bool) (string, error) {
	jobID, err := music.StartTypedJob(s.jobService, "analyze_completeness", "Album Completeness Analysis", CompletenessParams{Force: force})
	if err != nil {
		return "", fmt.Errorf("failed to start album completeness analysis: %w", err)
	}
	slog.Info("Album completeness analysis started", "jobID", jobID, "force", force)
	return jobID, nil
}

func (s *Service) localCount(ctx context.Context, albumID string) (int, error) {
	count, err := s.repo.GetTracksFilteredCount(ctx, &music.TrackFilter{AlbumIDs: []string{albumID}})
	if err != nil {
		return 0, fmt.Errorf("failed to count album tracks: %w", err)
	}
	return count, nil
}

// searchParams builds the provider lookup for an album, using its first named album artist.
// Providers file compilations under "Various Artists" too, so it is a valid artist to match on.
func searchParams(album *music.Album) metadata.AlbumSearchParams {
	params := metadata.AlbumSearchParams{
		Album:          album.Title,
		Barcode:        strings.TrimSpace(album.Barcode),
		ReleaseGroupID: strings.TrimSpace(album.ReleaseGroupID),
	}
	for _, ar := range album.Artists {
		if ar.Artist != nil && strings.TrimSpace(ar.Artist.Name) != "" {
			params.AlbumArtist = ar.Artist.Name
			break
		}
	}
	return params
}
