package fingerprint

import (
	"context"
	"fmt"
	"strings"

	"github.com/contre95/soulsolid/src/features/config"
	"github.com/contre95/soulsolid/src/features/importing"
)

// Service implements FingerprintReader for audio fingerprinting
type Service struct {
	config *config.Manager
}

// NewFingerprintService creates a new fingerprint service
func NewFingerprintService(cfg *config.Manager) importing.FingerprintProvider {
	return &Service{
		config: cfg,
	}
}

// GenerateFingerprint generates a fingerprint for an audio file
func (s *Service) GenerateFingerprint(ctx context.Context, filePath string) (string, error) {
	// Try to use fpcalc for now since it handles audio decoding
	// In future, we can implement proper audio decoding for gochroma
	fingerprint, err := s.generateFingerprintWithFpcalc(ctx, filePath)
	if err != nil {
		// If fpcalc is not available, provide a helpful error message
		if strings.Contains(err.Error(), "fpcalc not found") {
			return "", fmt.Errorf("fpcalc not found. Please install chromaprint tools:\n" +
				"  Nix: nix-shell -p chromaprint\n" +
				"  Ubuntu/Debian: sudo apt-get install chromaprint-tools\n" +
				"  macOS: brew install chromaprint\n" +
				"  Or download from: https://acoustid.org/chromaprint")
		}
		return "", err
	}
	return fingerprint, nil
}

// generateFingerprintWithFpcalc generates fingerprint using fpcalc command
// TODO: In future, replace this with gochroma library for pure Go implementation
// The gochroma library requires raw audio data, so we'd need to add audio decoding
func (s *Service) generateFingerprintWithFpcalc(ctx context.Context, filePath string) (string, error) {
	fingerprint, _, err := Fpcalc(ctx, filePath)
	return fingerprint, err
}

// CompareFingerprints compares two fingerprints and returns a similarity score (0.0 to 1.0)
func (s *Service) CompareFingerprints(fp1, fp2 string) (float64, error) {
	// For now, use simple string comparison
	// In a full implementation, this would use proper audio fingerprint comparison
	if fp1 == fp2 {
		return 1.0, nil
	}
	return 0.0, nil
}
