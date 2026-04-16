package duplicates

import "github.com/contre95/soulsolid/src/music"

const (
	DuplicateFPExact music.QueueItemType = "duplicate_fp_exact"
	DuplicateFPFuzzy music.QueueItemType = "duplicate_fp_fuzzy"
)

// DuplicateGroup holds a set of queue items that share the same fingerprint group,
// enriched with display-ready fields for templates.
type DuplicateGroup struct {
	Key        string            // primary track ID, used as URL-safe group key
	Items      []music.QueueItem // all non-primary tracks in this group
	PrimaryID  string
	Similarity string // formatted as "97.3" (already × 100), append "%" in template
	MatchType  string // "exact" or "fuzzy"
}
