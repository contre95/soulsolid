package metadata

import (
	"context"
)

// AcoustIDLookupResult holds the outcome of an AcoustID fingerprint lookup, including
// the recording title/artist AcoustID resolved via MusicBrainz, when available.
type AcoustIDLookupResult struct {
	AcoustID string
	Title    string
	Artist   string
}

// AcoustIDProvider handles AcoustID lookups using chromaprint fingerprints
type ChromaprintAcoustID interface {
	GenerateChromaprint(ctx context.Context, filePath string) (string, int, error)
	CompareChromaprints(cp1, cp2 string) (float64, error)
	// LookupAcoustID looks up AcoustID using a chromaprint fingerprint and duration
	LookupAcoustID(ctx context.Context, chromaprint string, duration int) (AcoustIDLookupResult, error)
}
