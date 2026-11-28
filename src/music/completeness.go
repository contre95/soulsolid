package music

// Album attribute keys used by the album completeness feature. They are stored in the
// album_attributes table, so no schema change is needed.
const (
	AlbumAttrExpectedTracks        = "expected_track_count"
	AlbumAttrCompletenessSource    = "completeness_source"
	AlbumAttrCompletenessSourceURL = "completeness_source_url"
	AlbumAttrCompletenessCheckedAt = "completeness_checked_at"
)

// CompletenessSourceManual marks an expected track count entered by the user.
const CompletenessSourceManual = "manual"

// AlbumCompletenessRow is a lightweight, read-only view of an album that is missing tracks.
type AlbumCompletenessRow struct {
	ID            string
	Title         string
	Artists       string // album artist names, comma separated
	LocalCount    int
	ExpectedCount int
	Source        string
	CheckedAt     string
	CoverTrackID  string // first track of the album, used to serve the artwork
}

// Missing returns how many tracks the library lacks.
func (r *AlbumCompletenessRow) Missing() int {
	return max(r.ExpectedCount-r.LocalCount, 0)
}
