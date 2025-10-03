package importing

import (
	"context"
)

// FingerprintProvider handles audio fingerprinting for uniqely idetifying song within the sysmtem
type FingerprintProvider interface {
	GenerateFingerprint(ctx context.Context, filePath string) (string, error)
	CompareFingerprints(fp1, fp2 string) (float64, error)
}
