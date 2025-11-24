package importing

import (
	"context"
)

// ChromaprintProvider handles audio chromaprinting for uniqely idetifying song within the sysmtem.
// Chromaprinter doesn't query any other external db such as AcoustID for anything
type ChromaprintProvider interface {
	GenerateChromaprint(ctx context.Context, filePath string) (string, error)
	CompareChromaprints(cp1, cp2 string) (float64, error)
}
