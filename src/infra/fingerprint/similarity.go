package fingerprint

import (
	"encoding/base64"
	"math/bits"
)

// SimilarityService provides fingerprint similarity calculations.
type SimilarityService struct{}

// NewSimilarityService creates a new SimilarityService.
func NewSimilarityService() *SimilarityService {
	return &SimilarityService{}
}

// Hamming returns the similarity ratio (0-1) between two chromaprint fingerprints using Hamming distance.
// 1.0 = identical, 0.0 = completely different.
func (s *SimilarityService) Hamming(fp1, fp2 string) float64 {
	b1, err := base64.RawStdEncoding.DecodeString(fp1)
	if err != nil || len(b1) == 0 {
		return 0.0
	}
	b2, err := base64.RawStdEncoding.DecodeString(fp2)
	if err != nil || len(b2) == 0 {
		return 0.0
	}
	minLen := len(b1)
	if len(b2) < minLen {
		minLen = len(b2)
	}
	var dist int
	for i := 0; i < minLen; i++ {
		dist += int(bits.OnesCount(uint(b1[i] ^ b2[i])))
	}
	return 1.0 - float64(dist)/float64(minLen*8)
}
