package albumcompleteness

import (
	"context"
	"errors"
	"strconv"

	"github.com/contre95/soulsolid/src/features/metadata"
	"github.com/contre95/soulsolid/src/music"
)

var (
	// ErrAlbumNotFound is returned when the album does not exist in the library.
	ErrAlbumNotFound = errors.New("album not found")
	// ErrNoMatch is returned when no provider knows the album's official track count.
	ErrNoMatch = errors.New("no matching release found at the metadata providers")
)

// Status tells whether the library holds every track of an album.
type Status string

const (
	StatusComplete   Status = "complete"
	StatusIncomplete Status = "incomplete"
	StatusUnknown    Status = "unknown" // no expected track count yet
)

// AlbumCompleteness compares the tracks held locally with the album's official track count.
type AlbumCompleteness struct {
	AlbumID       string
	LocalCount    int
	ExpectedCount int // 0 when unknown
	Missing       int
	Status        Status
	Source        string // provider name, or "manual"
	SourceURL     string
	CheckedAt     string // RFC3339, empty when never checked
	Manual        bool
}

// LibraryRepo is the subset of the library repository this feature needs. It is satisfied by
// *database.SqliteLibrary.
type LibraryRepo interface {
	GetAlbum(ctx context.Context, id string) (*music.Album, error)
	GetAlbumSummaries(ctx context.Context, limit, offset int, filter *music.AlbumFilter) ([]*music.AlbumSummary, error)
	GetAlbumSummariesCount(ctx context.Context, filter *music.AlbumFilter) (int, error)
	GetTracksFilteredCount(ctx context.Context, filter *music.TrackFilter) (int, error)
	SetAlbumAttributes(ctx context.Context, albumID string, attrs map[string]string) error
	GetIncompleteAlbums(ctx context.Context, limit, offset int) ([]*music.AlbumCompletenessRow, error)
	GetIncompleteAlbumsCount(ctx context.Context) (int, error)
}

// MetadataService looks up an album's official release info. It is satisfied by
// *metadata.Service.
type MetadataService interface {
	FetchAlbumInfo(ctx context.Context, params metadata.AlbumSearchParams) (*metadata.AlbumInfo, error)
}

// computeStatus compares the local and expected track counts. Having more tracks than expected
// (bonus tracks, other editions) still counts as complete.
func computeStatus(local, expected int) (Status, int) {
	switch {
	case expected <= 0:
		return StatusUnknown, 0
	case local >= expected:
		return StatusComplete, 0
	default:
		return StatusIncomplete, expected - local
	}
}

// fromAttributes builds an AlbumCompleteness from an album's stored attributes.
func fromAttributes(albumID string, attrs map[string]string, local int) *AlbumCompleteness {
	expected, _ := strconv.Atoi(attrs[music.AlbumAttrExpectedTracks])
	status, missing := computeStatus(local, expected)
	source := attrs[music.AlbumAttrCompletenessSource]
	return &AlbumCompleteness{
		AlbumID:       albumID,
		LocalCount:    local,
		ExpectedCount: max(expected, 0),
		Missing:       missing,
		Status:        status,
		Source:        source,
		SourceURL:     attrs[music.AlbumAttrCompletenessSourceURL],
		CheckedAt:     attrs[music.AlbumAttrCompletenessCheckedAt],
		Manual:        source == music.CompletenessSourceManual,
	}
}
