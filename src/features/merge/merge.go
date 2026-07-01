package merge

// Kind identifies which metadata dimension a merge group concerns.
type Kind string

const (
	KindArtist Kind = "artist"
	KindAlbum  Kind = "album"
	KindGenre  Kind = "genre"
)

// Variant is one member of a merge group: an existing entity (artist/album) or a raw genre value.
type Variant struct {
	// ID is the entity ID for artists/albums; for genres it equals Value.
	ID string
	// Value is the display string (artist name / album title / genre).
	Value string
	// Sub is an optional secondary line (e.g. an album's primary artist); empty otherwise.
	Sub string
}

// Group is a set of variants whose names normalize to the same key and can be merged into one.
type Group struct {
	// Key is the shared normalized key (used only internally / for stable ordering).
	Key string
	// Canonical is the suggested canonical Value (a smart default the user can override).
	Canonical string
	// Variants are the members of the group, sorted by Value.
	Variants []Variant
}
