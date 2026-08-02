package merge

// Kind identifies which metadata dimension a merge group concerns.
type Kind string

const (
	KindArtist Kind = "artist"
	KindAlbum  Kind = "album"
	KindGenre  Kind = "genre"
	KindTrack  Kind = "track"
)

// Variant is one member of a merge group: an existing entity (artist/album/track) or a raw genre value.
type Variant struct {
	// ID is the entity ID for artists/albums/tracks; for genres it equals Value.
	ID string
	// Value is the display string (artist name / album title / genre / track title).
	Value string
	// Sub is an optional secondary line (e.g. an album's primary artist or a track's file path).
	Sub string
}

// Group is a set of variants that share a merge key and can be merged into one.
type Group struct {
	// Key is the shared key: a normalized name for artists/albums/genres, an AcoustID for tracks.
	Key string
	// Canonical is the suggested member to keep: a Value for artists/albums/genres, an ID for
	// tracks (track titles in a group are usually identical, so the ID disambiguates).
	Canonical string
	// Variants are the members of the group, sorted by Value.
	Variants []Variant
}
